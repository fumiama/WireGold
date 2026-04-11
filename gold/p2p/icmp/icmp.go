package icmp

import (
	"errors"
	"net"
	"net/netip"
	"os"
	"sync"
	"sync/atomic"

	"github.com/RomiChan/syncx"
	"github.com/fumiama/WireGold/config"
	"github.com/fumiama/WireGold/gold/p2p"
	"github.com/fumiama/orbyte/pbuf"
	"github.com/sirupsen/logrus"

	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
	"golang.org/x/net/ipv6"
)

var (
	ErrInvalidBodyType = errors.New("invalid body type")
)

var (
	echoid = os.Getpid()
)

// peerState holds per-peer ICMP echo state within a Conn.
type peerState struct {
	id      int
	seq     atomic.Uintptr
	seqpool *sync.Pool
}

func newPeerState() *peerState {
	ps := &peerState{}
	ps.seqpool = &sync.Pool{
		New: func() any {
			return int(ps.seq.Add(1))
		},
	}
	return ps
}

type EndPoint netip.Addr

func (ep *EndPoint) String() string {
	return (*netip.Addr)(ep).String()
}

func (ep *EndPoint) Network() string {
	return "icmp"
}

func (ep *EndPoint) Equal(ep2 p2p.EndPoint) bool {
	if ep == nil || ep2 == nil {
		return ep == nil && ep2 == nil
	}
	ipep2, ok := ep2.(*EndPoint)
	if !ok {
		return false
	}
	ipep1 := ep
	return (*netip.Addr)(ipep1).Compare(*(*netip.Addr)(ipep2)) == 0
}

// network get ipv4/ipv6 info and choose different options.
func (ep *EndPoint) network() (string, *netip.Addr) {
	nw := "ip4:icmp"
	if (*netip.Addr)(ep).Is6() {
		nw = "ip6:ipv6-icmp"
	}
	return nw, (*netip.Addr)(ep)
}

func (ep *EndPoint) Listen() (p2p.Conn, error) {
	nw, addr := ep.network()
	conn, err := icmp.ListenPacket(nw, addr.String())
	if err != nil {
		return nil, err
	}
	return &Conn{inner: conn}, nil
}

type Conn struct {
	inner *icmp.PacketConn
	peers syncx.Map[netip.Addr, *peerState]
}

func (conn *Conn) getOrCreatePeerState(addr netip.Addr) *peerState {
	if ps, ok := conn.peers.Load(addr); ok {
		return ps
	}
	ps := newPeerState()
	actual, _ := conn.peers.LoadOrStore(addr, ps)
	return actual
}

func (conn *Conn) Close() error {
	return conn.inner.Close()
}

func (conn *Conn) String() string {
	return conn.inner.LocalAddr().String()
}

func (conn *Conn) LocalAddr() p2p.EndPoint {
	eps := conn.inner.LocalAddr().String()
	addr, err := netip.ParseAddrPort(eps)
	if err == nil {
		eps = addr.Addr().String()
	}
	ep, _ := NewEndpoint(eps)
	return ep
}

func (conn *Conn) ReadFromPeer(b []byte) (n int, ep p2p.EndPoint, err error) {
	buf := pbuf.NewBytes(8192)
	defer buf.ManualDestroy()
	var ipaddr netip.Addr
	buf.V(func(data []byte) {
		ok := false
		var msg *icmp.Message
		for !ok {
			var (
				cnt  int
				addr net.Addr
			)
			cnt, addr, err = conn.inner.ReadFrom(data)
			if err != nil {
				if config.ShowDebugLog {
					logrus.Debugln("[icmp] recv ReadFrom err:", err)
				}
				return
			}
			ipaddr, err = netip.ParseAddr(addr.String())
			if err != nil {
				if config.ShowDebugLog {
					logrus.Debugln("[icmp] recv ParseAddr err:", err, ", addr:", addr)
				}
				return
			}
			ep, err = NewEndpoint(ipaddr.String())
			if err != nil {
				if config.ShowDebugLog {
					logrus.Debugln("[icmp] recv NewEndpoint err:", err, ", addr:", addr)
				}
				return
			}
			proton := ipv4.ICMPTypeEcho.Protocol()
			if ipaddr.Is6() {
				proton = ipv6.ICMPTypeEchoRequest.Protocol()
			}

			msg, err = icmp.ParseMessage(proton, data[:cnt])
			if err != nil {
				if config.ShowDebugLog {
					logrus.Debugln("[icmp] recv ParseMessage err:", err, ", addr:", addr)
				}
				return
			}

			ok = msg.Type == ipv4.ICMPTypeEcho || msg.Type == ipv4.ICMPTypeEchoReply
			if ipaddr.Is6() {
				ok = msg.Type == ipv6.ICMPTypeEchoRequest || msg.Type == ipv6.ICMPTypeEchoReply
			}
			ok = ok && msg.Code == 1
			if config.ShowDebugLog {
				logrus.Debugln("[icmp] recv from", ipaddr, ", is valid:", ok)
			}
		}
		body, okk := msg.Body.(*icmp.Echo)
		if !okk {
			err = ErrInvalidBodyType
			return
		}
		if msg.Type == ipv4.ICMPTypeEcho || msg.Type == ipv6.ICMPTypeEchoRequest {
			ps := conn.getOrCreatePeerState(ipaddr)
			ps.id = body.ID
			ps.seq.Store(uintptr(body.Seq))
			ps.seqpool.Put(body.Seq)
		}
		n = copy(b, body.Data)
		if config.ShowDebugLog {
			logrus.Debugln("[icmp] recv", n, "bytes data from", ipaddr)
		}
	})
	return
}

func (conn *Conn) WriteToPeer(b []byte, ep p2p.EndPoint) (int, error) {
	icmpep, ok := ep.(*EndPoint)
	if !ok {
		return 0, p2p.ErrEndpointTypeMistatch
	}
	addr := (*netip.Addr)(icmpep)
	ps := conn.getOrCreatePeerState(*addr)
	seq := ps.seqpool.Get().(int)
	id := ps.id
	isrequest := id == 0
	if isrequest {
		id = echoid
	}
	var (
		ip  net.IP
		msg icmp.Message
	)
	if addr.Is4() {
		x := addr.As4()
		ip = x[:]
		msg = icmp.Message{
			Code: 1,
			Body: &icmp.Echo{
				ID:   id,
				Seq:  seq,
				Data: b,
			},
		}
		if isrequest {
			msg.Type = ipv4.ICMPTypeEcho
		} else {
			msg.Type = ipv4.ICMPTypeEchoReply
		}
	} else {
		x := addr.As16()
		ip = x[:]
		msg = icmp.Message{
			Code: 1,
			Body: &icmp.Echo{
				ID:   id,
				Seq:  seq,
				Data: b,
			},
		}
		if isrequest {
			msg.Type = ipv6.ICMPTypeEchoRequest
		} else {
			msg.Type = ipv6.ICMPTypeEchoReply
		}
	}
	buf := pbuf.NewBytes(8192)
	defer buf.ManualDestroy()
	var (
		data []byte
		err  error
		n    int
	)
	buf.V(func(bin []byte) {
		data, err = msg.Marshal(bin[:0])
		if err != nil {
			return
		}
		_, err = conn.inner.WriteTo(data, &net.IPAddr{
			IP:   ip,
			Zone: addr.Zone(),
		})
		if err == nil {
			n = len(b)
		}
	})
	return n, err
}

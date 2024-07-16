package tcp

import (
	"errors"
	"io"
	"net"
	"reflect"
	"runtime"
	"strconv"
	"time"

	"github.com/FloatTech/ttl"
	"github.com/fumiama/WireGold/gold/p2p"
	"github.com/fumiama/WireGold/helper"
	"github.com/sirupsen/logrus"
)

type EndPoint struct {
	addr         *net.TCPAddr
	dialtimeout  time.Duration
	peerstimeout time.Duration
	recvchansize int
}

func (ep *EndPoint) String() string {
	return ep.addr.String()
}

func (ep *EndPoint) Network() string {
	return ep.addr.Network()
}

func (ep *EndPoint) Euqal(ep2 p2p.EndPoint) bool {
	if ep == nil || ep2 == nil {
		return ep == nil && ep2 == nil
	}
	tcpep2, ok := ep2.(*EndPoint)
	if !ok {
		return false
	}
	tcpep1 := ep
	return tcpep1.addr.IP.Equal(tcpep2.addr.IP) &&
		tcpep1.addr.Port == tcpep2.addr.Port &&
		tcpep1.addr.Zone == tcpep2.addr.Zone
}

func (ep *EndPoint) Listen() (p2p.Conn, error) {
	lstn, err := net.ListenTCP(ep.addr.Network(), ep.addr)
	if err != nil {
		return nil, err
	}
	ep.addr = lstn.Addr().(*net.TCPAddr)
	peerstimeout := ep.peerstimeout
	if peerstimeout < time.Second {
		peerstimeout = time.Second * 5
	}
	chansz := ep.recvchansize
	if chansz < 32 {
		chansz = 32
	}
	conn := &Conn{
		addr:  ep,
		lstn:  lstn,
		peers: ttl.NewCache[string, *net.TCPConn](peerstimeout),
		recv:  make(chan *connrecv, chansz),
	}
	go conn.accept()
	return conn, nil
}

type connrecv struct {
	addr *EndPoint // cast from tcpconn.RemoteAddr()
	conn *net.TCPConn
	pckt packet
}

// Conn 伪装成无状态的有状态连接
type Conn struct {
	addr  *EndPoint
	lstn  *net.TCPListener
	peers *ttl.Cache[string, *net.TCPConn]
	recv  chan *connrecv
}

func (conn *Conn) accept() {
	for {
		tcpconn, err := conn.lstn.AcceptTCP()
		if err != nil {
			if errors.Is(err, net.ErrClosed) { // normal close
				logrus.Infoln("[tcp] accept of", conn.addr, "got closed")
				return
			}
			if conn.addr == nil || conn.lstn == nil || conn.peers == nil || conn.recv == nil {
				return
			}
			logrus.Warnln("[tcp] accept on", conn.addr, "err:", err)
			_ = conn.Close()
			newc, err := conn.addr.Listen()
			if err != nil {
				logrus.Warn("[tcp] re-listen on", conn.addr, "err:", err)
				return
			}
			*conn = *newc.(*Conn)
			logrus.Info("[tcp] re-listen on", conn.addr)
			continue
		}
		ep := newEndpoint(tcpconn.RemoteAddr().String(), &Config{
			DialTimeout:        conn.addr.dialtimeout,
			PeersTimeout:       conn.addr.peerstimeout,
			ReceiveChannelSize: conn.addr.recvchansize,
		})
		logrus.Debugln("[tcp] accept from", ep)
		conn.peers.Set(ep.String(), tcpconn)
		go conn.receive(ep)
	}
}

func (conn *Conn) receive(ep *EndPoint) {
	peerstimeout := ep.peerstimeout
	if peerstimeout < time.Second {
		peerstimeout = time.Second * 5
	}
	peerstimeout *= 2
	for {
		r := &connrecv{addr: ep}
		if conn.addr == nil || conn.lstn == nil || conn.peers == nil || conn.recv == nil {
			return
		}
		tcpconn := conn.peers.Get(ep.String())
		if tcpconn == nil {
			return
		}
		r.conn = tcpconn

		stopch := make(chan struct{})
		t := time.AfterFunc(peerstimeout, func() {
			stopch <- struct{}{}
		})

		var err error
		copych := make(chan struct{})
		go func() {
			_, err = io.Copy(&r.pckt, tcpconn)
			copych <- struct{}{}
		}()

		select {
		case <-stopch:
			logrus.Debugln("[tcp] recv from", ep, "timeout")
			continue
		case <-copych:
			t.Stop()
		}

		if err != nil {
			logrus.Debugln("[tcp] recv from", ep, "err:", err)
			return
		}
		logrus.Debugln("[tcp] dispatch packet from", ep, "typ", r.pckt.typ, "len", r.pckt.len)
		conn.recv <- r
	}
}

func (conn *Conn) Close() error {
	if conn.lstn != nil {
		_ = conn.lstn.Close()
	}
	if conn.peers != nil {
		conn.peers.Destroy()
	}
	if conn.recv != nil {
		close(conn.recv)
	}
	conn.addr = nil
	conn.lstn = nil
	conn.peers = nil
	conn.recv = nil
	return nil
}

func (conn *Conn) String() string {
	return conn.addr.String()
}

func (conn *Conn) LocalAddr() p2p.EndPoint {
	return conn.addr
}

func (conn *Conn) ReadFromPeer(b []byte) (int, p2p.EndPoint, error) {
	var p *connrecv
	for {
		p = <-conn.recv
		if p == nil {
			return 0, nil, net.ErrClosed
		}
		conn.peers.Set(p.addr.String(), p.conn)
		if p.pckt.typ == packetTypeNormal {
			break
		}
		defer helper.PutBytes(p.pckt.dat)
	}
	n := copy(b, p.pckt.dat)
	return n, p.addr, nil
}

func (conn *Conn) WriteToPeer(b []byte, ep p2p.EndPoint) (n int, err error) {
	tcpep, ok := ep.(*EndPoint)
	if !ok {
		return 0, p2p.ErrEndpointTypeMistatch
	}
	blen := len(b)
	if blen >= 65536 {
		return 0, errors.New("data size " + strconv.Itoa(blen) + " is too large")
	}
	tcpconn := conn.peers.Get(tcpep.String())
	if tcpconn == nil {
		dialtimeout := tcpep.dialtimeout
		if dialtimeout < time.Second {
			dialtimeout = time.Second
		}
		logrus.Debugln("[tcp] dial to", tcpep.addr, "timeout", dialtimeout)
		var cn net.Conn
		// must use another port to send because there's no exsiting conn
		cn, err = net.DialTimeout(tcpep.Network(), tcpep.addr.String(), dialtimeout)
		if err != nil {
			return
		}
		tcpconn, ok = cn.(*net.TCPConn)
		if !ok {
			return 0, errors.New("expect *net.TCPConn but got " + reflect.ValueOf(cn).Type().String())
		}
		runtime.SetFinalizer(tcpconn, func(t *net.TCPConn) {
			err := t.CloseWrite()
			if err != nil {
				logrus.Debugln("[tcp] close write from", t.LocalAddr(), "to", t.RemoteAddr(), "err:", err)
			} else {
				logrus.Debugln("[tcp] close write from", t.LocalAddr(), "to", t.RemoteAddr())
			}
		})
		logrus.Debugln("[tcp] dial to", tcpep.addr, "success, local:", tcpconn.LocalAddr())
		conn.peers.Set(tcpep.String(), tcpconn)
		go conn.receive(tcpep)
	} else {
		logrus.Debugln("[tcp] reuse tcpconn from", tcpconn.LocalAddr(), "to", tcpconn.RemoteAddr())
	}
	cnt, err := io.Copy(tcpconn, &packet{
		typ: packetTypeNormal,
		len: uint16(blen),
		dat: b,
	})
	return int(cnt) - 3, err
}

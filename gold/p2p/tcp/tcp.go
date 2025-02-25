package tcp

import (
	"errors"
	"io"
	"net"
	"reflect"
	"strconv"
	"sync"
	"time"

	"github.com/FloatTech/ttl"
	"github.com/sirupsen/logrus"

	"github.com/fumiama/WireGold/config"
	"github.com/fumiama/WireGold/gold/p2p"
	"github.com/fumiama/orbyte/pbuf"
)

type EndPoint struct {
	addr         *net.TCPAddr
	dialtimeout  time.Duration
	peerstimeout time.Duration
	keepinterval time.Duration
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
	if peerstimeout < time.Second*30 {
		peerstimeout = time.Second * 30
	}
	chansz := ep.recvchansize
	if chansz < 32 {
		chansz = 32
	}
	conn := &Conn{
		addr: ep,
		lstn: lstn,
		peers: ttl.NewCacheOn(peerstimeout, [4]func(string, *net.TCPConn){
			func(_ string, t *net.TCPConn) {
				_ = t.SetLinger(0)
				_ = t.SetNoDelay(true)
			}, nil, func(_ string, t *net.TCPConn) {
				err := t.CloseWrite()
				if config.ShowDebugLog {
					if err != nil {
						logrus.Debugln("[tcp] close write from", t.LocalAddr(), "to", t.RemoteAddr(), "err:", err)
					} else {
						logrus.Debugln("[tcp] close write from", t.LocalAddr(), "to", t.RemoteAddr())
					}
				}
			}, nil,
		}),
		recv: make(chan *connrecv, chansz),
		cplk: &sync.Mutex{},
		sblk: &sync.RWMutex{},
	}
	go conn.accept()
	return conn, nil
}

type connrecv struct {
	addr *EndPoint // cast from tcpconn.RemoteAddr()
	conn *net.TCPConn
	pckt packet
}

type subconn struct {
	cplk sync.Mutex
	last time.Time // last active time
	conn *net.TCPConn
}

// Conn 伪装成无状态的有状态连接
type Conn struct {
	addr   *EndPoint
	lstn   *net.TCPListener
	peers  *ttl.Cache[string, *net.TCPConn]
	recv   chan *connrecv
	cplk   *sync.Mutex
	sblk   *sync.RWMutex
	subs   []*subconn
	suberr bool
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
				logrus.Warnln("[tcp] re-listen on", conn.addr, "err:", err)
				return
			}
			*conn = *newc.(*Conn)
			logrus.Infoln("[tcp] re-listen on", conn.addr)
			continue
		}
		go conn.receive(tcpconn, false)
	}
}

func delsubs(i int, subs []*subconn) []*subconn {
	tcpconn := subs[i].conn
	err := tcpconn.CloseWrite()
	if config.ShowDebugLog {
		if err != nil {
			logrus.Debugln("[tcp] close sub write from", tcpconn.LocalAddr(), "to", tcpconn.RemoteAddr(), "err:", err)
		} else {
			logrus.Debugln("[tcp] close sub write from", tcpconn.LocalAddr(), "to", tcpconn.RemoteAddr())
		}
	}
	switch i {
	case 0:
		subs = subs[1:]
	case len(subs) - 1:
		subs = subs[:len(subs)-1]
	default:
		subs = append(subs[:i], subs[i+1:]...)
	}
	return subs
}

func (conn *Conn) receive(tcpconn *net.TCPConn, hasvalidated bool) {
	if conn.peers == nil {
		return
	}

	ep, _ := newEndpoint(tcpconn.RemoteAddr().String(), &Config{
		DialTimeout:        conn.addr.dialtimeout,
		PeersTimeout:       conn.addr.peerstimeout,
		KeepInterval:       conn.addr.keepinterval,
		ReceiveChannelSize: conn.addr.recvchansize,
	})

	issub, ok := false, false

	peerstimeout := conn.addr.peerstimeout
	if peerstimeout < time.Second*30 {
		peerstimeout = time.Second * 30
	}
	peerstimeout *= 2

	if !hasvalidated {
		issub, ok = isvalid(tcpconn, peerstimeout)
		if !ok {
			return
		}
		if config.ShowDebugLog {
			logrus.Debugln("[tcp] accept from", ep, "issub:", issub)
		}
		if issub {
			conn.sblk.Lock()
			conn.subs = append(conn.subs, &subconn{conn: tcpconn, last: time.Now()})
			conn.sblk.Unlock()
		} else {
			if conn.peers == nil {
				return
			}
			conn.peers.Set(ep.String(), tcpconn)
		}
	}

	if issub {
		defer func() {
			conn.sblk.Lock()
			subs := conn.subs
			for i, sub := range subs {
				if sub.conn == tcpconn {
					conn.subs = delsubs(i, conn.subs)
					break
				}
			}
			conn.sblk.Unlock()
		}()
	} else {
		if conn.peers == nil {
			return
		}
		defer conn.peers.Delete(ep.String())
	}

	go conn.keep(ep)

	for {
		r := &connrecv{addr: ep}
		if conn.addr == nil || conn.lstn == nil || conn.peers == nil || conn.recv == nil {
			return
		}
		if !issub {
			tcpconn = conn.peers.Get(ep.String())
			if tcpconn == nil {
				return
			}
		}
		r.conn = tcpconn

		t := time.NewTimer(peerstimeout)

		var err error
		copych := make(chan struct{})
		go func() {
			_, err = io.Copy(&r.pckt, tcpconn)
			copych <- struct{}{}
		}()

		select {
		case <-t.C:
			if config.ShowDebugLog {
				logrus.Debugln("[tcp] recv from", ep, "timeout")
			}
			_ = tcpconn.CloseRead()
			return
		case <-copych:
			t.Stop()
		}

		if conn.addr == nil || conn.lstn == nil || conn.peers == nil || conn.recv == nil {
			return
		}

		if err != nil {
			if config.ShowDebugLog {
				logrus.Debugln("[tcp] recv from", ep, "err:", err)
			}
			if errors.Is(err, net.ErrClosed) ||
				errors.Is(err, io.ErrClosedPipe) ||
				errors.Is(err, io.EOF) ||
				errors.Is(err, ErrInvalidMagic) {
				_ = tcpconn.CloseRead()
				return
			}
			continue
		}
		if r.pckt.typ >= packetTypeTop {
			if config.ShowDebugLog {
				logrus.Debugln("[tcp] close reading invalid conn from", ep, "typ", r.pckt.typ, "len", r.pckt.len)
			}
			_ = tcpconn.CloseRead()
			return
		}
		if config.ShowDebugLog {
			logrus.Debugln("[tcp] dispatch packet from", ep, "typ", r.pckt.typ, "len", r.pckt.len)
		}
		conn.recv <- r
	}
}

func (conn *Conn) keep(ep *EndPoint) {
	keepinterval := ep.keepinterval
	if keepinterval < time.Second*10 {
		keepinterval = time.Second * 10
	}
	t := time.NewTicker(keepinterval)
	defer t.Stop()
	for range t.C {
		if conn.addr == nil || conn.peers == nil {
			return
		}
		tcpconn := conn.peers.Get(ep.String())
		if tcpconn != nil {
			_, err := io.Copy(tcpconn, &packet{typ: packetTypeKeepAlive})
			if conn.addr == nil {
				return
			}
			if err != nil {
				logrus.Warnln("[tcp] keep main conn alive from", conn, "to", ep, "err:", err)
				conn.peers.Delete(ep.String())
			} else if config.ShowDebugLog {
				logrus.Debugln("[tcp] keep main conn alive from", conn, "to", ep)
			}
		}
		conn.sblk.RLock()
		subs := conn.subs
		for i, sub := range subs {
			if time.Since(sub.last) < keepinterval {
				if config.ShowDebugLog {
					logrus.Debugln("[tcp] skip to keep busy sub conn from", conn, "to", ep)
				}
				continue
			}
			_, err := io.Copy(sub.conn, &packet{typ: packetTypeSubKeepAlive})
			if conn.addr == nil {
				return
			}
			if err != nil {
				logrus.Warnln("[tcp] keep sub conn alive from", conn, "to", sub.conn.RemoteAddr(), "err:", err)
				conn.subs = delsubs(i, conn.subs) // del 1 link at once
				break
			}
			if config.ShowDebugLog {
				logrus.Debugln("[tcp] keep sub conn alive from", conn, "to", ep)
			}
		}
		conn.sblk.RUnlock()
	}
}

func (conn *Conn) Close() error {
	lstn := conn.lstn
	peers := conn.peers
	recv := conn.recv
	conn.addr = nil
	conn.lstn = nil
	conn.peers = nil
	conn.recv = nil

	if lstn != nil {
		_ = lstn.Close()
	}
	if peers != nil {
		peers.Destroy()
	}
	if recv != nil {
		close(recv)
	}

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
		if conn.peers == nil {
			return 0, nil, net.ErrClosed
		}
		conn.peers.Set(p.addr.String(), p.conn)
		if p.pckt.typ == packetTypeNormal {
			break
		}
	}
	n := copy(b, p.pckt.dat.Bytes())
	return n, p.addr, nil
}

// writeToPeer after acquiring lock
func (conn *Conn) writeToPeer(b []byte, tcpep *EndPoint, issub bool) (n int, err error) {
	retried := false
	ok := false
	var (
		tcpconn *net.TCPConn
		subc    *subconn
	)
RECONNECT:
	if issub {
		conn.sblk.RLock()
		for _, sub := range conn.subs {
			if sub.cplk.TryLock() {
				tcpconn = sub.conn
				subc = sub
				break
			}
		}
		conn.sblk.RUnlock()
	} else {
		tcpconn = conn.peers.Get(tcpep.String())
	}
	if tcpconn == nil {
		dialtimeout := tcpep.dialtimeout
		if dialtimeout < time.Second {
			dialtimeout = time.Second
		}
		if config.ShowDebugLog {
			logrus.Debugln("[tcp] dial to", tcpep.addr, "timeout", dialtimeout, "issub", issub)
		}
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
		pkt := &packet{}
		if issub {
			pkt.typ = packetTypeSubKeepAlive
		} else {
			pkt.typ = packetTypeKeepAlive
		}
		_, err = io.Copy(tcpconn, pkt)
		if err != nil {
			if config.ShowDebugLog {
				logrus.Debugln("[tcp] dial to", tcpep.addr, "issub", issub, "success, but write err:", err)
			}
			return 0, err
		}
		if config.ShowDebugLog {
			logrus.Debugln("[tcp] dial to", tcpep.addr, "success, local:", tcpconn.LocalAddr(), "issub", issub)
		}
		if !issub {
			conn.peers.Set(tcpep.String(), tcpconn)
		} else {
			conn.sblk.Lock()
			conn.subs = append(conn.subs, &subconn{conn: tcpconn, last: time.Now()})
			conn.sblk.Unlock()
		}
		go conn.receive(tcpconn, true)
	} else if config.ShowDebugLog {
		logrus.Debugln("[tcp] reuse tcpconn from", tcpconn.LocalAddr(), "to", tcpconn.RemoteAddr())
	}
	cnt, err := io.Copy(tcpconn, &packet{
		typ: packetTypeNormal,
		len: uint16(len(b)),
		dat: pbuf.ParseBytes(b...),
	})
	if err != nil {
		if subc == nil {
			conn.peers.Delete(tcpep.String())
		} else {
			conn.sblk.Lock()
			subs := conn.subs
			for i, sub := range subs {
				if sub == subc {
					conn.subs = delsubs(i, conn.subs)
					break
				}
			}
			conn.sblk.Unlock()
		}
		if !retried {
			logrus.Warnln("[tcp] reconnect due to write to", tcpconn.RemoteAddr(), "err:", err)
			retried = true
			tcpconn = nil
			goto RECONNECT
		}
	}
	if subc != nil {
		subc.last = time.Now()
		subc.cplk.Unlock()
	}
	return int(cnt) - 3, err
}

func (conn *Conn) WriteToPeer(b []byte, ep p2p.EndPoint) (n int, err error) {
	tcpep, ok := ep.(*EndPoint)
	if !ok {
		return 0, p2p.ErrEndpointTypeMistatch
	}
	if len(b) >= 65536 {
		return 0, errors.New("data size " + strconv.Itoa(len(b)) + " is too large")
	}
	locked := conn.cplk.TryLock()
	if !locked {
		if !conn.suberr || len(conn.subs) > 0 {
			if config.ShowDebugLog {
				logrus.Debug("[tcp] try sub write to", tcpep)
			}
			n, err = conn.writeToPeer(b, tcpep, true) // try sub write
			if err == nil {
				return
			}
			conn.suberr = true // fast fail
		}
		conn.cplk.Lock() // add to main queue
	}
	defer conn.cplk.Unlock()
	return conn.writeToPeer(b, tcpep, false)
}

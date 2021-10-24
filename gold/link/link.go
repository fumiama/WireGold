package link

import (
	"errors"
	"net"
	"sync"

	"github.com/fumiama/WireGold/gold/head"
)

type Link struct {
	conn          net.Conn
	peer          *Identity
	peerip        net.IP
	hasKeepRuning bool
}

var (
	connections = make(map[string]*Link)
	connmapmu   sync.RWMutex
)

func Connect(peer string) (l Link, err error) {
	peer = net.ParseIP(peer).String()
	p, ok := IsInPeer(peer)
	if ok {
		connmapmu.RLock()
		lnk, ok := connections[peer]
		connmapmu.RUnlock()
		if ok {
			return *lnk, nil
		}
		l.conn, err = net.Dial("udp", p.EndPoint)
		l.peer = p
		l.peerip = net.ParseIP(peer)
		connmapmu.Lock()
		connections[l.peerip.String()] = &l
		connmapmu.Unlock()
		l.keepAlive()
	} else {
		err = errors.New("peer not exist")
	}
	return
}

func (l *Link) Close() {
	l.conn.Close()
	connmapmu.Lock()
	delete(connections, l.peerip.String())
	connmapmu.Unlock()
}

func (l *Link) Read() *head.Packet {
	return <-l.peer.pipe
}

func (l *Link) Write(p *head.Packet) (n int, err error) {
	d := p.Mashal(me.String(), l.peerip.String())
	d, err = l.peer.Encode(d)
	if err == nil {
		n, err = l.conn.Write(d)
	}
	return
}

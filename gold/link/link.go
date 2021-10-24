package link

import (
	"errors"
	"net"
	"sync"

	"github.com/fumiama/WireGold/gold/head"
	"github.com/sirupsen/logrus"
)

type Link struct {
	PubicKey      [32]byte
	EndPoint      string
	KeepAlive     int64
	pipe          chan *head.Packet
	peerip        net.IP
	endpoint      *net.UDPAddr
	hasKeepRuning bool
}

var (
	connections = make(map[string]*Link)
	connmapmu   sync.RWMutex
	myconn      *net.UDPConn
)

func Connect(peer string) (*Link, error) {
	p, ok := IsInPeer(net.ParseIP(peer).String())
	if ok {
		p.keepAlive()
		return p, nil
	}
	return nil, errors.New("peer not exist")
}

func (l *Link) Close() {
	connmapmu.Lock()
	delete(connections, l.peerip.String())
	connmapmu.Unlock()
}

func (l *Link) Read() *head.Packet {
	return <-l.pipe
}

func (l *Link) Write(p *head.Packet) (n int, err error) {
	p.Data, err = l.Encode(p.Data)
	if err == nil {
		var d []byte
		d, err = p.Mashal(me.String(), l.peerip.String())
		logrus.Debugln("[link] write data", string(d))
		if err == nil {
			n, err = myconn.WriteToUDP(d, l.endpoint)
		}
	}
	return
}

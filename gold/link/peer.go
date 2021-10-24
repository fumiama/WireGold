package link

import (
	"net"
	"sync"

	"github.com/fumiama/WireGold/gold/head"
)

var (
	eps  = make(map[string]*Link)
	epmu sync.RWMutex
)

func AddPeer(peerip string, pubicKey [32]byte, endPoint string, keepAlive int64) (l *Link) {
	peerip = net.ParseIP(peerip).String()
	var ok bool
	l, ok = IsInPeer(peerip)
	if ok {
		return
	}
	e, err := net.ResolveUDPAddr("udp", endPoint)
	if err != nil {
		panic(err)
	}
	l = &Link{
		PubicKey:  pubicKey,
		EndPoint:  endPoint,
		KeepAlive: keepAlive,
		pipe:      make(chan *head.Packet, 32),
		peerip:    net.ParseIP(peerip),
		endpoint:  e,
	}
	connmapmu.Lock()
	epmu.Lock()
	connections[peerip] = l
	eps[endPoint] = l
	connmapmu.Unlock()
	epmu.Unlock()
	return
}

func IsInPeer(peer string) (p *Link, ok bool) {
	connmapmu.RLock()
	p, ok = connections[peer]
	connmapmu.RUnlock()
	return
}

func IsEndpointInPeer(ep string) (p *Link, ok bool) {
	epmu.RLock()
	p, ok = eps[ep]
	epmu.RUnlock()
	return
}

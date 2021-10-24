package link

import (
	"net"

	"github.com/fumiama/WireGold/gold/head"
)

func AddPeer(peerip string, pubicKey [32]byte, endPoint string, keepAlive int64) (l *Link) {
	peerip = net.ParseIP(peerip).String()
	var ok bool
	l, ok = IsInPeer(peerip)
	if ok {
		return
	}
	l = &Link{
		PubicKey:  pubicKey,
		KeepAlive: keepAlive,
		pipe:      make(chan *head.Packet, 32),
		peerip:    net.ParseIP(peerip),
	}
	if endPoint != "" {
		e, err := net.ResolveUDPAddr("udp", endPoint)
		if err != nil {
			panic(err)
		}
		l.EndPoint = endPoint
		l.endpoint = e
	}
	connmapmu.Lock()
	connections[peerip] = l
	connmapmu.Unlock()
	return
}

func IsInPeer(peer string) (p *Link, ok bool) {
	connmapmu.RLock()
	p, ok = connections[peer]
	connmapmu.RUnlock()
	return
}

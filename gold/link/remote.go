package link

import (
	"net"
	"sync"

	"github.com/fumiama/WireGold/gold/head"
)

type Identity struct {
	PubicKey  [32]byte
	EndPoint  string
	KeepAlive int64
	pipe      chan *head.Packet
}

var (
	peers   = make(map[string]*Identity)
	peersmu sync.RWMutex
)

func AddPeer(peerip string, pubicKey [32]byte, endPoint string, keepAlive int64) (i *Identity) {
	peerip = net.ParseIP(peerip).String()
	var ok bool
	peersmu.RLock()
	i, ok = peers[peerip]
	peersmu.RUnlock()
	if ok {
		return
	}
	i = &Identity{
		PubicKey:  pubicKey,
		EndPoint:  endPoint,
		KeepAlive: keepAlive,
		pipe:      make(chan *head.Packet, 32),
	}
	peersmu.Lock()
	peers[peerip] = i
	peersmu.Unlock()
	return
}

func IsInPeer(peer string) (p *Identity, ok bool) {
	peersmu.RLock()
	p, ok = peers[peer]
	peersmu.RUnlock()
	return
}

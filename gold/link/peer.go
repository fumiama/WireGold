package link

import (
	"net"

	curve "github.com/fumiama/go-x25519"

	"github.com/fumiama/WireGold/gold/head"
)

// AddPeer 添加一个 peer
func AddPeer(peerip string, pubicKey *[32]byte, endPoint string, allowedIPs []string, keepAlive int64, allowTrans bool) (l *Link) {
	peerip = net.ParseIP(peerip).String()
	var ok bool
	l, ok = IsInPeer(peerip)
	if ok {
		return
	}
	l = &Link{
		pubk:       pubicKey,
		keepalive:  keepAlive,
		pipe:       make(chan *head.Packet, 32),
		peerip:     net.ParseIP(peerip),
		allowtrans: allowTrans,
	}
	if pubicKey != nil {
		c := curve.Get(privKey)
		k, err := c.Shared(pubicKey)
		if err == nil {
			l.key = &k
		}
	}
	if endPoint != "" {
		e, err := net.ResolveUDPAddr("udp", endPoint)
		if err != nil {
			panic(err)
		}
		l.pep = endPoint
		l.endpoint = e
	}
	if allowedIPs != nil {
		l.allowedips = make([]*net.IPNet, len(allowedIPs))
		for _, ipnet := range allowedIPs {
			_, cidr, err := net.ParseCIDR(ipnet)
			if err == nil {
				l.allowedips = append(l.allowedips, cidr)
				routetable[cidr.String()] = append(routetable[cidr.String()], l)
			}
		}
	}
	connmapmu.Lock()
	connections[peerip] = l
	connmapmu.Unlock()
	return
}

// IsInPeer 查找 peer 是否已经在册
func IsInPeer(peer string) (p *Link, ok bool) {
	connmapmu.RLock()
	p, ok = connections[peer]
	connmapmu.RUnlock()
	return
}

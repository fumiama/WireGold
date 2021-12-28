package link

import (
	"fmt"
	"net"
	"unsafe"

	curve "github.com/fumiama/go-x25519"

	"github.com/fumiama/WireGold/gold/head"
)

// AddPeer 添加一个 peer
func (m *Me) AddPeer(peerip string, pubicKey *[32]byte, endPoint string, allowedIPs []string, keepAlive int64, allowTrans bool) (l *Link) {
	peerip = net.ParseIP(peerip).String()
	var ok bool
	l, ok = m.IsInPeer(peerip)
	if ok {
		return
	}
	l = &Link{
		pubk:       pubicKey,
		keepalive:  keepAlive,
		pipe:       make(chan *head.Packet, 32),
		peerip:     net.ParseIP(peerip),
		allowtrans: allowTrans,
		me:         m,
	}
	if pubicKey != nil {
		c := curve.Get(m.privKey[:])
		k, err := c.Shared(pubicKey)
		if err == nil {
			fmt.Println(len(k))
			l.key = (*[32]byte)(*(*unsafe.Pointer)(unsafe.Pointer(&k)))
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
				l.me.router.routetable[cidr.String()] = append(l.me.router.routetable[cidr.String()], l)
			}
		}
	}
	l.me.connmapmu.Lock()
	l.me.connections[peerip] = l
	l.me.connmapmu.Unlock()
	return
}

// IsInPeer 查找 peer 是否已经在册
func (m *Me) IsInPeer(peer string) (p *Link, ok bool) {
	m.connmapmu.RLock()
	p, ok = m.connections[peer]
	m.connmapmu.RUnlock()
	return
}

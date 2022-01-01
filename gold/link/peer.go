package link

import (
	"net"
	"time"
	"unsafe"

	"github.com/fumiama/WireGold/gold/head"
	curve "github.com/fumiama/go-x25519"
	"github.com/sirupsen/logrus"
)

// AddPeer 添加一个 peer
func (m *Me) AddPeer(peerip string, pubicKey *[32]byte, endPoint string, allowedIPs, querys []string, keepAlive, queryTick int64, allowTrans, nopipe bool) (l *Link) {
	peerip = net.ParseIP(peerip).String()
	var ok bool
	l, ok = m.IsInPeer(peerip)
	if ok {
		return
	}
	l = &Link{
		pubk:       pubicKey,
		keepalive:  keepAlive,
		peerip:     net.ParseIP(peerip),
		allowtrans: allowTrans,
		me:         m,
	}

	if !nopipe {
		l.pipe = make(chan *head.Packet, 32)
	}
	if pubicKey != nil {
		c := curve.Get(m.privKey[:])
		k, err := c.Shared(pubicKey)
		if err == nil {
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
		l.allowedips = make([]*net.IPNet, 0, len(allowedIPs))
		for _, ipnet := range allowedIPs {
			_, cidr, err := net.ParseCIDR(ipnet)
			if err == nil {
				l.allowedips = append(l.allowedips, cidr)
				l.me.router.SetItem(cidr, l)
				l.me.connmapmu.Lock()
				l.me.connections[peerip] = l
				l.me.connmapmu.Unlock()
			} else {
				panic(err)
			}
		}
	}
	logrus.Infoln("[peer] add peer:", peerip, "allow:", allowedIPs)
	go l.keepAlive()
	go l.sendquery(time.Second*time.Duration(queryTick), querys...)
	return
}

// IsInPeer 查找 peer 是否已经在册
func (m *Me) IsInPeer(peer string) (p *Link, ok bool) {
	m.connmapmu.RLock()
	p, ok = m.connections[peer]
	m.connmapmu.RUnlock()
	return
}

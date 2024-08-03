package link

import (
	"net"
	"time"

	"github.com/fumiama/WireGold/gold/head"
	"github.com/fumiama/WireGold/gold/p2p"
	curve "github.com/fumiama/go-x25519"
	"github.com/sirupsen/logrus"
	"golang.org/x/crypto/chacha20poly1305"
)

type PeerConfig struct {
	PeerIP                  string
	EndPoint                string
	AllowedIPs, Querys      []string
	PubicKey                *[32]byte
	PresharedKey            *[32]byte
	KeepAliveDur, QueryTick int64
	MTU                     uint16
	MTURandomRange          uint16
	AllowTrans, NoPipe      bool
	UseZstd                 bool
	DoublePacket            bool
}

// AddPeer 添加一个 peer
func (m *Me) AddPeer(cfg *PeerConfig) (l *Link) {
	cfg.PeerIP = net.ParseIP(cfg.PeerIP).String()
	var ok bool
	l, ok = m.IsInPeer(cfg.PeerIP)
	if ok {
		return
	}
	if m.mtu == 0 {
		panic("invalid mtu for peer " + cfg.PeerIP)
	}
	l = &Link{
		pubk:           cfg.PubicKey,
		peerip:         net.ParseIP(cfg.PeerIP),
		rawep:          cfg.EndPoint,
		allowtrans:     cfg.AllowTrans,
		usezstd:        cfg.UseZstd,
		doublepacket:   cfg.DoublePacket,
		me:             m,
		mtu:            cfg.MTU,
		mturandomrange: cfg.MTURandomRange,
	}

	if !cfg.NoPipe {
		l.pipe = make(chan *head.Packet, 32)
	}
	var k, p []byte
	if cfg.PubicKey != nil {
		k, _ = curve.Get(m.privKey[:]).Shared(cfg.PubicKey)
	}
	if cfg.PresharedKey != nil {
		p = cfg.PresharedKey[:]
	}
	if len(k) == 32 {
		var err error
		if len(p) == 32 {
			mixk := mixkeys(k, p)
			for i := range k {
				l.keys[i], err = chacha20poly1305.NewX(mixk[i : i+32])
				if err != nil {
					panic(err)
				}
			}
		} else {
			l.keys[0], err = chacha20poly1305.NewX(k)
			if err != nil {
				panic(err)
			}
		}
	}
	if cfg.EndPoint != "" {
		e, err := p2p.NewEndPoint(m.ep.Network(), cfg.EndPoint, m.networkconfigs...)
		if err != nil {
			panic(err)
		}
		l.endpoint = e
	}
	if cfg.AllowedIPs != nil {
		l.allowedips = make([]*net.IPNet, 0, len(cfg.AllowedIPs))
		for _, ipnet := range cfg.AllowedIPs {
			if len(ipnet) == 0 {
				continue
			}
			noroute := ipnet[0] == 'x'
			if noroute {
				ipnet = ipnet[1:]
				if len(ipnet) == 0 {
					continue
				}
			}
			_, cidr, err := net.ParseCIDR(ipnet)
			if err != nil {
				panic(err)
			}
			l.allowedips = append(l.allowedips, cidr)
			if noroute {
				continue
			}
			l.me.router.SetItem(cidr, l)
			l.me.connmapmu.Lock()
			l.me.connections[cfg.PeerIP] = l
			l.me.connmapmu.Unlock()
		}
	}
	logrus.Infoln("[peer] add peer:", cfg.PeerIP, "allow:", cfg.AllowedIPs)
	go l.keepAlive(cfg.KeepAliveDur)
	go l.sendquery(time.Second*time.Duration(cfg.QueryTick), cfg.Querys...)
	return
}

// IsInPeer 查找 peer 是否已经在册
func (m *Me) IsInPeer(peer string) (p *Link, ok bool) {
	m.connmapmu.RLock()
	p, ok = m.connections[peer]
	m.connmapmu.RUnlock()
	return
}

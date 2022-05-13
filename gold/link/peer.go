package link

import (
	"net"
	"time"

	"github.com/fumiama/WireGold/gold/head"
	curve "github.com/fumiama/go-x25519"
	tea "github.com/fumiama/gofastTEA"
	"github.com/sirupsen/logrus"
)

type PeerConfig struct {
	PeerIP                  string
	EndPoint                string
	AllowedIPs, Querys      []string
	PubicKey                *[32]byte
	KeepAliveDur, QueryTick int64
	MTU                     uint16
	AllowTrans, NoPipe      bool
}

// AddPeer 添加一个 peer
func (m *Me) AddPeer(cfg *PeerConfig) (l *Link) {
	cfg.PeerIP = net.ParseIP(cfg.PeerIP).String()
	var ok bool
	l, ok = m.IsInPeer(cfg.PeerIP)
	if ok {
		return
	}
	if cfg.MTU == 0 || cfg.MTU == 65535 || cfg.MTU > m.mtu {
		panic("invalid mtu for peer " + cfg.PeerIP)
	}
	l = &Link{
		pubk:       cfg.PubicKey,
		peerip:     net.ParseIP(cfg.PeerIP),
		allowtrans: cfg.AllowTrans,
		me:         m,
		mtu:        uint16(cfg.MTU),
	}

	if !cfg.NoPipe {
		l.pipe = make(chan *head.Packet, 32)
	}
	if cfg.PubicKey != nil {
		c := curve.Get(m.privKey[:])
		k, err := c.Shared(cfg.PubicKey)
		if err == nil {
			l.key = make([]tea.TEA, 16)
			for i := range l.key {
				l.key[i] = tea.NewTeaCipherLittleEndian(k[i : 16+i])
			}
		}
	}
	if cfg.EndPoint != "" {
		e, err := net.ResolveUDPAddr("udp", cfg.EndPoint)
		if err != nil {
			panic(err)
		}
		l.endpoint = e
	}
	if cfg.AllowedIPs != nil {
		l.allowedips = make([]*net.IPNet, 0, len(cfg.AllowedIPs))
		for _, ipnet := range cfg.AllowedIPs {
			_, cidr, err := net.ParseCIDR(ipnet)
			if err == nil {
				l.allowedips = append(l.allowedips, cidr)
				l.me.router.SetItem(cidr, l)
				l.me.connmapmu.Lock()
				l.me.connections[cfg.PeerIP] = l
				l.me.connmapmu.Unlock()
			} else {
				panic(err)
			}
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

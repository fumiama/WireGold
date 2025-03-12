package link

import (
	"net"
	"sync/atomic"
	"time"
	"unsafe"

	curve "github.com/fumiama/go-x25519"
	"github.com/sirupsen/logrus"
	"golang.org/x/crypto/chacha20poly1305"

	"github.com/fumiama/WireGold/config"
	"github.com/fumiama/WireGold/gold/p2p"
	"github.com/fumiama/WireGold/internal/algo"
	"github.com/fumiama/WireGold/internal/bin"
	"github.com/fumiama/WireGold/internal/file"
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
	if cfg.MTU == 0 {
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
		l.pipe = make(chan LinkData, 4096)
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
			mixk := algo.MixKeys(k, p)
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
			innerroute := ipnet[0] == 'y'
			if noroute || innerroute {
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
	go l.sendQuery(time.Second*time.Duration(cfg.QueryTick), cfg.Querys...)
	return
}

// IsInPeer 查找 peer 是否已经在册
func (m *Me) IsInPeer(peer string) (p *Link, ok bool) {
	m.connmapmu.RLock()
	p, ok = m.connections[peer]
	m.connmapmu.RUnlock()
	return
}

func (m *Me) extractPeer(srcip, dstip net.IP, addr p2p.EndPoint) *Link {
	p, ok := m.IsInPeer(srcip.String())
	if config.ShowDebugLog {
		logrus.Debugln(file.Header(), "recv from endpoint", addr, "src", srcip, "dst", dstip)
	}
	if !ok {
		logrus.Warnln(file.Header(), "packet from", srcip, "to", dstip, "is refused")
		return nil
	}
	if bin.IsNilInterface(p.endpoint) || !p.endpoint.Euqal(addr) {
		if m.ep.Network() == "tcp" && !addr.Euqal(p.endpoint) {
			logrus.Infoln(file.Header(), "set endpoint of peer", p.peerip, "to", addr.String())
			p.endpoint = addr
		} else { // others are all no status link
			logrus.Infoln(file.Header(), "set endpoint of peer", p.peerip, "to", addr.String())
			p.endpoint = addr
		}
	}
	now := time.Now()
	atomic.StorePointer((*unsafe.Pointer)(unsafe.Pointer(&p.lastalive)), unsafe.Pointer(&now))
	return p
}

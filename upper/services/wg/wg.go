package wg

import (
	"errors"
	"net"
	"strconv"

	base14 "github.com/fumiama/go-base16384"
	curve "github.com/fumiama/go-x25519"
	"github.com/sirupsen/logrus"

	_ "github.com/fumiama/WireGold/gold/p2p/ip"      // support ip connection
	_ "github.com/fumiama/WireGold/gold/p2p/tcp"     // support tcp connection
	_ "github.com/fumiama/WireGold/gold/p2p/udp"     // support udp connection
	_ "github.com/fumiama/WireGold/gold/p2p/udplite" // support udplite connection

	"github.com/fumiama/WireGold/config"
	"github.com/fumiama/WireGold/gold/link"
	"github.com/fumiama/WireGold/helper"
)

const suffix32 = "㴄"

type WG struct {
	c         *config.Config
	key       [32]byte
	PublicKey string
	me        link.Me
}

func NewWireGold(c *config.Config) (wg WG, err error) {
	wg.c = c

	var k []byte
	k, err = base14.UTF82UTF16BE(helper.StringToBytes(c.PrivateKey + suffix32))
	if err != nil {
		return
	}
	n := copy(wg.key[:], base14.Decode(k))
	if n != 32 {
		err = errors.New("private key length != 32, got " + strconv.Itoa(n))
		return
	}

	cur := curve.Get(wg.key[:])
	pubk, err := base14.UTF16BE2UTF8(base14.Encode((*cur.Public())[:]))
	if err != nil {
		return
	}
	wg.PublicKey = helper.BytesToString(pubk[:57])

	return
}

func (wg *WG) Start(srcport, destport uint16) {
	go wg.Run(srcport, destport)
}

func (wg *WG) Run(srcport, destport uint16) {
	wg.init(srcport, destport)
	_, _ = wg.me.ListenNIC()
	logrus.Info("[wg] stopped")
}

func (wg *WG) Stop() {
	logrus.Warnln("[wg] stopping...")
	_ = wg.me.Close()
}

func (wg *WG) init(srcport, dstport uint16) {
	cidrsmap := make(map[string]bool, 32)
	_, mysubnet, err := net.ParseCIDR(wg.c.SubNet)
	if err != nil {
		panic(err)
	}
	myip := net.ParseIP(wg.c.IP)
	if myip == nil {
		panic("invalid ip " + wg.c.IP)
	}
	for _, p := range wg.c.Peers {
		for _, ip := range p.AllowedIPs {
			if len(ip) == 0 || ip[0] == 'x' {
				continue
			}
			ipnet, _, err := net.ParseCIDR(ip)
			if err != nil {
				panic(err)
			}
			if !mysubnet.Contains(ipnet) {
				cidrsmap[ip] = true
			}
		}
	}
	cidrs := make([]string, len(cidrsmap))
	i := 0
	for k := range cidrsmap {
		cidrs[i] = k
		i++
	}

	wg.me = link.NewMe(&link.MyConfig{
		MyIPwithMask: myip.String() + "/32",
		MyEndpoint:   wg.c.EndPoint,
		Network:      wg.c.Network,
		PrivateKey:   &wg.key,
		NICConfig: &link.NICConfig{
			IP:     myip,
			SubNet: mysubnet,
			CIDRs:  cidrs,
		},
		SrcPort:   srcport,
		DstPort:   dstport,
		MTU:       uint16(wg.c.MTU),
		SpeedLoop: wg.c.SpeedLoop,
		Mask:      wg.c.Mask,
	})

	for _, peer := range wg.c.Peers {
		var peerkey [32]byte
		k, err := base14.UTF82UTF16BE(helper.StringToBytes(peer.PublicKey + suffix32))
		if err != nil {
			panic(err)
		}
		n := copy(peerkey[:], base14.Decode(k))
		if n != 32 {
			panic("peer " + peer.IP + ": public key length < 32")
		}
		var pshk *[32]byte
		if peer.PresharedKey != "" {
			k, err := base14.UTF82UTF16BE(helper.StringToBytes(peer.PresharedKey + suffix32))
			if err != nil {
				panic(err)
			}
			pshk = &[32]byte{}
			n := copy(pshk[:], base14.Decode(k))
			if n != 32 {
				panic("peer " + peer.IP + ": preshared key length < 32")
			}
		}
		if peer.MTU >= 65535 {
			panic("peer " + peer.IP + ": MTU too large")
		}
		if peer.MTURandomRange >= peer.MTU/2 {
			panic("peer " + peer.IP + ": MTURandomRange too large")
		}
		wg.me.AddPeer(&link.PeerConfig{
			PeerIP:         peer.IP,
			EndPoint:       peer.EndPoint,
			AllowedIPs:     peer.AllowedIPs,
			Querys:         peer.QueryList,
			PubicKey:       &peerkey,
			PresharedKey:   pshk,
			KeepAliveDur:   peer.KeepAliveSeconds,
			QueryTick:      peer.QuerySeconds,
			MTU:            uint16(peer.MTU),
			MTURandomRange: uint16(peer.MTURandomRange),
			AllowTrans:     peer.AllowTrans,
			NoPipe:         true,
			UseZstd:        peer.UseZstd,
			DoublePacket:   peer.DoublePacket,
		})
	}
}

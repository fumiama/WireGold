package wg

import (
	"errors"
	"net"

	base14 "github.com/fumiama/go-base16384"
	curve "github.com/fumiama/go-x25519"

	"github.com/fumiama/WireGold/config"
	"github.com/fumiama/WireGold/gold/link"
	"github.com/fumiama/WireGold/helper"
	"github.com/fumiama/WireGold/lower"
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
	k, err = base14.UTF82utf16be(helper.StringToBytes(c.PrivateKey + suffix32))
	if err != nil {
		return
	}
	n := copy(wg.key[:], base14.Decode(k))
	if n != 32 {
		err = errors.New("private key length is not 32")
		return
	}

	cur := curve.Get(wg.key[:])
	pubk, err := base14.UTF16be2utf8(base14.Encode((*cur.Public())[:]))
	if err != nil {
		return
	}
	wg.PublicKey = helper.BytesToString(pubk[:57])

	return
}

func (wg *WG) Start(srcport, destport, mtu uint16) {
	wg.init(srcport, destport, mtu)
	go wg.me.ListenFromNIC()
}

func (wg *WG) Run(srcport, destport, mtu uint16) {
	wg.init(srcport, destport, mtu)
	wg.me.ListenFromNIC()
}

func (wg *WG) Stop() {
	_ = wg.me.Close()
}

func (wg *WG) init(srcport, destport, mtu uint16) {
	cidrsmap := make(map[string]bool, 32)
	_, mysubnet, err := net.ParseCIDR(wg.c.SubNet)
	if err != nil {
		panic(err)
	}
	for _, p := range wg.c.Peers {
		for _, ip := range p.AllowedIPs {
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

	wg.me = link.NewMe(
		&wg.key,
		wg.c.IP+"/32",
		wg.c.EndPoint,
		lower.NewNIC(wg.c.IP, wg.c.SubNet, cidrs...),
		srcport, destport, mtu,
	)

	for _, peer := range wg.c.Peers {
		var peerkey [32]byte
		k, err := base14.UTF82utf16be(helper.StringToBytes(peer.PublicKey + suffix32))
		if err != nil {
			panic(err)
		}
		n := copy(peerkey[:], base14.Decode(k))
		if n != 32 {
			panic("peer public key length is not 32")
		}
		wg.me.AddPeer(peer.IP, &peerkey, peer.EndPoint, peer.AllowedIPs, peer.KeepAliveSeconds, peer.AllowTrans, true)
	}
}

package ip

import (
	"net"
	"net/netip"

	"github.com/fumiama/WireGold/gold/p2p"
	"github.com/fumiama/WireGold/helper"
)

func NewEndpoint(endpoint string, configs ...any) (p2p.EndPoint, error) {
	addr, err := netip.ParseAddr(endpoint)
	if err != nil {
		return nil, err
	}
	ptcl := uint(0x04) // IPIP
	if len(configs) > 0 {
		ptcl = configs[0].(uint)
	}
	return &EndPoint{
		addr: &net.IPAddr{
			IP:   addr.AsSlice(),
			Zone: addr.Zone(),
		},
		ptcl: ptcl,
	}, nil
}

func init() {
	name := helper.FolderName()
	_, hasexist := p2p.Register(name, NewEndpoint)
	if hasexist {
		panic("network " + name + " has been registered")
	}
}

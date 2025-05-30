package ip

import (
	"net"
	"net/netip"

	"github.com/fumiama/WireGold/gold/p2p"
	"github.com/fumiama/WireGold/internal/file"
)

func NewEndpoint(endpoint string, configs ...any) (p2p.EndPoint, error) {
	addr, err := netip.ParseAddr(endpoint)
	if err != nil {
		return nil, err
	}
	ptcl := uint(0x6C) // IPComp https://datatracker.ietf.org/doc/html/rfc3173
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
	name := file.FolderName()
	_, hasexist := p2p.Register(name, NewEndpoint)
	if hasexist {
		panic("network " + name + " has been registered")
	}
}

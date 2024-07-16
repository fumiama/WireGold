package udp

import (
	"net"
	"net/netip"

	"github.com/fumiama/WireGold/gold/p2p"
)

func NewEndpoint(endpoint string, _ ...any) (p2p.EndPoint, error) {
	addr, err := netip.ParseAddrPort(endpoint)
	if err != nil {
		return nil, err
	}
	return (*EndPoint)(net.UDPAddrFromAddrPort(addr)), nil
}

func init() {
	_, hasexist := p2p.Register("udp", NewEndpoint)
	if hasexist {
		panic("network udp has been registered")
	}
}

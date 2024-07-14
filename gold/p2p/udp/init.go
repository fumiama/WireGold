package udp

import (
	"errors"
	"net"
	"net/netip"

	"github.com/fumiama/WireGold/gold/p2p"
)

var (
	ErrEndpointTypeMistatch = errors.New("endpoint type mismatch")
)

func NewEndpoint(endpoint string, _ ...any) p2p.EndPoint {
	return (*EndPoint)(net.UDPAddrFromAddrPort(
		netip.MustParseAddrPort(endpoint),
	))
}

func init() {
	_, hasexist := p2p.Register("udp", NewEndpoint)
	if hasexist {
		panic("network udp has been registered")
	}
}

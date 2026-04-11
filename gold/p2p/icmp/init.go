// Package icmp for non-privileged datagram-oriented ICMP endpoints,
// currently only Darwin and Linux support this.
package icmp

import (
	"net/netip"

	"github.com/fumiama/WireGold/gold/p2p"
	"github.com/fumiama/WireGold/internal/file"
)

func NewEndpoint(endpoint string, _ ...any) (p2p.EndPoint, error) {
	addr, err := netip.ParseAddr(endpoint)
	if err != nil {
		return nil, err
	}
	return (*EndPoint)(&addr), nil
}

func init() {
	name := file.FolderName()
	_, hasexist := p2p.Register(name, NewEndpoint)
	if hasexist {
		panic("network " + name + " has been registered")
	}
}

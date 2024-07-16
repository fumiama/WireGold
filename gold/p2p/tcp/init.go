package tcp

import (
	"net"
	"net/netip"
	"time"

	"github.com/fumiama/WireGold/gold/p2p"
)

type Config struct {
	PeersTimeout       time.Duration
	ReceiveChannelSize int
}

func NewEndpoint(endpoint string, configs ...any) p2p.EndPoint {
	return newEndpoint(endpoint, configs...)
}

func newEndpoint(endpoint string, configs ...any) *EndPoint {
	var cfg *Config
	if len(configs) == 0 || configs[0] == nil {
		cfg = &Config{}
	} else {
		cfg = configs[0].(*Config)
	}
	return &EndPoint{
		addr: net.TCPAddrFromAddrPort(
			netip.MustParseAddrPort(endpoint),
		),
		peerstimeout: cfg.PeersTimeout,
		recvchansize: cfg.ReceiveChannelSize,
	}
}

func init() {
	_, hasexist := p2p.Register("tcp", NewEndpoint)
	if hasexist {
		panic("network tcp has been registered")
	}
}

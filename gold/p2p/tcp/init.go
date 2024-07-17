package tcp

import (
	"net"
	"net/netip"
	"time"

	"github.com/fumiama/WireGold/gold/p2p"
	"github.com/fumiama/WireGold/helper"
)

type Config struct {
	DialTimeout        time.Duration
	PeersTimeout       time.Duration
	ReceiveChannelSize int
}

func NewEndpoint(endpoint string, configs ...any) (p2p.EndPoint, error) {
	return newEndpoint(endpoint, configs...)
}

func newEndpoint(endpoint string, configs ...any) (*EndPoint, error) {
	var cfg *Config
	if len(configs) == 0 || configs[0] == nil {
		cfg = &Config{}
	} else {
		cfg = configs[0].(*Config)
	}
	addr, err := netip.ParseAddrPort(endpoint)
	if err != nil {
		return nil, err
	}
	return &EndPoint{
		addr:         net.TCPAddrFromAddrPort(addr),
		dialtimeout:  cfg.DialTimeout,
		peerstimeout: cfg.PeersTimeout,
		recvchansize: cfg.ReceiveChannelSize,
	}, nil
}

func init() {
	name := helper.FolderName()
	_, hasexist := p2p.Register(name, NewEndpoint)
	if hasexist {
		panic("network " + name + " has been registered")
	}
}

package link

import (
	"errors"
	"net"

	"github.com/fumiama/WireGold/gold/head"
	"github.com/fumiama/WireGold/helper"
	base14 "github.com/fumiama/go-base16384"
	tea "github.com/fumiama/gofastTEA"
)

// Link 是本机到 peer 的连接抽象
type Link struct {
	// peer 的公钥
	pubk *[32]byte
	// 收到的包的队列
	// 没有下层 nic 时
	// 包会分发到此
	pipe chan *head.Packet
	// peer 的虚拟 ip
	peerip net.IP
	// peer 的公网 endpoint
	endpoint *net.UDPAddr
	// 本机允许接收/发送的 ip 网段
	allowedips []*net.IPNet
	// 连接所用对称加密密钥
	key []tea.TEA
	// 本机信息
	me *Me
	// 连接的状态，详见下方 const
	status int
	// 是否允许转发
	allowtrans bool
}

const (
	LINK_STATUS_DOWN = iota
	LINK_STATUS_HALFUP
	LINK_STATUS_UP
)

// Connect 初始化与 peer 的连接
func (m *Me) Connect(peer string) (*Link, error) {
	p, ok := m.IsInPeer(net.ParseIP(peer).String())
	if ok {
		return p, nil
	}
	return nil, errors.New("peer not exist")
}

// Close 关闭到 peer 的连接
func (l *Link) Close() {
	l.status = LINK_STATUS_DOWN
}

// Destroy 从 connections 移除 peer
func (l *Link) Destroy() {
	l.me.connmapmu.Lock()
	delete(l.me.connections, l.peerip.String())
	l.me.connmapmu.Unlock()
}

func (l *Link) String() (n string) {
	n = "default"
	if l.pubk != nil {
		b, err := base14.UTF16be2utf8(base14.Encode(l.pubk[:7]))
		if err == nil {
			n = helper.BytesToString(b)
		} else {
			n = err.Error()
		}
	}
	return
}

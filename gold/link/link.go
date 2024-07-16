package link

import (
	"crypto/cipher"
	"errors"
	"net"
	"sync/atomic"

	"github.com/fumiama/WireGold/gold/head"
	"github.com/fumiama/WireGold/gold/p2p"
	"github.com/fumiama/WireGold/helper"
	base14 "github.com/fumiama/go-base16384"
)

var (
	ErrPerrNotExist = errors.New("peer not exist")
)

// Link 是本机到 peer 的连接抽象
type Link struct {
	// peer 的公钥
	pubk *[32]byte
	// 发包计数, 分片算一个
	sendcount uintptr
	// 收到的包的队列
	// 没有下层 nic 时
	// 包会分发到此
	pipe chan *head.Packet
	// peer 的虚拟 ip
	peerip net.IP
	// peer 的公网 endpoint
	endpoint p2p.EndPoint
	// peer 在设置的原始值
	rawep string
	// 本机允许接收/发送的 ip 网段
	allowedips []*net.IPNet
	// 连接所用对称加密密钥集
	keys [32]cipher.AEAD
	// 本机信息
	me *Me
	// 连接的状态，详见下方 const
	status int8
	// 是否允许转发
	allowtrans bool
	// 是否对数据进行 zstd 压缩
	usezstd bool
	// udp 数据包的最大大小
	mtu uint16
	// 随机放缩 mtu 范围 (只减不增)
	mturandomrange uint16
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
	return nil, ErrPerrNotExist
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
		b, err := base14.UTF16BE2UTF8(base14.Encode(l.pubk[:7]))
		if err == nil {
			n = helper.BytesToString(b)
		} else {
			n = err.Error()
		}
	}
	return
}

func (l *Link) incgetsndcnt() uintptr {
	return atomic.AddUintptr(&l.sendcount, 1)
}

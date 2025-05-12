package link

import (
	"crypto/cipher"
	"errors"
	"net"
	"sync/atomic"
	"time"

	"github.com/fumiama/WireGold/config"
	"github.com/fumiama/WireGold/gold/head"
	"github.com/fumiama/WireGold/gold/p2p"
	"github.com/fumiama/WireGold/internal/bin"
	base14 "github.com/fumiama/go-base16384"
	"github.com/sirupsen/logrus"
)

var (
	ErrPerrNotExist = errors.New("peer not exist")
)

type LinkData struct {
	H head.Packet
	D []byte
}

// Link 是本机到 peer 的连接抽象
type Link struct {
	// peer 的公钥
	pubk *[32]byte
	// 发包计数, 分片算一个
	sendcount uintptr
	// 收到的包的队列
	// 没有下层 nic 时
	// 包会分发到此
	pipe chan LinkData
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
	// 最后一次收到报文的时间
	lastalive *time.Time
	// 是否允许转发
	allowtrans bool
	// 是否对数据进行 zstd 压缩
	usezstd bool
	// 是否采用双倍发包对抗强丢包
	doublepacket bool
	// udp 数据包的最大大小
	mtu uint16
	// 随机放缩 mtu 范围 (只减不增)
	mturandomrange uint16
}

// Connect 初始化与 peer 的连接
func (m *Me) Connect(peer string) (*Link, error) {
	p, ok := m.IsInPeer(net.ParseIP(peer).String())
	if ok {
		return p, nil
	}
	return nil, ErrPerrNotExist
}

func (l *Link) ToLower(header *head.Packet, data []byte) {
	if l.pipe != nil {
		d := make([]byte, len(data))
		copy(d, data)
		l.pipe <- LinkData{
			H: *header,
			D: d,
		}
		if config.ShowDebugLog {
			logrus.Debugln("[listen] deliver to pipe of", l.peerip)
		}
		return
	}
	_, err := l.me.nic.Write(data)
	if err != nil {
		logrus.Errorln("[listen] deliver", len(data), "bytes data to nic err:", err)
	} else if config.ShowDebugLog {
		logrus.Debugln("[listen] deliver", len(data), "bytes data to nic")
	}
}

// Close 关闭到 peer 的连接
func (l *Link) Close() {
	l.Destroy()
}

// IP is wiregold peer ip
func (l *Link) IP() net.IP {
	return l.peerip
}

// RawEndPoint is initial ep in cfg
func (l *Link) RawEndPoint() string {
	return l.rawep
}

func (l *Link) EndPoint() p2p.EndPoint {
	return l.endpoint
}

func (l *Link) SetEndPoint(ep p2p.EndPoint) {
	l.endpoint = ep
}

func (l *Link) Me() *Me {
	return l.me
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
			n = bin.BytesToString(b)
		} else {
			n = err.Error()
		}
	}
	return
}

func (l *Link) incgetsndcnt() uintptr {
	return atomic.AddUintptr(&l.sendcount, 1)
}

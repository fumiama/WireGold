package link

import (
	"errors"
	"net"

	"github.com/sirupsen/logrus"

	"github.com/fumiama/WireGold/gold/head"
	"github.com/fumiama/WireGold/helper"
	base14 "github.com/fumiama/go-base16384"
)

// Link 是本机到 peer 的连接抽象
type Link struct {
	// peer 的公钥
	pubk *[32]byte
	// peer 的公网 ip:port
	pep string
	// 决定本机是否定时向 peer 发送 hello 保持 NAT。
	// 以秒为单位，小于等于 0 不发送
	keepalive int64
	// 收到的包的队列
	pipe chan *head.Packet
	// peer 的虚拟 ip
	peerip net.IP
	// peer 的公网 endpoint
	endpoint *net.UDPAddr
	// 本机允许接收/发送的 ip 网段
	allowedips []*net.IPNet
	// 是否已经调用过 keepAlive
	haskeepruning bool
	// 是否允许转发
	allowtrans bool
	// 连接的状态，详见下方 const
	status int
	// 连接所用对称加密密钥
	key *[32]byte
	// 本机信息
	me *Me
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
		p.keepAlive()
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

// Read 从 peer 收包
func (l *Link) Read() *head.Packet {
	return <-l.pipe
}

// Write 向 peer 发包
func (l *Link) Write(p *head.Packet, istransfer bool) (n int, err error) {
	p.FillHash()
	p.Data = l.Encode(p.Data)
	var d []byte
	if istransfer {
		d = p.Marshal(nil)
	} else {
		d = p.Marshal(l.me.me)
	}
	if d == nil {
		return 0, errors.New("[link] ttl exceeded")
	}
	logrus.Debugln("[link] write", len(d), "bytes data")
	if err == nil {
		peerlink := l.me.router.NextHop(l.peerip.String())
		if peerlink != nil {
			peerep := peerlink.endpoint
			if peerep == nil {
				return 0, errors.New("[link] nil endpoint of " + l.peerip.String())
			}
			logrus.Infoln("[link] write data from ep", l.me.myconn.LocalAddr(), "to", peerep)
			n, err = l.me.myconn.WriteToUDP(d, peerep)
		} else {
			logrus.Warnln("[link] drop packet: nil peerlink")
		}
	}
	return
}

func (l *Link) String() (n string) {
	n = "default"
	if l.pubk != nil {
		b, err := base14.UTF16be2utf8(base14.Encode(l.pubk[:21]))
		if err == nil {
			n = helper.BytesToString(b)
		} else {
			n = err.Error()
		}
	}
	return
}

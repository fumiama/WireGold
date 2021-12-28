package link

import (
	"errors"
	"net"

	"github.com/sirupsen/logrus"

	"github.com/fumiama/WireGold/gold/head"
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
	l.me.connmapmu.Lock()
	delete(l.me.connections, l.peerip.String())
	l.me.connmapmu.Unlock()
	l.status = LINK_STATUS_DOWN
}

// Read 从 peer 收包
func (l *Link) Read() *head.Packet {
	return <-l.pipe
}

// Write 向 peer 发包
func (l *Link) Write(p *head.Packet) (n int, err error) {
	p.FillHash()
	p.Data = l.Encode(p.Data)
	var d []byte
	d, err = p.Marshal(l.me.me.String(), l.peerip.String())
	logrus.Debugln("[link] write data", string(d))
	if err == nil {
		n, err = l.me.myconn.WriteToUDP(d, l.me.router.NextHop(l.peerip.String()+"/32").endpoint)
	}
	return
}

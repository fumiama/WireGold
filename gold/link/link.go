package link

import (
	"errors"
	"fmt"
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
	// 没有下层 nic 时
	// 包会分发到此
	pipe chan *head.Packet
	// peer 的虚拟 ip
	peerip net.IP
	// peer 的公网 endpoint
	endpoint *net.UDPAddr
	// 本机允许接收/发送的 ip 网段
	allowedips []*net.IPNet
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
	if len(p.Data) <= int(l.me.mtu) {
		if !istransfer {
			p.FillHash()
			p.Data = l.Encode(p.Data)
		}
		return l.write(p, uint32(len(p.Data)), 0, istransfer, false)
	}
	if !istransfer {
		p.FillHash()
		p.Data = l.Encode(p.Data)
	}
	data := p.Data
	totl := uint32(len(data))
	i := 0
	for ; int(totl)-i > int(l.me.mtu); i += int(l.me.mtu) {
		logrus.Debugln("[link] split frag", i, ":", i+int(l.me.mtu), ", remain:", int(totl)-i-int(l.me.mtu))
		packet := *p
		packet.Data = data[:int(l.me.mtu)]
		cnt, err := l.write(&packet, totl, uint16(uint(i)>>3), istransfer, true)
		n += cnt
		if err != nil {
			return n, err
		}
		data = data[int(l.me.mtu):]
	}
	p.Data = data
	cnt, err := l.write(p, totl, uint16(uint(i)>>3), istransfer, false)
	n += cnt
	if err != nil {
		return n, err
	}
	return n, nil
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

// write 向 peer 发一个包
func (l *Link) write(p *head.Packet, datasz uint32, offset uint16, istransfer, hasmore bool) (n int, err error) {
	var d []byte
	var cl func()
	if istransfer {
		if p.Flags&0x4000 == 0x4000 && len(p.Data) > int(l.me.mtu) {
			return len(p.Data), errors.New("drop dont fragmnet big trans packet")
		}
		d, cl = p.Marshal(nil, 0, 0, false, false)
	} else {
		d, cl = p.Marshal(l.me.me, datasz, offset, false, hasmore)
	}
	if d == nil {
		return 0, errors.New("[link] ttl exceeded")
	}
	if err == nil {
		peerep := l.endpoint
		if peerep == nil {
			return 0, errors.New("[link] nil endpoint of " + p.Dst.String())
		}
		logrus.Debugln("[link] write", len(d), "bytes data from ep", l.me.myconn.LocalAddr(), "to", peerep, "offset:", fmt.Sprintf("%04x", offset))
		n, err = l.me.myconn.WriteToUDP(d, peerep)
		cl()
	}
	return
}

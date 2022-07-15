package link

import (
	"encoding/binary"
	"io"
	"net"
	"strconv"
	"sync"
	"time"

	"github.com/FloatTech/ttl"
	"github.com/fumiama/WireGold/gold/head"
	"github.com/fumiama/WireGold/helper"
	"github.com/fumiama/WireGold/lower"
	"github.com/fumiama/water/waterutil"
	"github.com/sirupsen/logrus"
)

// Me 是本机的抽象
type Me struct {
	// 本机私钥
	// 利用 Curve25519 生成
	// https://pkg.go.dev/golang.org/x/crypto/curve25519
	// https://www.zhihu.com/question/266758647
	privKey [32]byte
	// 本机虚拟 ip
	me net.IP
	// 本机子网
	subnet net.IPNet
	// 本机 endpoint
	myend *net.UDPAddr
	// 本机环回 link
	loop *Link
	// 本机活跃的所有连接
	connections map[string]*Link
	// 读写同步锁
	connmapmu sync.RWMutex
	// 本机监听的 endpoint
	myep *net.UDPConn
	// 本机网卡
	nic lower.NICIO
	// 本机路由表
	router *Router
	// 本机发送缓冲区
	writer *helper.Writer
	// 本机未接收完全分片池
	recving *ttl.Cache[[32]byte, *head.Packet]
	// 抗重放攻击记录池
	recved *ttl.Cache[uint64, uint8]
	// 本机上层配置
	srcport, dstport, mtu uint16
}

type MyConfig struct {
	MyIPwithMask          string
	MyEndpoint            string
	PrivateKey            *[32]byte
	NIC                   lower.NICIO
	SrcPort, DstPort, MTU uint16
}

// NewMe 设置本机参数
func NewMe(cfg *MyConfig) (m Me) {
	m.privKey = *cfg.PrivateKey
	var err error
	m.myend, err = net.ResolveUDPAddr("udp", cfg.MyEndpoint)
	if err != nil {
		panic(err)
	}
	ip, cidr, err := net.ParseCIDR(cfg.MyIPwithMask)
	if err != nil {
		panic(err)
	}
	m.me = ip
	m.subnet = *cidr
	m.myep, err = m.listen()
	if err != nil {
		panic(err)
	}
	m.connections = make(map[string]*Link)
	m.nic = cfg.NIC
	m.router = &Router{
		list:  make([]*net.IPNet, 1, 16),
		table: make(map[string]*Link, 16),
		cache: ttl.NewCache[string, *Link](time.Minute),
	}
	m.router.SetDefault(nil)
	m.loop = m.AddPeer(&PeerConfig{
		PeerIP:     m.me.String(),
		EndPoint:   "127.0.0.1:56789",
		AllowedIPs: []string{cfg.MyIPwithMask},
		NoPipe:     cfg.NIC != nil,
		MTU:        cfg.MTU,
	})
	m.srcport = cfg.SrcPort
	m.dstport = cfg.DstPort
	m.mtu = cfg.MTU & 0xfff8
	if m.writer == nil {
		m.writer = helper.SelectWriter()
	}
	m.recving = ttl.NewCache[[32]byte, *head.Packet](time.Second * 30)
	m.recved = ttl.NewCache[uint64, uint8](time.Second * 30)
	return
}

func (m *Me) SrcPort() uint16 {
	return m.srcport
}

func (m *Me) DstPort() uint16 {
	return m.dstport
}

func (m *Me) MTU() uint16 {
	return m.mtu
}

func (m *Me) Close() error {
	m.nic.Down()
	return m.nic.Close()
}

func (m *Me) Write(packet []byte) (n int, err error) {
	remain := m.writer.Len()
	if remain > 0 {
		m.writer.Write(packet)
		packet = m.writer.Bytes()
	}
	logrus.Debugln("[me] writer eating", len(packet), "bytes...")
	n = m.sendAllSameDst(packet)
	if len(packet) > n {
		_, _ = m.writer.Skip(remain + n - len(packet))
		logrus.Debugln("[me] writer remain", m.writer.Len(), "bytes")
	} else if n > 0 && remain > 0 {
		m.writer.Reset()
		logrus.Debugln("[me] writer becomes empty")
	}
	return
}

func (m *Me) ListenFromNIC() (written int64, err error) {
	m.nic.Up()
	return io.Copy(m, m.nic)
}

type packetID [2]byte

func newpacketid(packet []byte) packetID {
	return waterutil.IPv4Identification(packet)
}

func (p packetID) issame(packet []byte) bool {
	return p == waterutil.IPv4Identification(packet)
}

func (m *Me) sendAllSameDst(packet []byte) (n int) {
	rem := packet
	if !waterutil.IsIPv4(packet) {
		for len(rem) > 20 && waterutil.IsIPv6(rem) {
			pktl := int(binary.BigEndian.Uint16(packet[4:6])) + 40
			if pktl > len(rem) {
				return
			}
			n += pktl
			rem = packet[n:]
			logrus.Debugln("[me] skip to send", len(packet), "bytes ipv6 packet")
		}
		if len(rem) == 0 || !waterutil.IsIPv4(rem) {
			logrus.Warnln("[me] skip to send", len(packet), "bytes full packet")
			return len(packet)
		}
	}
	p := newpacketid(rem)
	ptr := rem
	i := 0
	for len(ptr) > 20 && p.issame(ptr) {
		totl := waterutil.IPv4TotalLength(ptr)
		if int(totl) > len(ptr) {
			break
		}
		i += int(totl)
		ptr = rem[i:]
		logrus.Debugln("[me] wrap", totl, "bytes packet to send together")
	}
	if i == 0 {
		return
	}
	n += i
	packet = rem[:i]
	rem = rem[i:]
	dst := waterutil.IPv4Destination(packet)
	logrus.Debugln("[me] sending", len(packet), "bytes packet from :"+strconv.Itoa(int(m.SrcPort())), "to", dst.String()+":"+strconv.Itoa(int(m.DstPort())), "remain:", len(rem), "bytes")
	lnk := m.router.NextHop(dst.String())
	if lnk == nil {
		logrus.Warnln("[me] drop packet to", dst.String()+":"+strconv.Itoa(int(m.DstPort())), ": nil nexthop")
		return
	}
	_, err := lnk.WriteAndPut(head.NewPacket(head.ProtoData, m.SrcPort(), lnk.peerip, m.DstPort(), packet), false)
	if err != nil {
		logrus.Warnln("[me] write to peer", lnk.peerip, "err:", err)
	}
	return
}

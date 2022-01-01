package link

import (
	"encoding/binary"
	"io"
	"net"
	"strconv"
	"sync"

	"github.com/fumiama/WireGold/gold/head"
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
	myconn *net.UDPConn
	// 本机网卡
	nic lower.NICIO
	// 本机路由表
	router *Router
	// 本机未接收完全分片池
	recving map[[32]byte]*head.Packet
	recvmu  sync.Mutex
	// 超时定时器
	clock map[*head.Packet]uint8
	// 本机上层配置
	srcport, dstport, mtu uint16
}

// NewMe 设置本机参数
func NewMe(privateKey *[32]byte, myipwithmask string, myEndpoint string, nic lower.NICIO, srcport, dstport, mtu uint16) (m Me) {
	m.privKey = *privateKey
	var err error
	m.myend, err = net.ResolveUDPAddr("udp", myEndpoint)
	if err != nil {
		panic(err)
	}
	ip, cidr, err := net.ParseCIDR(myipwithmask)
	if err != nil {
		panic(err)
	}
	m.me = ip
	m.subnet = *cidr
	m.myconn, err = m.listen()
	if err != nil {
		panic(err)
	}
	m.connections = make(map[string]*Link)
	m.nic = nic
	m.router = &Router{
		list:  make([]*net.IPNet, 1, 16),
		table: make(map[string]*Link, 16),
	}
	m.router.SetDefault(nil)
	m.loop = m.AddPeer(m.me.String(), nil, "127.0.0.1:56789", []string{myipwithmask}, 0, false, nic != nil)
	m.srcport = srcport
	m.dstport = dstport
	m.mtu = mtu & 0xfff8
	go m.initrecvpool()
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

func (m *Me) ListenFromNIC() {
	m.nic.Up()

	// 双缓冲区
	buf := make([]byte, m.MTU()+68)  // 增加报头长度与 TEA 冗余
	buf2 := make([]byte, m.MTU()+68) // 增加报头长度与 TEA 冗余

	off := 0
	isrev := false
	for { // 从 NIC 发送
		var packet []byte
		if off > 0 && !isrev {
			packet = buf2
		} else {
			packet = buf
		}
		n, err := m.nic.Read(packet[off:])
		if isrev {
			off = 0
		}
		if err != nil {
			logrus.Errorln("[lower] send read from nic err:", err)
			break
		}
		if n == 0 {
			continue
		}
		packet = packet[:n]
		n, rem := m.send(m.nic, packet)
		for len(rem) > 20 && n > 0 {
			n, rem = m.send(m.nic, rem)
		}
		if len(rem) > 0 {
			logrus.Debugln("[lower] remain", len(rem), "bytes to send")
			if off > 0 {
				off = copy(buf, rem)
				isrev = true
			} else {
				off = copy(buf2, rem)
			}
		} else {
			off = 0
		}
	}
}

func (m *Me) send(nc io.Reader, packet []byte) (n int, rem []byte) {
	if !waterutil.IsIPv4(packet) {
		if waterutil.IsIPv6(packet) {
			n = int(binary.BigEndian.Uint16(packet[4:6])) + 40
			if n > len(packet) {
				rem = packet
				logrus.Warnln("[lower] skip to send", len(packet), "bytes ipv6 packet head")
			} else {
				rem = packet[n:]
				logrus.Warnln("[lower] skip to send", n, "bytes ipv6 packet")
			}
			return
		}
		logrus.Warnln("[lower] skip to send", len(packet), "bytes non-ipv4/v6 packet")
		return len(packet), nil
	}
	totl := waterutil.IPv4TotalLength(packet)
	if int(totl) > len(packet) {
		buf := make([]byte, int(totl))
		copy(buf, packet)
		cnt, err := m.nic.Read(buf[len(packet):])
		if err != nil {
			rem = packet
			return
		}
		packet = buf[:cnt+len(packet)]
	}
	rem = packet[totl:]
	packet = packet[:totl]
	n = int(totl)
	dst := waterutil.IPv4Destination(packet)
	logrus.Debugln("[lower] sending", len(packet), "bytes packet from :"+strconv.Itoa(int(m.SrcPort())), "to", dst.String()+":"+strconv.Itoa(int(m.DstPort())))
	lnk, err := m.Connect(dst.String())
	if err != nil {
		logrus.Warnln("[lower] connect to peer", dst.String(), "err:", err)
		return
	}
	_, err = lnk.Write(head.NewPacket(head.ProtoData, m.SrcPort(), dst, m.DstPort(), packet), false)
	if err != nil {
		logrus.Warnln("[lower] write to peer", dst.String(), "err:", err)
	}
	return
}

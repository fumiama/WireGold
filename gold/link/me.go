package link

import (
	"encoding/binary"
	"encoding/hex"
	"io"
	"net"
	"strconv"
	"sync"
	"time"

	"github.com/FloatTech/ttl"
	"github.com/fumiama/orbyte"
	"github.com/fumiama/orbyte/pbuf"
	"github.com/fumiama/water/waterutil"
	"github.com/sirupsen/logrus"

	"github.com/fumiama/WireGold/config"
	"github.com/fumiama/WireGold/gold/head"
	"github.com/fumiama/WireGold/gold/p2p"
	"github.com/fumiama/WireGold/helper"
	"github.com/fumiama/WireGold/lower"
)

// Me 是本机的抽象
type Me struct {
	// 用于自我重连
	cfg *MyConfig
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
	ep p2p.EndPoint
	// 本机活跃的所有连接
	connections map[string]*Link
	// 读写同步锁
	connmapmu sync.RWMutex
	// 本机监听的连接端点, 也用于向对端直接发送报文
	conn p2p.Conn
	// 本机网卡
	nic *lower.NICIO
	// 本机路由表
	router *Router
	// 本机未接收完全分片池
	recving *ttl.Cache[uint64, *orbyte.Item[head.Packet]]
	// 抗重放攻击记录池
	recved *ttl.Cache[uint64, struct{}]
	// 本机上层配置
	srcport, dstport, mtu, speedloop uint16
	// 报头掩码
	mask uint64
	// 本机总接收字节数
	recvtotlcnt uint64
	// 上一次触发循环计数时间
	recvlooptime int64
	// 本机总接收数据包计数
	recvloopcnt uintptr
	// 是否进行 base16384 编码
	base14 bool
	// 本机网络端点初始化配置
	networkconfigs []any
}

type MyConfig struct {
	MyIPwithMask                     string
	MyEndpoint                       string
	Network                          string
	NetworkConfigs                   []any
	PrivateKey                       *[32]byte
	NICConfig                        *NICConfig
	SrcPort, DstPort, MTU, SpeedLoop uint16
	Mask                             uint64
	Base14                           bool
}

type NICConfig struct {
	IP     net.IP
	SubNet *net.IPNet
	CIDRs  []string
}

// NewMe 设置本机参数
func NewMe(cfg *MyConfig) (m Me) {
	m.cfg = cfg
	m.privKey = *cfg.PrivateKey
	var err error
	nw := cfg.Network
	if nw == "" {
		nw = "udp"
	}
	m.networkconfigs = cfg.NetworkConfigs
	m.ep, err = p2p.NewEndPoint(nw, cfg.MyEndpoint, m.networkconfigs...)
	if err != nil {
		panic(err)
	}
	ip, cidr, err := net.ParseCIDR(cfg.MyIPwithMask)
	if err != nil {
		panic(err)
	}
	m.me = ip
	m.subnet = *cidr
	m.speedloop = cfg.SpeedLoop
	if m.speedloop == 0 {
		m.speedloop = 4096
	}
	m.conn, err = m.listen()
	if err != nil {
		panic(err)
	}
	m.connections = make(map[string]*Link)
	m.router = &Router{
		list:  make([]*net.IPNet, 1, 16),
		table: make(map[string]*Link, 16),
		cache: ttl.NewCache[string, *Link](time.Minute),
	}
	m.router.SetDefault(nil)
	m.srcport = cfg.SrcPort
	m.dstport = cfg.DstPort
	m.mtu = (cfg.MTU - head.PacketHeadLen) & 0xfff8
	if cfg.NICConfig != nil {
		m.nic = lower.NewNIC(
			cfg.NICConfig.IP, cfg.NICConfig.SubNet,
			strconv.FormatUint(uint64(m.MTU()), 10), cfg.NICConfig.CIDRs...,
		)
	}
	m.mask = cfg.Mask
	m.recvlooptime = time.Now().UnixMilli()
	m.base14 = cfg.Base14
	var buf [8]byte
	binary.BigEndian.PutUint64(buf[:], m.mask)
	logrus.Infoln("[me] xor mask", hex.EncodeToString(buf[:]))
	m.recving = ttl.NewCache[uint64, *orbyte.Item[head.Packet]](time.Second * 10)
	m.recved = ttl.NewCache[uint64, struct{}](time.Minute)
	return
}

// Restart 重新连接
func (m *Me) Restart() error {
	oldconn := m.conn
	m.conn = nil
	if helper.IsNonNilInterface(oldconn) {
		_ = oldconn.Close()
	}
	var err error
	nw := m.cfg.Network
	if nw == "" {
		nw = "udp"
	}
	m.networkconfigs = m.cfg.NetworkConfigs
	m.ep, err = p2p.NewEndPoint(nw, m.cfg.MyEndpoint, m.networkconfigs...)
	if err != nil {
		return err
	}
	ip, cidr, err := net.ParseCIDR(m.cfg.MyIPwithMask)
	if err != nil {
		return err
	}
	m.me = ip
	m.subnet = *cidr
	m.recvlooptime = time.Now().UnixMilli()
	m.conn, err = m.listen()
	return err
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

func (m *Me) EndPoint() p2p.EndPoint {
	return m.ep
}

func (m *Me) Close() error {
	m.connections = nil
	if helper.IsNonNilInterface(m.conn) {
		_ = m.conn.Close()
		m.conn = nil
	}
	m.router = nil
	if m.recving != nil {
		m.recving.Destroy()
		m.recving = nil
	}
	if m.recved != nil {
		m.recved.Destroy()
		m.recved = nil
	}
	if m.nic != nil {
		m.nic.Down()
		return m.nic.Close()
	}
	return nil
}

func (m *Me) Write(packet []byte) (n int, err error) {
	n = m.sendAllSameDst(packet)
	if config.ShowDebugLog {
		logrus.Debugln("[me] writer ate", len(packet), "bytes, remain", len(packet)-n, "bytes")
	}
	return
}

func (m *Me) ListenNIC() (written int64, err error) {
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
				return 0
			}
			n += pktl
			rem = packet[n:]
			if config.ShowDebugLog {
				logrus.Debugln("[me] skip to send", len(packet), "bytes ipv6 packet")
			}
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
			if config.ShowDebugLog {
				logrus.Debugln("[me] wrap got invalid totl, break")
			}
			break
		}
		i += int(totl)
		ptr = rem[i:]
		if config.ShowDebugLog {
			logrus.Debugln("[me] wrap", totl, "bytes packet to send together")
		}
	}
	if i == 0 {
		return
	}
	n += i
	packet = rem[:i]
	rem = rem[i:]
	dst := waterutil.IPv4Destination(packet)
	if config.ShowDebugLog {
		logrus.Debugln("[me] sending", len(packet), "bytes packet from :"+strconv.Itoa(int(m.SrcPort())), "to", dst.String()+":"+strconv.Itoa(int(m.DstPort())), "remain:", len(rem), "bytes")
	}
	if m.me.Equal(dst) { // is to myself, write to nic (pipe not allow loopback)
		if config.ShowDebugLog {
			logrus.Debugln("[me] loopback packet")
		}
		_, err := m.nic.Write(packet)
		if err != nil {
			logrus.Warnln("[me] write to loopback err:", err)
		}
		return
	}
	lnk := m.router.NextHop(dst.String())
	if lnk == nil {
		logrus.Warnln("[me] drop packet to", dst.String()+":"+strconv.Itoa(int(m.DstPort())), ": nil nexthop")
		return
	}
	pcp := pbuf.NewBytes(len(packet))
	copy(pcp.Bytes(), packet)
	go func(packet pbuf.Bytes) {
		_, err := lnk.WritePacket(head.NewPacketPartial(head.ProtoData, m.SrcPort(), lnk.peerip, m.DstPort(), packet), false)
		if err != nil {
			logrus.Warnln("[me] write to peer", lnk.peerip, "err:", err)
		}
	}(pcp)
	return
}

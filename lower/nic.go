package lower

import (
	"encoding/binary"
	"os"
	"os/exec"
	"strconv"

	"github.com/fumiama/water"
	"github.com/fumiama/water/waterutil"
	"github.com/sirupsen/logrus"

	"github.com/fumiama/WireGold/gold/head"
	"github.com/fumiama/WireGold/gold/link"
)

// NIC 虚拟网卡
type NIC struct {
	ifce     *water.Interface
	ip       string
	subnet   string
	cidrs    []string
	hasstart bool
}

// NewNIC 新建 TUN 网络接口卡
// 网卡地址为 ip, 所属子网为 subnet
// 所有路由为 cidrs
func NewNIC(ip, subnet string, cidrs ...string) (n *NIC) {
	ifce, err := water.New(water.Config{DeviceType: water.TUN})
	if err != nil {
		panic(err)
	}
	n = &NIC{
		ifce:   ifce,
		ip:     ip,
		cidrs:  cidrs,
		subnet: subnet,
	}
	n.prepare()
	return
}

// Start 开始处理网卡消息，阻塞
func (nc *NIC) Start(m *link.Me) {
	if nc.hasstart {
		return
	}
	nc.hasstart = true
	go func() { // 接收到 NIC
		for nc.hasstart {
			packet := m.Read()
			n, err := nc.ifce.Write(packet.Data)
			if err != nil {
				logrus.Errorln("[lower] recv write to nic err:", err)
				break
			}
			logrus.Infoln("[lower] recv write", n, "bytes packet to nic")
		}
	}()
	buf := make([]byte, m.MTU()+64) // 增加报头长度与 TEA 冗余
	off := 0
	for nc.hasstart { // 从 NIC 发送
		packet := buf[off:]
		n, err := nc.ifce.Read(packet)
		if err != nil {
			logrus.Errorln("[lower] send read from nic err:", err)
			break
		}
		if n == 0 {
			continue
		}
		packet = packet[:n]
		_, rem := send(m, packet)
		for len(rem) > 20 {
			_, rem = send(m, rem)
		}
		if len(rem) > 0 {
			off = copy(buf, rem)
		}
	}
}

// Stop 停止处理
func (n *NIC) Stop() {
	n.hasstart = false
}

// Destroy 关闭网卡
func (n *NIC) Destroy() error {
	return n.ifce.Close()
}

func execute(c string, args ...string) {
	logrus.Printf("[lower] exec cmd: %v %v:", c, args)
	cmd := exec.Command(c, args...)
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	cmd.Stdin = os.Stdin
	err := cmd.Run()
	if err != nil {
		logrus.Panicln("[lower] failed to exec cmd:", err)
	}
}

func send(m *link.Me, packet []byte) (n int, rem []byte) {
	if !waterutil.IsIPv4(packet) {
		if waterutil.IsIPv6(packet) {
			n = int(binary.BigEndian.Uint16(packet[4:6])) + 40
			rem = packet[n:]
			logrus.Warnln("[lower] skip to send", n, "bytes ipv6 packet")
			return
		}
		logrus.Warnln("[lower] skip to send", len(packet), "bytes non-ipv4/v6 packet")
		return len(packet), nil
	}
	totl := waterutil.IPv4TotalLength(packet)
	rem = packet[totl:]
	packet = packet[:totl]
	n = int(totl)
	dst := waterutil.IPv4Destination(packet)
	logrus.Infoln("[lower] sending", len(packet), "bytes packet from :"+strconv.Itoa(int(m.SrcPort())), "to", dst.String()+":"+strconv.Itoa(int(m.DstPort())))
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

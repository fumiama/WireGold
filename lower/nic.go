package lower

import (
	"os"
	"os/exec"

	"github.com/sirupsen/logrus"
	"github.com/songgao/water"
	"github.com/songgao/water/waterutil"

	"github.com/fumiama/WireGold/gold/head"
	"github.com/fumiama/WireGold/gold/link"
)

// NIC 虚拟网卡
type NIC struct {
	ifce     *water.Interface
	ip       string
	subnet   string
	hasstart bool
}

// NewNIC 新建 TUN 网络接口卡
// 网卡地址为 ip, 所属子网为 subnet
func NewNIC(ip, subnet string) (n *NIC) {
	ifce, err := water.New(water.Config{DeviceType: water.TUN})
	if err != nil {
		panic(err)
	}
	n = &NIC{
		ifce:   ifce,
		ip:     ip,
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
			logrus.Infoln("[lower] recv", len(packet.Data), "bytes packet")
			if !waterutil.IsIPv4(packet.Data) {
				logrus.Warnln("[lower] recv recv non-ipv4 packet")
				continue
			}
			_, err := nc.ifce.Write(packet.Data)
			if err != nil {
				logrus.Errorln("[lower] recv write to nic err:", err)
				break
			}
		}
	}()
	buf := make([]byte, 4096)
	for nc.hasstart { // 从 NIC 发送
		packet := buf
		n, err := nc.ifce.Read(packet)
		if err != nil {
			logrus.Errorln("[lower] send read from nic err:", err)
			break
		}
		if n == 0 {
			continue
		}
		packet = packet[:n]
		if !waterutil.IsIPv4(packet) {
			logrus.Warnln("[lower] send recv non-ipv4 packet")
			continue
		}
		logrus.Infoln("[lower] send", n, "bytes packet")
		dst := waterutil.IPv4Destination(packet)
		srcport := waterutil.IPv4SourcePort(packet)
		dstport := waterutil.IPv4DestinationPort(packet)
		lnk, err := m.Connect(dst.String())
		if err != nil {
			logrus.Errorln("[lower] connect to peer err:", err)
			continue
		}
		lnk.Write(head.NewPacket(head.ProtoData, srcport, dstport, packet))
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

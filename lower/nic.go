package lower

import (
	"net"
	"os"
	"os/exec"
	"strconv"

	"github.com/fumiama/water"
	"github.com/sirupsen/logrus"
)

// NICIO 虚拟网卡
type NICIO struct {
	ifce     *water.Interface
	ip       net.IP
	subnet   *net.IPNet
	rawipnet string
	mtu      string
	cidrs    []string
}

// NewNIC 新建 TUN 网络接口卡
// 网卡地址为 ip, 所属子网为 subnet
// 以本网卡为下一跳的所有子网为 cidrs
// cidrs 不包括本网卡 subnet
func NewNIC(ip net.IP, subnet *net.IPNet, mtu string, cidrs ...string) *NICIO {
	ifce, err := water.New(water.Config{DeviceType: water.TUN})
	if err != nil {
		logrus.Error(err)
		os.Exit(1)
	}
	subn, bitsn := subnet.Mask.Size()
	if bitsn != 32 {
		panic("mask len " + strconv.Itoa(bitsn) + " is not supported")
	}
	n := &NICIO{
		ifce:     ifce,
		ip:       ip,
		subnet:   subnet,
		rawipnet: ip.String() + "/" + strconv.Itoa(subn),
		mtu:      mtu,
		cidrs:    cidrs,
	}
	return n
}

// Read 匹配 PacketsIO Interface
func (nc *NICIO) Read(buf []byte) (int, error) {
	return nc.ifce.Read(buf)
}

func (nc *NICIO) Write(packet []byte) (int, error) {
	return nc.ifce.Write(packet)
}

// Close 关闭网卡
func (n *NICIO) Close() error {
	return n.ifce.Close()
}

// nolint: unparam
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

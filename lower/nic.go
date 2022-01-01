package lower

import (
	"io"
	"os"
	"os/exec"

	"github.com/fumiama/water"
	"github.com/sirupsen/logrus"
)

type NICIO interface {
	io.ReadWriteCloser
	Up()
	Down()
}

// NIC 虚拟网卡
type NIC struct {
	ifce   *water.Interface
	ip     string
	subnet string
	mtu    string
	cidrs  []string
}

// NewNIC 新建 TUN 网络接口卡
// 网卡地址为 ip, 所属子网为 subnet
// 以本网卡为下一跳的所有子网为 cidrs
// cidrs 不包括本网卡 subnet
func NewNIC(ip, subnet, mtu string, cidrs ...string) NICIO {
	ifce, err := water.New(water.Config{DeviceType: water.TUN})
	if err != nil {
		panic(err)
	}
	n := &NIC{
		ifce:   ifce,
		ip:     ip,
		subnet: subnet,
		mtu:    mtu,
		cidrs:  cidrs,
	}
	return n
}

// Read 匹配 PacketsIO Interface
func (nc *NIC) Read(buf []byte) (int, error) {
	return nc.ifce.Read(buf)
}

func (nc *NIC) Write(packet []byte) (int, error) {
	return nc.ifce.Write(packet)
}

// Close 关闭网卡
func (n *NIC) Close() error {
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

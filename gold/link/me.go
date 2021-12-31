package link

import (
	"net"
	"sync"

	"github.com/fumiama/WireGold/gold/head"
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
	// 不分目的 link 的接收队列
	pipe chan []byte
	// 本机路由表
	router *Router
	// 本机未接收完全分片池
	recving map[[32]byte]*head.Packet
	recvmu  sync.Mutex
	// 超时定时器
	clock map[*head.Packet]uint8
	// 本机上层配置
	srcport, dstport, mtu uint16
	readptr               []byte
}

// NewMe 设置本机参数
func NewMe(privateKey *[32]byte, myipwithmask string, myEndpoint string, nopipeinlink bool, srcport, dstport, mtu uint16) (m Me) {
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
	if nopipeinlink {
		m.pipe = make(chan []byte, 32)
	}
	m.router = &Router{
		list:  make([]*net.IPNet, 1, 16),
		table: make(map[string]*Link, 16),
	}
	m.router.SetDefault(nil)
	m.loop = m.AddPeer(m.me.String(), nil, "127.0.0.1:56789", []string{myipwithmask}, 0, false, nopipeinlink)
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

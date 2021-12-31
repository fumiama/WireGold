package upper

import "io"

// 常用服务端口
const (
	// ServiceNull 不在意端口号的服务
	ServiceNull = iota
	// ServiceTunnel 管道通信服务
	ServiceTunnel
	// ServiceWireGold 虚拟组网服务
	ServiceWireGold
)

type Service interface {
	// Start 无阻塞运行
	Start(srcport, destport, mtu uint16)
	// Run 阻塞运行
	Run(srcport, destport, mtu uint16)
	// Stop 停止
	Stop()
	io.ReadWriter
}

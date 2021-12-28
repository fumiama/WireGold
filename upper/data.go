package upper

import "io"

// 常用服务端口
const (
	// ServiceNull 不在意端口号的服务
	ServiceNull = iota
	// ServiceTunnel 管道通信服务
	ServiceTunnel
)

type Service interface {
	Create(peer string, srcport, destport, mtu uint16) (Service, error)
	io.ReadWriteCloser
}

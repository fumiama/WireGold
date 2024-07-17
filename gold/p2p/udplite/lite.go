//go:build !darwin

package udplite

import (
	"context"
	"net"
	"syscall"
	"unsafe"

	"github.com/fumiama/WireGold/gold/head"
)

// https://www.kernel.org/doc/Documentation/networking/udplite.txt
const (
	IPPROTO_UDPLITE    = 136
	SOL_UDPLITE        = 136
	UDPLITE_SEND_CSCOV = 10
	UDPLITE_RECV_CSCOV = 11
)

type sysListener struct {
	net.ListenConfig
	network, address string
}

type sockaddr interface {
	net.Addr
}

//go:linkname toLocal net.(*UDPAddr).toLocal
func toLocal(a *net.UDPAddr, net string) sockaddr

//go:linkname internetSocket net.internetSocket
func internetSocket(ctx context.Context, net string, laddr, raddr sockaddr, sotype, proto int, mode string, ctrlCtxFn func(context.Context, string, string, syscall.RawConn) error) (fd unsafe.Pointer, err error)

//go:linkname newUDPConn net.newUDPConn
func newUDPConn(fd unsafe.Pointer) *net.UDPConn

var sockaddrinterfaceinstance = toLocal(&net.UDPAddr{}, "")

func (sl *sysListener) listenUDP(ctx context.Context, laddr *net.UDPAddr) (*net.UDPConn, error) {
	var ctrlCtxFn func(cxt context.Context, network, address string, c syscall.RawConn) error
	if sl.ListenConfig.Control != nil {
		ctrlCtxFn = func(cxt context.Context, network, address string, c syscall.RawConn) error {
			return sl.ListenConfig.Control(network, address, c)
		}
	}
	sockladdr := sockaddrinterfaceinstance
	*(**net.UDPAddr)(unsafe.Add(unsafe.Pointer(&sockladdr), unsafe.Sizeof(uintptr(0)))) = laddr
	sockraddr := sockaddrinterfaceinstance
	sockladdr = nil
	fd, err := internetSocket(ctx, sl.network, sockladdr, sockraddr, syscall.SOCK_DGRAM, IPPROTO_UDPLITE, "listen", ctrlCtxFn)
	if err != nil {
		return nil, err
	}
	return newUDPConn(fd), nil
}

func listenUDPLite(network string, laddr *net.UDPAddr) (*net.UDPConn, error) {
	if laddr == nil {
		laddr = &net.UDPAddr{}
	}
	sl := &sysListener{network: network, address: laddr.String()}
	conn, err := sl.listenUDP(context.Background(), laddr)
	if err != nil {
		var laddrgeneral net.Addr
		if laddr != nil {
			laddrgeneral = laddr
		}
		return nil, &net.OpError{Op: "listen", Net: network, Source: nil, Addr: laddrgeneral, Err: err}
	}
	rc, err := conn.SyscallConn()
	if err != nil {
		_ = conn.Close()
		return nil, err
	}
	var errsys error
	err = rc.Control(func(fd uintptr) {
		errsys = syscall.SetsockoptInt(int(fd), SOL_UDPLITE, UDPLITE_SEND_CSCOV, head.PacketHeadLen)
		if errsys != nil {
			return
		}
		errsys = syscall.SetsockoptInt(int(fd), SOL_UDPLITE, UDPLITE_RECV_CSCOV, head.PacketHeadLen)
	})
	if err != nil {
		_ = conn.Close()
		return nil, err
	}
	if errsys != nil {
		_ = conn.Close()
		return nil, errsys
	}
	return conn, nil
}

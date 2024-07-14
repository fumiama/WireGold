package p2p

import (
	"errors"
	"fmt"
	"io"

	"github.com/RomiChan/syncx"
)

type Initializer func(endpoint string, configs ...any) EndPoint

var factory syncx.Map[string, Initializer]

func Register(network string, initializer Initializer) (actual Initializer, hasexist bool) {
	return factory.LoadOrStore(network, initializer)
}

type EndPoint interface {
	fmt.Stringer
	Network() string
	Euqal(EndPoint) bool
	Listen() (Conn, error)
}

func NewEndPoint(network, endpoint string, configs ...any) (EndPoint, error) {
	initializer, ok := factory.Load(network)
	if !ok {
		return nil, errors.New("network " + network + " not found")
	}
	return initializer(endpoint, configs...), nil
}

type Conn interface {
	io.Closer
	fmt.Stringer // basically, the local address string
	LocalAddr() EndPoint
	ReadFromPeer([]byte) (int, EndPoint, error)
	WriteToPeer([]byte, EndPoint) (int, error)
}

package upper

import "io"

type Service interface {
	Create(peer string, srcport uint16, destport uint16) (Service, error)
	io.ReadWriteCloser
}

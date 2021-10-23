package upper

import "io"

type Service interface {
	io.ReadWriteCloser
}

package helper

import (
	"github.com/fumiama/orbyte/pbuf"
)

// SelectWriter 从池中取出一个 Writer
func SelectWriter() *Writer {
	return (*Writer)(pbuf.NewBuffer(nil))
}

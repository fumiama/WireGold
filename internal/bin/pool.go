package bin

import (
	"github.com/fumiama/orbyte/pbuf"
)

// SelectWriter 从池中取出一个 Writer
//
// 不要忘记调用 Destroy 以快速回收资源
func SelectWriter() *Writer {
	return (*Writer)(pbuf.NewBuffer(nil))
}

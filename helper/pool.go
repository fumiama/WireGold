package helper

import (
	"bytes"
	"sync"
)

// https://github.com/Mrs4s/MiraiGo/blob/master/binary/pool.go

var bufferPool = sync.Pool{
	New: func() interface{} {
		return new(Writer)
	},
}

// SelectWriter 从池中取出一个 Writer
func SelectWriter() *Writer {
	// 因为 bufferPool 定义有 New 函数
	// 所以 bufferPool.Get() 永不为 nil
	// 不用判空
	return bufferPool.Get().(*Writer)
}

// PutWriter 将 Writer 放回池中
func PutWriter(w *Writer) {
	// See https://golang.org/issue/23199
	const maxSize = 1 << 16
	if (*bytes.Buffer)(w).Cap() < maxSize { // 对于大Buffer直接丢弃
		w.Reset()
		bufferPool.Put(w)
	}
}

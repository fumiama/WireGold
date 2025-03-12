package bin

// https://github.com/Mrs4s/MiraiGo/blob/master/binary/writer.go

import (
	"encoding/binary"

	"github.com/fumiama/orbyte/pbuf"
)

// Writer 写入
type Writer pbuf.OBuffer

func NewWriterF(f func(writer *Writer)) pbuf.Bytes {
	w := SelectWriter()
	f(w)
	return w.ToBytes()
}

func (w *Writer) P(f func(*pbuf.Buffer)) *Writer {
	(*pbuf.OBuffer)(w).P(f)
	return w
}

func (w *Writer) Write(b []byte) (n int, err error) {
	w.P(func(buf *pbuf.Buffer) {
		n, err = buf.Write(b)
	})
	return
}

func (w *Writer) WriteByte(b byte) (err error) {
	w.P(func(buf *pbuf.Buffer) {
		err = buf.WriteByte(b)
	})
	return
}

func (w *Writer) WriteString(s string) (n int, err error) {
	w.P(func(buf *pbuf.Buffer) {
		n, err = buf.WriteString(s)
	})
	return
}

func (w *Writer) WriteUInt16(v uint16) {
	b := make([]byte, 2)
	binary.LittleEndian.PutUint16(b, v)
	w.Write(b)
}

func (w *Writer) WriteUInt32(v uint32) {
	b := make([]byte, 4)
	binary.LittleEndian.PutUint32(b, v)
	w.Write(b)
}

func (w *Writer) WriteUInt64(v uint64) {
	b := make([]byte, 8)
	binary.LittleEndian.PutUint64(b, v)
	w.Write(b)
}

func (w *Writer) ToBytes() pbuf.Bytes {
	return pbuf.BufferItemToBytes((*pbuf.OBuffer)(w))
}

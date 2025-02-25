package helper

// https://github.com/Mrs4s/MiraiGo/blob/master/binary/writer.go

import (
	"bytes"
	"encoding/binary"

	"github.com/fumiama/orbyte"
	"github.com/fumiama/orbyte/pbuf"
)

// Writer 写入
type Writer orbyte.Item[bytes.Buffer]

func NewWriterF(f func(writer *Writer)) pbuf.Bytes {
	w := SelectWriter()
	f(w)
	return w.TransBytes()
}

func (w *Writer) p() *orbyte.Item[bytes.Buffer] {
	return (*orbyte.Item[bytes.Buffer])(w)
}

func (w *Writer) pp() *bytes.Buffer {
	return w.p().Pointer()
}

func (w *Writer) Write(b []byte) (n int, err error) {
	return w.pp().Write(b)
}

func (w *Writer) WriteByte(b byte) error {
	return w.pp().WriteByte(b)
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

func (w *Writer) WriteString(v string) {
	//w.WriteUInt32(uint32(len(v) + 4))
	w.pp().WriteString(v)
}

func (w *Writer) Len() int {
	return w.pp().Len()
}

func (w *Writer) UnsafeBytes() []byte {
	return w.pp().Bytes()
}

func (w *Writer) TransUnderlyingBytes() []byte {
	return w.p().Trans().Pointer().Bytes()
}

func (w *Writer) TransBytes() pbuf.Bytes {
	return pbuf.BufferItemToBytes(w.p().Trans())
}

func (w *Writer) Reset() {
	w.pp().Reset()
}

func (w *Writer) Grow(n int) {
	w.pp().Grow(n)
}

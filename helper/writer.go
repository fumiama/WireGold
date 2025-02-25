package helper

// https://github.com/Mrs4s/MiraiGo/blob/master/binary/writer.go

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"

	"github.com/fumiama/orbyte"
	"github.com/fumiama/orbyte/pbuf"
)

// Writer 写入
type Writer orbyte.Item[bytes.Buffer]

func NewWriterF(f func(writer *Writer)) pbuf.Bytes {
	w := SelectWriter()
	f(w)
	return pbuf.BufferItemToBytes((*orbyte.Item[bytes.Buffer])(w).Trans())
}

func (w *Writer) Write(b []byte) (n int, err error) {
	return (*orbyte.Item[bytes.Buffer])(w).Pointer().Write(b)
}

func (w *Writer) WriteHex(h string) {
	b, _ := hex.DecodeString(h)
	w.Write(b)
}

func (w *Writer) WriteByte(b byte) error {
	return (*orbyte.Item[bytes.Buffer])(w).Pointer().WriteByte(b)
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
	(*orbyte.Item[bytes.Buffer])(w).Pointer().WriteString(v)
}

func (w *Writer) Len() int {
	return (*orbyte.Item[bytes.Buffer])(w).Pointer().Len()
}

func (w *Writer) UnsafeBytes() []byte {
	return (*orbyte.Item[bytes.Buffer])(w).Pointer().Bytes()
}

func (w *Writer) TransBytes() []byte {
	return (*orbyte.Item[bytes.Buffer])(w).Trans().Pointer().Bytes()
}

func (w *Writer) Reset() {
	(*orbyte.Item[bytes.Buffer])(w).Pointer().Reset()
}

func (w *Writer) Grow(n int) {
	(*orbyte.Item[bytes.Buffer])(w).Pointer().Grow(n)
}

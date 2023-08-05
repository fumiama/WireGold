package helper

// https://github.com/Mrs4s/MiraiGo/blob/master/binary/writer.go

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"io"
	"unsafe"
)

// Writer 写入
type Writer bytes.Buffer

func NewWriterF(f func(writer *Writer)) []byte {
	w := SelectWriter()
	f(w)
	b := append([]byte(nil), w.Bytes()...)
	w.put()
	return b
}

// OpenWriterF must call func cl to close
func OpenWriterF(f func(*Writer)) (b []byte, cl func()) {
	w := SelectWriter()
	f(w)
	return w.Bytes(), w.put
}

func (w *Writer) FillUInt16() (pos int) {
	pos = w.Len()
	(*bytes.Buffer)(w).Write([]byte{0, 0})
	return
}

func (w *Writer) WriteUInt16At(pos int, v uint16) {
	newdata := (*bytes.Buffer)(w).Bytes()[pos:]
	binary.LittleEndian.PutUint16(newdata, v)
}

func (w *Writer) FillUInt32() (pos int) {
	pos = w.Len()
	(*bytes.Buffer)(w).Write([]byte{0, 0, 0, 0})
	return
}

func (w *Writer) WriteUInt32At(pos int, v uint32) {
	newdata := (*bytes.Buffer)(w).Bytes()[pos:]
	binary.LittleEndian.PutUint32(newdata, v)
}

func (w *Writer) Write(b []byte) (n int, err error) {
	return (*bytes.Buffer)(w).Write(b)
}

func (w *Writer) WriteHex(h string) {
	b, _ := hex.DecodeString(h)
	w.Write(b)
}

func (w *Writer) WriteByte(b byte) error {
	return (*bytes.Buffer)(w).WriteByte(b)
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
	(*bytes.Buffer)(w).WriteString(v)
}

func (w *Writer) WriteStringShort(v string) {
	w.WriteUInt16(uint16(len(v)))
	(*bytes.Buffer)(w).WriteString(v)
}

func (w *Writer) WriteBool(b bool) {
	if b {
		w.WriteByte(0x01)
	} else {
		w.WriteByte(0x00)
	}
}

func (w *Writer) WriteBytesShort(data []byte) {
	w.WriteUInt16(uint16(len(data)))
	w.Write(data)
}

func (w *Writer) Len() int {
	return (*bytes.Buffer)(w).Len()
}

func (w *Writer) Bytes() []byte {
	return (*bytes.Buffer)(w).Bytes()
}

func (w *Writer) Reset() {
	(*bytes.Buffer)(w).Reset()
}

func (w *Writer) Grow(n int) {
	(*bytes.Buffer)(w).Grow(n)
}

func (w *Writer) Skip(n int) (int, error) {
	b := (*buffer)(unsafe.Pointer(w))
	b.lastRead = opInvalid
	if len(b.buf) <= b.off {
		// Buffer is empty, reset to recover space.
		w.Reset()
		if n == 0 {
			return 0, nil
		}
		return 0, io.EOF
	}
	n = min(n, len(b.buf[b.off:]))
	b.off += n
	if n > 0 {
		b.lastRead = opRead
	}
	return n, nil
}

func (w *Writer) put() {
	PutWriter(w)
}

// A Buffer is a variable-sized buffer of bytes with Read and Write methods.
// The zero value for Buffer is an empty buffer ready to use.
type buffer struct {
	buf      []byte // contents are the bytes buf[off : len(buf)]
	off      int    // read at &buf[off], write at &buf[len(buf)]
	lastRead readOp // last read operation, so that Unread* can work correctly.
}

// The readOp constants describe the last action performed on
// the buffer, so that UnreadRune and UnreadByte can check for
// invalid usage. opReadRuneX constants are chosen such that
// converted to int they correspond to the rune size that was read.
type readOp int8

// Don't use iota for these, as the values need to correspond with the
// names and comments, which is easier to see when being explicit.
const (
	opRead      readOp = -1 // Any other read operation.
	opInvalid   readOp = 0  // Non-read operation.
	opReadRune1 readOp = 1  // Read rune of size 1.
	opReadRune2 readOp = 2  // Read rune of size 2.
	opReadRune3 readOp = 3  // Read rune of size 3.
	opReadRune4 readOp = 4  // Read rune of size 4.
)

// min 返回两数最小值，该函数将被内联
func min[T int | int8 | uint8 | int16 | uint16 | int32 | uint32 | int64 | uint64](a, b T) T {
	if a > b {
		return b
	}
	return a
}

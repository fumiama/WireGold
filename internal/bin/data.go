package bin

import (
	"encoding/binary"
	"reflect"
	"unsafe"
)

// IsLittleEndian judge by binary packet.
var IsLittleEndian = reflect.ValueOf(&binary.NativeEndian).Elem().Field(0).Type().String() == "binary.littleEndian"

// slice is the runtime representation of a slice.
// It cannot be used safely or portably and its representation may
// change in a later release.
//
// Unlike reflect.SliceHeader, its Data field is sufficient to guarantee the
// data it references will not be garbage collected.
type slice struct {
	data unsafe.Pointer
	len  int
	cap  int
}

// BytesToString 没有内存开销的转换
func BytesToString(b []byte) string {
	return *(*string)(unsafe.Pointer(&b))
}

// StringToBytes 没有内存开销的转换
func StringToBytes(s string) (b []byte) {
	bh := (*slice)(unsafe.Pointer(&b))
	sh := (*slice)(unsafe.Pointer(&s)) // 不要访问 sh.cap
	bh.data = sh.data
	bh.len = sh.len
	bh.cap = sh.len
	return b
}

func IsNilInterface(x any) bool {
	return x == nil || reflect.ValueOf(x).IsZero()
}

func IsNonNilInterface(x any) bool {
	return !IsNilInterface(x)
}

package head

import (
	"unsafe"
)

const (
	ttloffset = unsafe.Offsetof(Packet{}.TTL)
)

// ClearTTL for hash use
func ClearTTL(data []byte) {
	data[ttloffset] = 0
}

// DecTTL on transferring
func DecTTL(data []byte) (drop bool) {
	data[ttloffset]--
	return data[ttloffset] == 0
}

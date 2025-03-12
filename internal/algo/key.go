package algo

import (
	"encoding/binary"
	"math/bits"
	"math/rand"
)

func RandKeyIndex() uint8 {
	return uint8(rand.Intn(32))
}

func MixKeys(k1, k2 []byte) []byte {
	if len(k1) != 32 || len(k2) != 32 {
		panic("unexpected key len")
	}
	k := make([]byte, 64)
	for i := range k1 {
		k1i, k2i := i, 31-i
		k1v, k2v := k1[k1i], k2[k2i]
		binary.LittleEndian.PutUint16(
			k[i*2:(i+1)*2],
			expandkeyunit(k1v, k2v),
		)
	}
	return k
}

func expandkeyunit(v1, v2 byte) (v uint16) {
	v1s, v2s := uint16(v1), uint16(bits.Reverse8(v2))
	for i := 0; i < 8; i++ {
		v |= v1s & (1 << (i * 2))
		v1s <<= 1
	}
	for i := 0; i < 8; i++ {
		v2s <<= 1
		v |= v2s & (2 << (i * 2))
	}
	return
}

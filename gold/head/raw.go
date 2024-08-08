package head

import (
	"crypto/md5"
	"encoding/binary"
)

// CRC64 extract packet header checksum
func CRC64(data []byte) uint64 {
	return binary.LittleEndian.Uint64(data[52:PacketHeadLen])
}

// CalcCRC64 calculate packet header checksum
func CalcCRC64(data []byte) uint64 {
	m := md5.Sum(data[:52])
	return binary.LittleEndian.Uint64(m[:8])
}

// Hash extract 32 bytes blake2b hash from raw bytes
func Hash(data []byte) []byte {
	return data[20:52]
}

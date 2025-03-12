package algo

import (
	"crypto/cipher"
	"crypto/rand"
	"encoding/binary"
	"errors"

	"github.com/fumiama/orbyte/pbuf"
)

var (
	ErrCipherTextTooShort = errors.New("ciphertext too short")
)

func EncodeAEAD(aead cipher.AEAD, additional uint16, b []byte) pbuf.Bytes {
	nsz := aead.NonceSize()
	// Accocate capacity for all the stuffs.
	buf := pbuf.NewBytes(2 + nsz + len(b) + aead.Overhead())
	n := 0
	buf.V(func(buf []byte) {
		binary.LittleEndian.PutUint16(buf[:2], additional)
		nonce := buf[2 : 2+nsz]
		// Select a random nonce
		_, err := rand.Read(nonce)
		if err != nil {
			panic(err)
		}
		// Encrypt the message and append the ciphertext to the nonce.
		eb := aead.Seal(nonce[nsz:nsz], nonce, b, buf[:2])
		n = len(eb)
	})
	return buf.Slice(2, 2+nsz+n)
}

func DecodeAEAD(aead cipher.AEAD, additional uint16, b []byte) (data pbuf.Bytes, err error) {
	nsz := aead.NonceSize()
	if len(b) < nsz {
		return pbuf.Bytes{}, ErrCipherTextTooShort
	}
	// Split nonce and ciphertext.
	nonce, ciphertext := b[:nsz], b[nsz:]
	if len(ciphertext) == 0 {
		return pbuf.Bytes{}, nil
	}
	// Decrypt the message and check it wasn't tampered with.
	var buf [2]byte
	binary.LittleEndian.PutUint16(buf[:], additional)
	data = pbuf.NewBytes(len(ciphertext))
	n := 0
	data.V(func(b []byte) {
		var d []byte
		d, err = aead.Open(b[:0], nonce, ciphertext, buf[:])
		n = len(d)
	})
	if err != nil {
		return
	}
	return data.SliceTo(n), nil
}

func EncodeXORLen(datalen int) int {
	batchsz := datalen / 8
	return 8 + batchsz*8 + 8 // seqrand dat tail
}

// EncodeXOR 按 8 字节, 以初始 mask 循环异或编码 data
func EncodeXOR(data []byte, mask uint64, seq uint32) pbuf.Bytes {
	batchsz := len(data) / 8
	remain := len(data) % 8
	sum := mask
	newdat := pbuf.NewBytes(EncodeXORLen(len(data)))
	newdat.V(func(buf []byte) {
		binary.LittleEndian.PutUint32(buf[:4], seq)
		_, _ = rand.Read(buf[4:8])                 // seqrand
		sum ^= binary.LittleEndian.Uint64(buf[:8]) // init from seqrand
		binary.LittleEndian.PutUint64(buf[:8], sum)
		for i := 0; i < batchsz; i++ { // range on batch data
			a := i * 8
			b := (i + 1) * 8
			sum ^= binary.LittleEndian.Uint64(data[a:b])
			binary.LittleEndian.PutUint64(buf[a+8:b+8], sum)
		}
		p := batchsz * 8
		copy(buf[8+p:], data[p:])
		buf[newdat.Len()-1] = byte(remain)
		sum ^= binary.LittleEndian.Uint64(buf[8+p:])
		binary.LittleEndian.PutUint64(buf[8+p:], sum)
	})
	return newdat
}

// DecodeXOR 按 8 字节, 以初始 mask 循环异或解码 data,
// 解码结果完全覆盖 data.
func DecodeXOR(data []byte, mask uint64) (uint32, []byte) {
	if len(data) < 16 || len(data)%8 != 0 {
		return 0, nil
	}
	batchsz := len(data) / 8
	sum := mask
	for i := 0; i < batchsz; i++ {
		a := i * 8
		b := (i + 1) * 8
		x := binary.LittleEndian.Uint64(data[a:b])
		sum ^= x
		binary.LittleEndian.PutUint64(data[a:b], sum)
		sum = x
	}
	remain := data[len(data)-1]
	if remain >= 8 {
		return 0, nil
	}
	return binary.LittleEndian.Uint32(data[:4]),
		data[8 : len(data)-8+int(remain)]
}

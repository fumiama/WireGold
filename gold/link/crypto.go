package link

import (
	"crypto/cipher"
	"crypto/rand"
	"encoding/binary"
	"errors"
	"math/bits"
	mrand "math/rand"

	"github.com/fumiama/WireGold/helper"
	"github.com/sirupsen/logrus"
)

var (
	ErrCipherTextTooShort = errors.New("ciphertext too short")
)

func (l *Link) randkeyidx() uint8 {
	if l.keys[1] == nil {
		return 0
	}
	return uint8(mrand.Intn(32))
}

func mixkeys(k1, k2 []byte) []byte {
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

// Encode by aead and put b into pool
func (l *Link) Encode(teatype uint8, additional uint16, b []byte) (eb []byte) {
	if len(b) == 0 || teatype >= 32 {
		return
	}
	if l.keys[0] == nil {
		eb = helper.MakeBytes(len(b))
		copy(eb, b)
		return
	}
	aead := l.keys[teatype]
	if aead == nil {
		logrus.Warnln("[crypto] cipher key at index", teatype, "is empty")
		return
	}
	eb = encode(aead, additional, b)
	return
}

// Decode by aead and put b into pool
func (l *Link) Decode(teatype uint8, additional uint16, b []byte) (db []byte, err error) {
	if len(b) == 0 || teatype >= 32 {
		return
	}
	if l.keys[0] == nil {
		db = helper.MakeBytes(len(b))
		copy(db, b)
		return
	}
	aead := l.keys[teatype]
	if aead == nil {
		return
	}
	return decode(aead, additional, b)
}

func encode(aead cipher.AEAD, additional uint16, b []byte) []byte {
	nsz := aead.NonceSize()
	// Accocate capacity for all the stuffs.
	buf := helper.MakeBytes(2 + nsz + len(b) + aead.Overhead())
	binary.LittleEndian.PutUint16(buf[:2], additional)
	nonce := buf[2 : 2+nsz]
	// Select a random nonce
	_, err := rand.Read(nonce)
	if err != nil {
		panic(err)
	}
	// Encrypt the message and append the ciphertext to the nonce.
	eb := aead.Seal(nonce[nsz:nsz], nonce, b, buf[:2])
	return nonce[:nsz+len(eb)]
}

func decode(aead cipher.AEAD, additional uint16, b []byte) ([]byte, error) {
	nsz := aead.NonceSize()
	if len(b) < nsz {
		return nil, ErrCipherTextTooShort
	}
	// Split nonce and ciphertext.
	nonce, ciphertext := b[:nsz], b[nsz:]
	if len(ciphertext) == 0 {
		return nil, nil
	}
	// Decrypt the message and check it wasn't tampered with.
	var buf [2]byte
	binary.LittleEndian.PutUint16(buf[:], additional)
	return aead.Open(helper.SelectWriter().Bytes(), nonce, ciphertext, buf[:])
}

// xorenc 按 8 字节, 以初始 m.mask 循环异或编码 data
func (m *Me) xorenc(data []byte, seq uint32) []byte {
	batchsz := len(data) / 8
	remain := len(data) % 8
	sum := m.mask
	newdat := helper.MakeBytes(len(data) + 8)
	binary.LittleEndian.PutUint32(newdat[:4], seq)
	_, _ = rand.Read(newdat[4:8])
	if remain > 0 {
		var buf [8]byte
		p := batchsz * 8
		copy(buf[:], data[p:])
		sum ^= binary.LittleEndian.Uint64(buf[:])
		binary.LittleEndian.PutUint64(buf[:], sum)
		copy(newdat[8+p:], buf[:])
	}
	for i := batchsz - 1; i >= 0; i-- {
		a := i * 8
		b := (i + 1) * 8
		sum ^= binary.LittleEndian.Uint64(data[a:b])
		binary.LittleEndian.PutUint64(newdat[a+8:b+8], sum)
	}
	sum ^= binary.LittleEndian.Uint64(newdat[:8])
	binary.LittleEndian.PutUint64(newdat[:8], sum)
	return newdat
}

// xordec 按 8 字节, 以初始 m.mask 循环异或解码 data
func (m *Me) xordec(data []byte) (uint32, []byte) {
	if len(data) <= 8 {
		return 0, nil
	}
	batchsz := len(data) / 8
	remain := len(data) % 8
	this := uint64(0)
	next := uint64(0)
	if len(data) >= 8 {
		next = binary.LittleEndian.Uint64(data[:8])
	}
	for i := 0; i < batchsz-1; i++ {
		a := i * 8
		b := (i + 1) * 8
		this = next
		next = binary.LittleEndian.Uint64(data[a+8 : b+8])
		binary.LittleEndian.PutUint64(data[a:b], this^next)
	}
	if remain > 0 {
		var buf [8]byte
		a := (batchsz - 1) * 8
		b := batchsz * 8
		copy(buf[:], data[b:])
		this = next
		next = binary.LittleEndian.Uint64(buf[:]) | (m.mask & (uint64(0xffffffff_ffffffff) << (uint64(remain) * 8)))
		if batchsz > 0 {
			binary.LittleEndian.PutUint64(data[a:b], this^next)
		}
		binary.LittleEndian.PutUint64(buf[:], next^m.mask)
		copy(data[b:], buf[:])
	} else {
		binary.LittleEndian.PutUint64(data[len(data)-8:], next^m.mask)
	}
	return binary.LittleEndian.Uint32(data[:4]), data[8:]
}

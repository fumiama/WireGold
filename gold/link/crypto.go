package link

import (
	"crypto/cipher"
	"crypto/rand"
	"encoding/binary"
	"errors"
	"math/bits"
	mrand "math/rand"
	"runtime"

	"github.com/fumiama/orbyte/pbuf"
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
func (l *Link) Encode(teatype uint8, additional uint16, b []byte) (eb pbuf.Bytes) {
	if len(b) == 0 || teatype >= 32 {
		return
	}
	if l.keys[0] == nil {
		return pbuf.ParseBytes(b...)
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
func (l *Link) Decode(teatype uint8, additional uint16, b []byte) (db pbuf.Bytes, err error) {
	if len(b) == 0 || teatype >= 32 {
		return
	}
	if l.keys[0] == nil {
		return pbuf.ParseBytes(b...), nil
	}
	aead := l.keys[teatype]
	if aead == nil {
		return
	}
	return decode(aead, additional, b)
}

func encode(aead cipher.AEAD, additional uint16, b []byte) pbuf.Bytes {
	nsz := aead.NonceSize()
	// Accocate capacity for all the stuffs.
	buf := pbuf.NewBytes(2 + nsz + len(b) + aead.Overhead())
	binary.LittleEndian.PutUint16(buf.Bytes()[:2], additional)
	nonce := buf.Bytes()[2 : 2+nsz]
	// Select a random nonce
	_, err := rand.Read(nonce)
	if err != nil {
		panic(err)
	}
	// Encrypt the message and append the ciphertext to the nonce.
	eb := aead.Seal(nonce[nsz:nsz], nonce, b, buf.Bytes()[:2])
	return buf.Trans().Slice(2, 2+nsz+len(eb))
}

func decode(aead cipher.AEAD, additional uint16, b []byte) (pbuf.Bytes, error) {
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
	data, err := aead.Open(
		pbuf.NewBytes(4096).Trans().Bytes()[:0],
		nonce, ciphertext, buf[:],
	)
	if err != nil {
		return pbuf.Bytes{}, nil
	}
	return pbuf.ParseBytes(data...), nil
}

// xorenc 按 8 字节, 以初始 m.mask 循环异或编码 data
func (m *Me) xorenc(data []byte, seq uint32) pbuf.Bytes {
	batchsz := len(data) / 8
	remain := len(data) % 8
	sum := m.mask
	newdat := pbuf.NewBytes(8 + batchsz*8 + 8) // seqrand dat tail
	binary.LittleEndian.PutUint32(newdat.Bytes()[:4], seq)
	_, _ = rand.Read(newdat.Bytes()[4:8])                 // seqrand
	sum ^= binary.LittleEndian.Uint64(newdat.Bytes()[:8]) // init from seqrand
	binary.LittleEndian.PutUint64(newdat.Bytes()[:8], sum)
	for i := 0; i < batchsz; i++ { // range on batch data
		a := i * 8
		b := (i + 1) * 8
		sum ^= binary.LittleEndian.Uint64(data[a:b])
		binary.LittleEndian.PutUint64(newdat.Bytes()[a+8:b+8], sum)
	}
	p := batchsz * 8
	copy(newdat.Bytes()[8+p:], data[p:])
	runtime.KeepAlive(data)
	newdat.Bytes()[newdat.Len()-1] = byte(remain)
	sum ^= binary.LittleEndian.Uint64(newdat.Bytes()[8+p:])
	binary.LittleEndian.PutUint64(newdat.Bytes()[8+p:], sum)
	return newdat
}

// xordec 按 8 字节, 以初始 m.mask 循环异或解码 data
func (m *Me) xordec(data []byte) (uint32, []byte) {
	if len(data) < 16 || len(data)%8 != 0 {
		return 0, nil
	}
	batchsz := len(data) / 8
	sum := m.mask
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

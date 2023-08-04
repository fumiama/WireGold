package link

import (
	"crypto/rand"
	"encoding/binary"
)

// Encode 使用 TEA 加密
func (l *Link) Encode(teatype uint8, b []byte) (eb []byte) {
	if b == nil || teatype >= 16 {
		return
	}
	if l.key == nil {
		eb = b
		return
	}
	// 在此处填写加密逻辑，密钥是l.key，输入是b，输出是eb
	// 不用写return，直接赋值给eb即可
	eb = l.key[teatype].Encrypt(b)
	return
}

// Decode 使用 TEA 解密
func (l *Link) Decode(teatype uint8, b []byte) (db []byte) {
	if b == nil || teatype >= 16 {
		return
	}
	if l.key == nil {
		db = b
		return
	}
	// 在此处填写解密逻辑，密钥是l.key，输入是b，输出是db
	// 不用写return，直接赋值给db即可
	db = l.key[teatype].Decrypt(b)
	return
}

// EncodePreshared 使用 xchacha20poly1305 加密
func (l *Link) EncodePreshared(additional uint16, b []byte) (eb []byte) {
	nsz := l.aead.NonceSize()
	// Select a random nonce, and leave capacity for the ciphertext.
	nonce := make([]byte, nsz, nsz+len(b)+l.aead.Overhead())
	_, err := rand.Read(nonce)
	if err != nil {
		return
	}
	// Encrypt the message and append the ciphertext to the nonce.
	var buf [2]byte
	binary.LittleEndian.PutUint16(buf[:], additional)
	eb = l.aead.Seal(nonce, nonce, b, buf[:])
	return
}

// DecodePreshared 使用 xchacha20poly1305 解密
func (l *Link) DecodePreshared(additional uint16, b []byte) (db []byte) {
	nsz := l.aead.NonceSize()
	if len(b) < nsz { // ciphertext too short
		return
	}
	// Split nonce and ciphertext.
	nonce, ciphertext := b[:nsz], b[nsz:]
	// Decrypt the message and check it wasn't tampered with.
	var buf [2]byte
	binary.LittleEndian.PutUint16(buf[:], additional)
	db, _ = l.aead.Open(nil, nonce, ciphertext, buf[:])
	return
}

// xor 按 8 字节, 以初始 m.mask 循环异或 data
func (m *Me) xor(data []byte) []byte {
	batchsz := len(data) / 8
	remain := len(data) % 8
	sum := m.mask
	for i := 0; i < batchsz; i++ {
		a := i * 8
		b := (i + 1) * 8
		sum ^= binary.LittleEndian.Uint64(data[a:b])
		binary.LittleEndian.PutUint64(data[a:b], sum)
	}
	if remain > 0 {
		var buf [8]byte
		copy(buf[:], data[remain:])
		sum ^= binary.LittleEndian.Uint64(buf[:])
		binary.LittleEndian.PutUint64(buf[:], sum)
		copy(data[remain:], buf[:])
	}
	return data
}

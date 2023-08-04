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

package link

import (
	"unsafe"

	tea "github.com/fumiama/gofastTEA"
)

// Encode 使用 TEA 加密
func (l *Link) Encode(b []byte) (eb []byte) {
	if b == nil {
		return
	}
	if l.key == nil {
		eb = b
	} else {
		// 在此处填写加密逻辑，密钥是l.key，输入是b，输出是eb
		// 不用写return，直接赋值给eb即可
		eb = (*tea.TEA)(unsafe.Pointer(l.key)).Encrypt(b)
	}
	return
}

// Decode 使用 TEA 解密
func (l *Link) Decode(b []byte) (db []byte) {
	if b == nil {
		return
	}
	if l.key == nil {
		db = b
	} else {
		// 在此处填写解密逻辑，密钥是l.key，输入是b，输出是db
		// 不用写return，直接赋值给db即可
		db = (*tea.TEA)(unsafe.Pointer(l.key)).Decrypt(b)
	}
	return
}

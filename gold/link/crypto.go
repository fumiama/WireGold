package link

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

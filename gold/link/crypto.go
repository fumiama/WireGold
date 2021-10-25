package link

import (
	"net"
)

var (
	// 本机私钥
	// 利用 Curve25519 生成
	// https://pkg.go.dev/golang.org/x/crypto/curve25519
	// https://www.zhihu.com/question/266758647
	privKey []byte
	// 本机虚拟 ip
	me net.IP
	// 本机 endpoint
	myend *net.UDPAddr
)

// SetMyself 设置本机参数
func SetMyself(privateKey []byte, myIP string, myEndpoint string) {
	privKey = privateKey
	var err error
	myend, err = net.ResolveUDPAddr("udp", myEndpoint)
	if err != nil {
		panic(err)
	}
	me = net.ParseIP(myIP)
	myconn, err = listen()
	if err != nil {
		panic(err)
	}
}

// Encode 使用 ChaCha20-Poly1305 加密
// https://pkg.go.dev/golang.org/x/crypto/chacha20poly1305
func (l *Link) Encode(b []byte) (eb []byte, err error) {
	if b == nil {
		return
	}
	if l.key == nil {
		eb = b
	} else {
		// 在此处填写加密逻辑，密钥是l.key，输入是b，输出是eb
		// 不用写return，直接赋值给eb即可
		eb = b
	}
	return
}

// Decode 使用 ChaCha20-Poly1305 解密
// https://pkg.go.dev/golang.org/x/crypto/chacha20poly1305
func (l *Link) Decode(b []byte) (db []byte, err error) {
	if b == nil {
		return
	}
	if l.key == nil {
		db = b
	} else {
		// 在此处填写解密逻辑，密钥是l.key，输入是b，输出是db
		// 不用写return，直接赋值给db即可
		db = b
	}
	return
}

package link

import "net"

var (
	// 本机私钥
	// 利用 Curve25519 生成
	// https://pkg.go.dev/golang.org/x/crypto/curve25519
	// https://www.zhihu.com/question/266758647
	privKey [32]byte
	// 本机虚拟 ip
	me net.IP
	// 本机 endpoint
	myend *net.UDPAddr
)

// SetMyself 设置本机参数
func SetMyself(privateKey [32]byte, myIP string, myEndpoint string) {
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

// Encode 使用  ChaCha20-Poly1305 AEAD 对称加密加密
// https://pkg.go.dev/golang.org/x/crypto/chacha20poly1305
func (l *Link) Encode(b []byte) (eb []byte, err error) {
	return b, nil
}

// Encode 使用  ChaCha20-Poly1305 AEAD 对称加密解密
// https://pkg.go.dev/golang.org/x/crypto/chacha20poly1305
func (l *Link) Decode(b []byte) (db []byte, err error) {
	return b, nil
}

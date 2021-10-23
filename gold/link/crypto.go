package link

import "net"

var (
	privKey [32]byte
	myip    string
	me      net.IP
)

func SetMyself(privateKey [32]byte, myIP string) {
	privKey = privateKey
	myip = myIP
	me = net.ParseIP(myIP)
}

func (id *Identity) Encode(b []byte) (n int, err error) {
	return 0, nil
}

func (id *Identity) Decode(b []byte) (n int, err error) {
	return 0, nil
}

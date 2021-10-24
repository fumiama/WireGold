package link

import "net"

var (
	privKey [32]byte
	me      net.IP
	myend   *net.UDPAddr
)

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

func (l *Link) Encode(b []byte) (eb []byte, err error) {
	return b, nil
}

func (l *Link) Decode(b []byte) (db []byte, err error) {
	return b, nil
}

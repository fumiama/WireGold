package link

import (
	"net"

	"github.com/fumiama/WireGold/gold/head"
	"github.com/sirupsen/logrus"
)

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

func listen() (conn *net.UDPConn, err error) {
	conn, err = net.ListenUDP("udp", myend)
	if err == nil {
		go func() {
			listenbuff := make([]byte, 65536)
			for {
				lbf := listenbuff
				n, addr, err := conn.ReadFromUDP(lbf)
				if err == nil {
					lbf = lbf[:n]
					p, ok := IsEndpointInPeer(addr.String())
					logrus.Infoln("[link] recv from endpoint", addr)
					logrus.Debugln("[link] recv:", string(lbf))
					if ok {
						packet := head.Packet{}
						d, err := p.Decode(lbf)
						if err == nil {
							packet.UnMashal(d)
							r := packet.DataSZ - uint32(len(packet.Data))
							if r > 0 {
								i := 0
								n := 0
								remain := make([]byte, r)
								for r > 0 {
									n, _, err = conn.ReadFromUDP(remain[i:])
									if err == nil {
										i += n
										r -= uint32(n)
									} else {
										logrus.Errorln("[link.listen]", err)
										return
									}
								}
								packet.Data = append(packet.Data, remain...)
							}
							logrus.Infoln("[link] deliver to", p.peerip)
							p.pipe <- &packet
						}
					}
				}
			}
		}()
	}
	return
}

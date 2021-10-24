package link

import (
	"net"

	"github.com/fumiama/WireGold/gold/head"
	"github.com/sirupsen/logrus"
)

var (
	privKey [32]byte
	me      net.IP
)

func SetMyself(privateKey [32]byte, myIP string) {
	privKey = privateKey
	me = net.ParseIP(myIP)
}

func (id *Identity) Encode(b []byte) (eb []byte, err error) {
	return b, nil
}

func (id *Identity) Decode(b []byte) (db []byte, err error) {
	return b, nil
}

func Listen(endpoint string) error {
	conn, err := net.ListenPacket("udp", endpoint)
	if err == nil {
		go func() {
			listenbuff := make([]byte, 65536)
			for {
				_, addr, err := conn.ReadFrom(listenbuff)
				if err == nil {
					p, ok := IsInPeer(addr.String())
					if ok {
						packet := head.Packet{}
						d, err := p.Decode(listenbuff)
						if err == nil {
							packet.UnMashal(d)
							r := packet.DataSZ - uint32(len(packet.Data))
							if r > 0 {
								i := 0
								n := 0
								remain := make([]byte, r)
								for r > 0 {
									n, _, err = conn.ReadFrom(remain[i:])
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
							p.pipe <- &packet
						}
					}
				}
			}
		}()
	}
	return err
}

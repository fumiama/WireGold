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
					packet := head.Packet{}
					err = packet.UnMashal(lbf)
					if err == nil {
						r := int(packet.DataSZ) - len(packet.Data)
						if r > 0 {
							remain, err := readAll(conn, r)
							if err == nil {
								packet.Data = append(packet.Data, remain...)
							}
						}
						p, ok := IsInPeer(packet.Src)
						logrus.Infoln("[link] recv from endpoint", addr, "src", packet.Src, "dst", packet.Dst)
						logrus.Debugln("[link] recv:", string(lbf))
						if ok {
							packet.Data, err = p.Decode(packet.Data)
							if err == nil {
								logrus.Infoln("[link] deliver to", p.peerip)
								if p.EndPoint == "" {
									logrus.Infoln("[link] set endpoint of peer", p.peerip, "to", addr.String())
									p.endpoint = addr
									p.EndPoint = addr.String()
								}
								p.pipe <- &packet
							}
						}
					}
				}
			}
		}()
	}
	return
}

func readAll(conn *net.UDPConn, sz int) ([]byte, error) {
	i := 0
	n := 0
	r := sz
	var err error
	remain := make([]byte, r)
	for sz > 0 {
		n, _, err = conn.ReadFromUDP(remain[i:])
		if err == nil {
			i += n
			r -= n
		} else {
			logrus.Errorln("[link] read all err:", err)
			return nil, err
		}
	}
	return remain, nil
}

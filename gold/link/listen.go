package link

import (
	"net"

	"github.com/fumiama/WireGold/gold/head"
	"github.com/sirupsen/logrus"
)

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
						if p.EndPoint == "" || p.EndPoint != addr.String() {
							logrus.Infoln("[link] set endpoint of peer", p.peerip, "to", addr.String())
							p.endpoint = addr
							p.EndPoint = addr.String()
						}
						if ok && p.Accept(net.IP(packet.Dst)) {
							packet.Data, err = p.Decode(packet.Data)
							if err == nil {
								switch packet.Proto {
								case head.ProtoHello:
									switch p.status {
									case LINK_STATUS_DOWN:
										_, _ = p.Write(head.NewPacket(head.ProtoHello, 0, 0, nil))
										logrus.Infoln("[link] send hello ack packet")
										p.status = LINK_STATUS_HALFUP
									case LINK_STATUS_HALFUP:
										p.status = LINK_STATUS_UP
									case LINK_STATUS_UP:
										break
									}
								case head.ProtoNotify:
									logrus.Infoln("[link] recv notify")
									onNotify(&packet)
								case head.ProtoQuery:
									logrus.Infoln("[link] recv query")
									onQuery(&packet)
								case head.ProtoData:
									logrus.Infoln("[link] deliver to", p.peerip)
									p.pipe <- &packet
								default:
									break
								}
							}
						} else {
							logrus.Infoln("[link] packet to", packet.Dst, "is refused")
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

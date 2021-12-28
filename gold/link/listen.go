package link

import (
	"net"

	"github.com/sirupsen/logrus"

	"github.com/fumiama/WireGold/gold/head"
)

// 监听本机 endpoint
func (m *Me) listen() (conn *net.UDPConn, err error) {
	conn, err = net.ListenUDP("udp", m.myend)
	if err == nil {
		go func() {
			listenbuff := make([]byte, 65536)
			for {
				lbf := listenbuff
				n, addr, err := conn.ReadFromUDP(lbf)
				if err == nil {
					lbf = lbf[:n]
					packet := head.Packet{}
					err = packet.Unmarshal(lbf)
					if err == nil {
						r := int(packet.DataSZ) - len(packet.Data)
						if r > 0 {
							remain, err := readAll(conn, r)
							if err == nil {
								packet.Data = append(packet.Data, remain...)
							}
						}
						p, ok := m.IsInPeer(packet.Src)
						logrus.Infoln("[link] recv from endpoint", addr, "src", packet.Src, "dst", packet.Dst)
						logrus.Debugln("[link] recv:", string(lbf))
						if ok {
							if p.pep == "" || p.pep != addr.String() {
								logrus.Infoln("[link] set endpoint of peer", p.peerip, "to", addr.String())
								p.endpoint = addr
								p.pep = addr.String()
							}
							if p.IsToMe(net.ParseIP(packet.Dst)) {
								packet.Data = p.Decode(packet.Data)
								if packet.IsVaildHash() {
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
										p.onNotify(&packet)
									case head.ProtoQuery:
										logrus.Infoln("[link] recv query")
										p.onQuery(&packet)
									case head.ProtoData:
										logrus.Infoln("[link] deliver to", p.peerip)
										if p.pipe != nil {
											p.pipe <- &packet
										} else {
											m.pipe <- &packet
										}
									default:
										break
									}
								} else {
									logrus.Infoln("[link] drop invalid packet")
								}
							} else if p.Accept(net.ParseIP(packet.Dst)) && p.allowtrans {
								// 转发
								p.Write(&packet)
								logrus.Infoln("[link] trans")
							}
						} else {
							logrus.Infoln("[link] packet to", packet.Dst, "is refused", "(me:", m.me, ")")
						}
					}
				}
			}
		}()
	}
	return
}

// Read 接收所有发送给本机的报文
// 需要开启 nopipe
func (m *Me) Read() *head.Packet {
	return <-m.pipe
}

// 从 conn 读取 sz 字节数据
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

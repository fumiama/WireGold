package link

import (
	"net"
	"strconv"

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
						p, ok := m.IsInPeer(packet.Src.String())
						logrus.Infoln("[link] recv from endpoint", addr, "src", packet.Src, "dst", packet.Dst)
						// logrus.Debugln("[link] recv:", hex.EncodeToString(lbf))
						if ok {
							if p.pep == "" || p.pep != addr.String() {
								logrus.Infoln("[link] set endpoint of peer", p.peerip, "to", addr.String())
								p.endpoint = addr
								p.pep = addr.String()
							}
							if p.IsToMe(packet.Dst) {
								packet.Data = p.Decode(packet.Data)
								if packet.IsVaildHash() {
									switch packet.Proto {
									case head.ProtoHello:
										switch p.status {
										case LINK_STATUS_DOWN:
											n, err = p.Write(head.NewPacket(head.ProtoHello, 0, p.peerip, 0, nil), false)
											if err == nil {
												logrus.Infoln("[link] send", n, "bytes hello ack packet")
												p.status = LINK_STATUS_HALFUP
											} else {
												logrus.Errorln("[link] send hello ack packet error:", err)
											}
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
										if p.pipe != nil {
											p.pipe <- &packet
											logrus.Infoln("[link] deliver to pipe of", p.peerip)
										} else {
											m.pipe <- &packet
											logrus.Infoln("[link] deliver to pipe of me")
										}
									default:
										logrus.Warnln("[link] recv unknown proto:", packet.Proto)
									}
								} else {
									logrus.Infoln("[link] drop invalid packet")
								}
							} else if p.Accept(packet.Dst) {
								if p.allowtrans {
									// 转发
									n, err = p.Write(&packet, true)
									if err == nil {
										logrus.Infoln("[link] trans", n, "bytes packet to", packet.Dst.String()+":"+strconv.Itoa(int(packet.DstPort)))
									} else {
										logrus.Errorln("[link] trans packet to", packet.Dst.String()+":"+strconv.Itoa(int(packet.DstPort)), "err:", err)
									}
								} else {
									logrus.Warnln("[link] refused to trans packet to", packet.Dst.String()+":"+strconv.Itoa(int(packet.DstPort)))
								}
							} else {
								logrus.Warnln("[link] packet dst", packet.Dst.String()+":"+strconv.Itoa(int(packet.DstPort)), "is not in peers")
							}
						} else {
							logrus.Warnln("[link] packet to", packet.Dst, "is refused")
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

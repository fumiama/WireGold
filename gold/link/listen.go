package link

import (
	"net"
	"runtime"
	"strconv"
	"sync"

	"github.com/sirupsen/logrus"

	"github.com/fumiama/WireGold/gold/head"
)

// 监听本机 endpoint
func (m *Me) listen() (conn *net.UDPConn, err error) {
	conn, err = net.ListenUDP("udp", m.myend)
	if err != nil {
		return
	}
	var mu sync.Mutex
	for i := 0; i < runtime.NumCPU()*4; i++ {
		go m.listenthread(conn, &mu)
	}
	return
}

func (m *Me) listenthread(conn *net.UDPConn, mu *sync.Mutex) {
	listenbuff := make([]byte, 65536)
	lbf := listenbuff
	for {
		lbf = listenbuff
		mu.Lock()
		n, addr, err := conn.ReadFromUDP(lbf)
		mu.Unlock()
		if err != nil {
			continue
		}
		lbf = lbf[:n]
		packet := m.wait(lbf)
		if packet == nil {
			continue
		}
		sz := packet.TeaTypeDataSZ & 0x0000ffff
		r := int(sz) - len(packet.Data)
		if r > 0 {
			logrus.Warnln("[link] packet from endpoint", addr, "is smaller than it declared: drop it")
			packet.Put()
			continue
		}
		p, ok := m.IsInPeer(packet.Src.String())
		logrus.Debugln("[link] recv from endpoint", addr, "src", packet.Src, "dst", packet.Dst)
		// logrus.Debugln("[link] recv:", hex.EncodeToString(lbf))
		if !ok {
			logrus.Warnln("[link] packet from", packet.Src, "to", packet.Dst, "is refused")
			packet.Put()
			continue
		}
		if p.endpoint == nil || p.endpoint.String() != addr.String() {
			logrus.Infoln("[link] set endpoint of peer", p.peerip, "to", addr.String())
			p.endpoint = addr
		}
		switch {
		case p.IsToMe(packet.Dst):
			packet.Data = p.Decode(uint8(packet.TeaTypeDataSZ>>28), packet.Data)
			if !packet.IsVaildHash() {
				logrus.Debugln("[link] drop invalid packet")
				packet.Put()
				continue
			}
			switch packet.Proto {
			case head.ProtoHello:
				switch p.status {
				case LINK_STATUS_DOWN:
					n, err = p.WriteAndPut(head.NewPacket(head.ProtoHello, m.SrcPort(), p.peerip, m.DstPort(), nil), false)
					if err == nil {
						logrus.Debugln("[link] send", n, "bytes hello ack packet")
						p.status = LINK_STATUS_HALFUP
					} else {
						logrus.Errorln("[link] send hello ack packet error:", err)
					}
				case LINK_STATUS_HALFUP:
					p.status = LINK_STATUS_UP
				case LINK_STATUS_UP:
				}
				packet.Put()
			case head.ProtoNotify:
				logrus.Infoln("[link] recv notify from", packet.Src)
				go p.onNotify(packet.Data)
				packet.Put()
			case head.ProtoQuery:
				logrus.Infoln("[link] recv query from", packet.Src)
				go p.onQuery(packet.Data)
				packet.Put()
			case head.ProtoData:
				if p.pipe != nil {
					p.pipe <- packet
					logrus.Debugln("[link] deliver to pipe of", p.peerip)
				} else {
					m.nic.Write(packet.Data)
					logrus.Debugln("[link] deliver", len(packet.Data), "bytes data to nic")
					packet.Put()
				}
			default:
				logrus.Warnln("[link] recv unknown proto:", packet.Proto)
				packet.Put()
			}
		case p.Accept(packet.Dst):
			if !p.allowtrans {
				logrus.Warnln("[link] refused to trans packet to", packet.Dst.String()+":"+strconv.Itoa(int(packet.DstPort)))
				packet.Put()
				continue
			}
			// 转发
			lnk := m.router.NextHop(packet.Dst.String())
			if lnk == nil {
				logrus.Warnln("[link] transfer drop packet: nil nexthop")
				packet.Put()
				continue
			}
			n, err = lnk.WriteAndPut(packet, true)
			if err == nil {
				logrus.Debugln("[link] trans", n, "bytes packet to", packet.Dst.String()+":"+strconv.Itoa(int(packet.DstPort)))
			} else {
				logrus.Errorln("[link] trans packet to", packet.Dst.String()+":"+strconv.Itoa(int(packet.DstPort)), "err:", err)
			}
		default:
			logrus.Warnln("[link] packet dst", packet.Dst.String()+":"+strconv.Itoa(int(packet.DstPort)), "is not in peers")
			packet.Put()
		}
	}
}

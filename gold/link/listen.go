package link

import (
	"bytes"
	"io"
	"net"
	"net/netip"
	"runtime"
	"strconv"
	"sync"
	"sync/atomic"
	"unsafe"

	"github.com/klauspost/compress/zstd"
	"github.com/sirupsen/logrus"

	"github.com/fumiama/WireGold/gold/head"
)

// 监听本机 endpoint
func (m *Me) listen() (conn *net.UDPConn, err error) {
	conn, err = net.ListenUDP("udp", net.UDPAddrFromAddrPort(netip.MustParseAddrPort(m.myend.String())))
	if err != nil {
		return
	}
	m.myend = conn.LocalAddr()
	logrus.Infoln("[listen] at", m.myend)
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
			logrus.Warnln("[listen] packet from endpoint", addr, "is smaller than it declared: drop it")
			packet.Put()
			continue
		}
		p, ok := m.IsInPeer(packet.Src.String())
		logrus.Debugln("[listen] recv from endpoint", addr, "src", packet.Src, "dst", packet.Dst)
		if !ok {
			logrus.Warnln("[listen] packet from", packet.Src, "to", packet.Dst, "is refused")
			packet.Put()
			continue
		}
		if p.endpoint == nil || p.endpoint.String() != addr.String() {
			logrus.Infoln("[listen] set endpoint of peer", p.peerip, "to", addr.String())
			atomic.StorePointer((*unsafe.Pointer)(unsafe.Pointer(&p.endpoint)), unsafe.Pointer(addr))
		}
		switch {
		case p.IsToMe(packet.Dst):
			packet.Data = p.Decode(uint8(packet.TeaTypeDataSZ>>28), packet.Data)
			if p.aead != nil {
				addt := packet.AdditionalData()
				packet.Data = p.DecodePreshared(addt, packet.Data)
				if packet.Data == nil {
					logrus.Debugln("[listen] drop invalid preshared packet, addt:", addt)
					packet.Put()
					continue
				}
			}
			if p.usezstd {
				dec, _ := zstd.NewReader(bytes.NewReader(packet.Data))
				packet.Data, err = io.ReadAll(dec)
				dec.Close()
				if err != nil {
					logrus.Debugln("[listen] drop invalid zstd packet:", err)
					packet.Put()
					continue
				}
			}
			if !packet.IsVaildHash() {
				logrus.Debugln("[listen] drop invalid hash packet")
				packet.Put()
				continue
			}
			switch packet.Proto {
			case head.ProtoHello:
				switch p.status {
				case LINK_STATUS_DOWN:
					n, err = p.WriteAndPut(head.NewPacket(head.ProtoHello, m.SrcPort(), p.peerip, m.DstPort(), nil), false)
					if err == nil {
						logrus.Debugln("[listen] send", n, "bytes hello ack packet")
						p.status = LINK_STATUS_HALFUP
					} else {
						logrus.Errorln("[listen] send hello ack packet error:", err)
					}
				case LINK_STATUS_HALFUP:
					p.status = LINK_STATUS_UP
				case LINK_STATUS_UP:
				}
				packet.Put()
			case head.ProtoNotify:
				logrus.Infoln("[listen] recv notify from", packet.Src)
				go p.onNotify(packet.Data)
				packet.Put()
			case head.ProtoQuery:
				logrus.Infoln("[listen] recv query from", packet.Src)
				go p.onQuery(packet.Data)
				packet.Put()
			case head.ProtoData:
				if p.pipe != nil {
					p.pipe <- packet
					logrus.Debugln("[listen] deliver to pipe of", p.peerip)
				} else {
					m.nic.Write(packet.Data)
					logrus.Debugln("[listen] deliver", len(packet.Data), "bytes data to nic")
					packet.Put()
				}
			default:
				logrus.Warnln("[listen] recv unknown proto:", packet.Proto)
				packet.Put()
			}
		case p.Accept(packet.Dst):
			if !p.allowtrans {
				logrus.Warnln("[listen] refused to trans packet to", packet.Dst.String()+":"+strconv.Itoa(int(packet.DstPort)))
				packet.Put()
				continue
			}
			// 转发
			lnk := m.router.NextHop(packet.Dst.String())
			if lnk == nil {
				logrus.Warnln("[listen] transfer drop packet: nil nexthop")
				packet.Put()
				continue
			}
			n, err = lnk.WriteAndPut(packet, true)
			if err == nil {
				logrus.Debugln("[listen] trans", n, "bytes packet to", packet.Dst.String()+":"+strconv.Itoa(int(packet.DstPort)))
			} else {
				logrus.Errorln("[listen] trans packet to", packet.Dst.String()+":"+strconv.Itoa(int(packet.DstPort)), "err:", err)
			}
		default:
			logrus.Warnln("[listen] packet dst", packet.Dst.String()+":"+strconv.Itoa(int(packet.DstPort)), "is not in peers")
			packet.Put()
		}
	}
}

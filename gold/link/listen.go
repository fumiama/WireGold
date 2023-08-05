package link

import (
	"bytes"
	"io"
	"net"
	"net/netip"
	"runtime"
	"strconv"
	"sync/atomic"
	"time"
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
	go func() {
		recvtotlcnt := 0
		recvloopcnt := 0
		recvlooptime := time.Now().UnixMilli()
		n := runtime.NumCPU()
		if n > 64 {
			n = 64 // 只用最多 64 核
		}
		logrus.Infoln("[listen] use cpu num:", n)
		listenbuff := make([]byte, 65536*n)
		hasntfinished := make([]bool, n)
		for i := 0; err == nil; i++ {
			i %= n
			for hasntfinished[i] {
				time.Sleep(time.Millisecond)
				i++
				i %= n
			}
			lbf := listenbuff[i*65536 : (i+1)*65536]
			n, addr, err := conn.ReadFromUDP(lbf)
			if err != nil {
				logrus.Warnln("[listen] read from udp err, reconnect:", err)
				conn, err = net.ListenUDP("udp", net.UDPAddrFromAddrPort(netip.MustParseAddrPort(m.myend.String())))
				if err != nil {
					logrus.Errorln("[listen] reconnect udp err:", err)
					return
				}
				i--
				continue
			}
			recvtotlcnt += len(lbf)
			recvloopcnt++
			if recvloopcnt >= 4096 {
				now := time.Now().UnixMilli()
				logrus.Infof("[listen] recv avg speed: %.2f KB/s", float64(recvtotlcnt)/float64(now-recvlooptime))
				recvtotlcnt = 0
				recvloopcnt = 0
				recvlooptime = now
			}
			packet := m.wait(lbf[:n])
			if packet == nil {
				i--
				continue
			}
			hasntfinished[i] = true
			go m.listenthread(packet, addr, i, func() { hasntfinished[i] = false })
		}
	}()
	return
}

func (m *Me) listenthread(packet *head.Packet, addr *net.UDPAddr, index int, finish func()) {
	defer finish()
	sz := packet.TeaTypeDataSZ & 0x0000ffff
	r := int(sz) - len(packet.Data)
	if r > 0 {
		logrus.Warnln("[listen] @", index, "packet from endpoint", addr, "is smaller than it declared: drop it")
		packet.Put()
		return
	}
	p, ok := m.IsInPeer(packet.Src.String())
	logrus.Debugln("[listen] @", index, "recv from endpoint", addr, "src", packet.Src, "dst", packet.Dst)
	if !ok {
		logrus.Warnln("[listen] @", index, "packet from", packet.Src, "to", packet.Dst, "is refused")
		packet.Put()
		return
	}
	if p.endpoint == nil || p.endpoint.String() != addr.String() {
		logrus.Infoln("[listen] @", index, "set endpoint of peer", p.peerip, "to", addr.String())
		atomic.StorePointer((*unsafe.Pointer)(unsafe.Pointer(&p.endpoint)), unsafe.Pointer(addr))
	}
	switch {
	case p.IsToMe(packet.Dst):
		packet.Data = p.Decode(uint8(packet.TeaTypeDataSZ>>28), packet.Data)
		if p.aead != nil {
			addt := packet.AdditionalData()
			packet.Data = p.DecodePreshared(addt, packet.Data)
			if packet.Data == nil {
				logrus.Debugln("[listen] @", index, "drop invalid preshared packet, addt:", addt)
				packet.Put()
				return
			}
		}
		if p.usezstd {
			dec, _ := zstd.NewReader(bytes.NewReader(packet.Data))
			var err error
			packet.Data, err = io.ReadAll(dec)
			dec.Close()
			if err != nil {
				logrus.Debugln("[listen] @", index, "drop invalid zstd packet:", err)
				packet.Put()
				return
			}
		}
		if !packet.IsVaildHash() {
			logrus.Debugln("[listen] @", index, "drop invalid hash packet")
			packet.Put()
			return
		}
		switch packet.Proto {
		case head.ProtoHello:
			switch p.status {
			case LINK_STATUS_DOWN:
				n, err := p.WriteAndPut(head.NewPacket(head.ProtoHello, m.SrcPort(), p.peerip, m.DstPort(), nil), false)
				if err == nil {
					logrus.Debugln("[listen] @", index, "send", n, "bytes hello ack packet")
					p.status = LINK_STATUS_HALFUP
				} else {
					logrus.Errorln("[listen] @", index, "send hello ack packet error:", err)
				}
			case LINK_STATUS_HALFUP:
				p.status = LINK_STATUS_UP
			case LINK_STATUS_UP:
			}
			packet.Put()
		case head.ProtoNotify:
			logrus.Infoln("[listen] @", index, "recv notify from", packet.Src)
			go p.onNotify(packet.Data)
			packet.Put()
		case head.ProtoQuery:
			logrus.Infoln("[listen] @", index, "recv query from", packet.Src)
			go p.onQuery(packet.Data)
			packet.Put()
		case head.ProtoData:
			if p.pipe != nil {
				p.pipe <- packet
				logrus.Debugln("[listen] @", index, "deliver to pipe of", p.peerip)
			} else {
				m.nic.Write(packet.Data)
				logrus.Debugln("[listen] @", index, "deliver", len(packet.Data), "bytes data to nic")
				packet.Put()
			}
		default:
			logrus.Warnln("[listen] @", index, "recv unknown proto:", packet.Proto)
			packet.Put()
		}
	case p.Accept(packet.Dst):
		if !p.allowtrans {
			logrus.Warnln("[listen] @", index, "refused to trans packet to", packet.Dst.String()+":"+strconv.Itoa(int(packet.DstPort)))
			packet.Put()
			return
		}
		// 转发
		lnk := m.router.NextHop(packet.Dst.String())
		if lnk == nil {
			logrus.Warnln("[listen] @", index, "transfer drop packet: nil nexthop")
			packet.Put()
			return
		}
		n, err := lnk.WriteAndPut(packet, true)
		if err == nil {
			logrus.Debugln("[listen] @", index, "trans", n, "bytes packet to", packet.Dst.String()+":"+strconv.Itoa(int(packet.DstPort)))
		} else {
			logrus.Errorln("[listen] @", index, "trans packet to", packet.Dst.String()+":"+strconv.Itoa(int(packet.DstPort)), "err:", err)
		}
	default:
		logrus.Warnln("[listen] @", index, "packet dst", packet.Dst.String()+":"+strconv.Itoa(int(packet.DstPort)), "is not in peers")
		packet.Put()
	}
}

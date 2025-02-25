package link

import (
	"errors"
	"net"
	"runtime"
	"strconv"
	"sync/atomic"
	"time"
	"unsafe"

	"github.com/sirupsen/logrus"

	"github.com/fumiama/WireGold/config"
	"github.com/fumiama/WireGold/gold/head"
	"github.com/fumiama/WireGold/gold/p2p"
	"github.com/fumiama/WireGold/helper"
	"github.com/fumiama/orbyte"
	"github.com/fumiama/orbyte/pbuf"
)

const lstnbufgragsz = 65536

// 监听本机 endpoint
func (m *Me) listen() (conn p2p.Conn, err error) {
	conn, err = m.ep.Listen()
	if err != nil {
		return
	}
	m.ep = conn.LocalAddr()
	logrus.Infoln("[listen] at", m.ep)
	go func() {
		for {
			lbf := pbuf.NewBytes(lstnbufgragsz)
			n, addr, err := conn.ReadFromPeer(lbf.Bytes())
			lbf.KeepAlive()
			if m.connections == nil || errors.Is(err, net.ErrClosed) {
				logrus.Warnln("[listen] quit listening")
				return
			}
			if err != nil {
				logrus.Warnln("[listen] read from conn err, reconnect:", err)
				conn, err = m.ep.Listen()
				if err != nil {
					logrus.Errorln("[listen] reconnect udp err:", err)
					return
				}
				continue
			}
			if n <= 0 {
				if config.ShowDebugLog {
					logrus.Debugln("[listen] unexpected read n =", n)
				}
				continue
			}
			go m.waitordispatch(addr, lbf, n)
		}
	}()
	return
}

func (m *Me) waitordispatch(addr p2p.EndPoint, buf pbuf.Bytes, n int) {
	recvtotlcnt := atomic.AddUint64(&m.recvtotlcnt, uint64(buf.Len()))
	recvloopcnt := atomic.AddUintptr(&m.recvloopcnt, 1)
	recvlooptime := atomic.LoadInt64(&m.recvlooptime)
	if recvloopcnt%uintptr(m.speedloop) == 0 {
		now := time.Now().UnixMilli()
		logrus.Infof("[listen] queue recv avg speed: %.2f KB/s", float64(recvtotlcnt)/float64(now-recvlooptime))
		atomic.StoreUint64(&m.recvtotlcnt, 0)
		atomic.StoreInt64(&m.recvlooptime, now)
	}
	packet := m.wait(buf.SliceTo(n).Bytes())
	buf.KeepAlive()
	if packet == nil {
		if config.ShowDebugLog {
			logrus.Debugln("[listen] queue waiting")
		}
		return
	}
	if config.ShowDebugLog {
		logrus.Debugln("[listen] dispatch", len(packet.Pointer().UnsafeBody()), "bytes packet")
	}
	m.dispatch(packet, addr)
}

func (m *Me) dispatch(packet *orbyte.Item[head.Packet], addr p2p.EndPoint) {
	pp := packet.Pointer
	r := pp().Len() - pp().BodyLen()
	if r > 0 {
		logrus.Warnln("[listen] packet from endpoint", addr, "len", pp().BodyLen(), "is smaller than it declared len", pp().Len(), ", drop it")
		return
	}
	p, ok := m.IsInPeer(pp().Src.String())
	if config.ShowDebugLog {
		logrus.Debugln("[listen] recv from endpoint", addr, "src", pp().Src, "dst", pp().Dst)
	}
	if !ok {
		logrus.Warnln("[listen] packet from", pp().Src, "to", pp().Dst, "is refused")
		return
	}
	if helper.IsNilInterface(p.endpoint) || !p.endpoint.Euqal(addr) {
		if m.ep.Network() == "tcp" && !addr.Euqal(p.endpoint) {
			logrus.Infoln("[listen] set endpoint of peer", p.peerip, "to", addr.String())
			p.endpoint = addr
		} else { // others are all no status link
			logrus.Infoln("[listen] set endpoint of peer", p.peerip, "to", addr.String())
			p.endpoint = addr
		}
	}
	now := time.Now()
	atomic.StorePointer((*unsafe.Pointer)(unsafe.Pointer(&p.lastalive)), unsafe.Pointer(&now))
	switch {
	case p.IsToMe(pp().Dst):
		if !p.Accept(pp().Src) {
			logrus.Warnln("[listen] refused packet from", pp().Src.String()+":"+strconv.Itoa(int(pp().SrcPort)))
			return
		}
		addt := pp().AdditionalData()
		var err error
		data, err := p.decode(pp().CipherIndex(), addt, pp().TransBody().Bytes())
		if err != nil {
			if config.ShowDebugLog {
				logrus.Debugln("[listen] drop invalid packet key idx:", pp().CipherIndex(), "addt:", addt, "err:", err)
			}
			return
		}
		if p.usezstd {
			dat, err := decodezstd(data.Trans().Bytes())
			if err != nil {
				if config.ShowDebugLog {
					logrus.Debugln("[listen] drop invalid zstd packet:", err)
				}
				return
			}
			if config.ShowDebugLog {
				logrus.Debugln("[listen] zstd decoded len:", dat.Len())
			}
			data = dat
		}
		pp().SetBody(data)
		if !pp().IsVaildHash() {
			if config.ShowDebugLog {
				logrus.Debugln("[listen] drop invalid hash packet")
			}
			return
		}
		switch pp().Proto {
		case head.ProtoHello:
			switch {
			case len(pp().UnsafeBody()) == 0:
				logrus.Warnln("[listen] recv old hello packet, do nothing")
			case pp().UnsafeBody()[0] == byte(head.HelloPing):
				n, err := p.WritePacket(head.NewPacketPartial(
					head.ProtoHello, m.SrcPort(), p.peerip, m.DstPort(), pbuf.ParseBytes(byte(head.HelloPong))), false)
				if err == nil {
					logrus.Infoln("[listen] recv hello, send", n, "bytes hello ack packet")
				} else {
					logrus.Errorln("[listen] send hello ack packet error:", err)
				}
			default:
				logrus.Infoln("[listen] recv hello ack packet, do nothing")
			}
		case head.ProtoNotify:
			logrus.Infoln("[listen] recv notify from", pp().Src)
			p.onNotify(pp().UnsafeBody())
			runtime.KeepAlive(packet)
		case head.ProtoQuery:
			logrus.Infoln("[listen] recv query from", pp().Src)
			p.onQuery(pp().UnsafeBody())
			runtime.KeepAlive(packet)
		case head.ProtoData:
			if p.pipe != nil {
				p.pipe <- packet.Copy()
				if config.ShowDebugLog {
					logrus.Debugln("[listen] deliver to pipe of", p.peerip)
				}
			} else {
				_, err := m.nic.Write(pp().UnsafeBody())
				if err != nil {
					logrus.Errorln("[listen] deliver", pp().BodyLen(), "bytes data to nic err:", err)
				} else if config.ShowDebugLog {
					logrus.Debugln("[listen] deliver", pp().BodyLen(), "bytes data to nic")
				}
			}
		default:
			logrus.Warnln("[listen] recv unknown proto:", pp().Proto)
		}
	case p.Accept(pp().Dst):
		if !p.allowtrans {
			logrus.Warnln("[listen] refused to trans packet to", pp().Dst.String()+":"+strconv.Itoa(int(pp().DstPort)))
			return
		}
		// 转发
		lnk := m.router.NextHop(pp().Dst.String())
		if lnk == nil {
			logrus.Warnln("[listen] transfer drop packet: nil nexthop")
			return
		}
		n, err := lnk.WritePacket(packet, true)
		if err == nil {
			if config.ShowDebugLog {
				logrus.Debugln("[listen] trans", n, "bytes packet to", pp().Dst.String()+":"+strconv.Itoa(int(pp().DstPort)))
			}
		} else {
			logrus.Errorln("[listen] trans packet to", pp().Dst.String()+":"+strconv.Itoa(int(pp().DstPort)), "err:", err)
		}
	default:
		logrus.Warnln("[listen] packet dst", pp().Dst.String()+":"+strconv.Itoa(int(pp().DstPort)), "is not in peers")
	}
}

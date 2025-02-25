package link

import (
	"bytes"
	"errors"
	"io"
	"net"
	"runtime"
	"strconv"
	"sync"
	"sync/atomic"
	"time"
	"unsafe"

	"github.com/klauspost/compress/zstd"
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
		n := uint(runtime.NumCPU())
		if n > 64 {
			n = 64 // 只用最多 64 核
		}
		logrus.Infoln("[listen] use cpu num:", n)
		listenbuf := make([]byte, lstnbufgragsz*n)
		hasntfinished := make([]sync.Mutex, n)
		for {
			usenewbuf := false
			i := uint(0)
			for !hasntfinished[i].TryLock() {
				i++
				i %= n
				if i == 0 { // looked up a full round, make a new buf
					usenewbuf = true
					if config.ShowDebugLog {
						logrus.Debugln("[listen] use new buf")
					}
					break
				}
			}
			if config.ShowDebugLog && !usenewbuf {
				logrus.Debugln("[listen] lock index", i)
			}
			var lbf pbuf.Bytes
			if usenewbuf {
				lbf = pbuf.NewBytes(lstnbufgragsz)
			} else {
				if config.ShowDebugLog {
					logrus.Debugln("[listen] take index", i, "slice", i*lstnbufgragsz, (i+1)*lstnbufgragsz, "cap", lstnbufgragsz)
				}
				lbf = pbuf.ParseBytes(listenbuf[i*lstnbufgragsz : (i+1)*lstnbufgragsz : (i+1)*lstnbufgragsz]...)
			}
			n, addr, err := conn.ReadFromPeer(lbf.Bytes())
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
				if !usenewbuf {
					if config.ShowDebugLog {
						logrus.Debugln("[listen] unlock index", i)
					}
					hasntfinished[i].Unlock()
					i--
				}
				continue
			}
			if n <= 0 {
				if config.ShowDebugLog {
					logrus.Debugln("[listen] unexpected read n =", n)
				}
				continue
			}
			index := -1
			if !usenewbuf {
				index = int(i)
			}
			go m.waitordispatch(index, addr, lbf.Trans().SliceTo(n), hasntfinished)
		}
	}()
	return
}

func (m *Me) waitordispatch(index int, addr p2p.EndPoint, buf pbuf.Bytes, hasntfinished []sync.Mutex) {
	recvtotlcnt := atomic.AddUint64(&m.recvtotlcnt, uint64(buf.Len()))
	recvloopcnt := atomic.AddUintptr(&m.recvloopcnt, 1)
	recvlooptime := atomic.LoadInt64(&m.recvlooptime)
	if recvloopcnt%uintptr(m.speedloop) == 0 {
		now := time.Now().UnixMilli()
		logrus.Infof("[listen] queue recv avg speed: %.2f KB/s", float64(recvtotlcnt)/float64(now-recvlooptime))
		atomic.StoreUint64(&m.recvtotlcnt, 0)
		atomic.StoreInt64(&m.recvlooptime, now)
	}
	packet := m.wait(buf.Trans().Bytes())
	if packet == nil {
		if index < 0 {
			if config.ShowDebugLog {
				logrus.Debugln("[listen] queue waiting")
			}
			return
		}
		if config.ShowDebugLog {
			logrus.Debugln("[listen] queue waiting, unlock index", index)
		}
		hasntfinished[index].Unlock()
		return
	}
	if config.ShowDebugLog {
		logrus.Debugln("[listen] index", index, "dispatch", len(packet.Pointer().Body()), "bytes packet")
	}
	if index >= 0 {
		defer hasntfinished[index].Unlock()
		m.dispatch(packet, addr, index)
		return
	}
	m.dispatch(packet, addr, index)
}

func (m *Me) dispatch(packet *orbyte.Item[head.Packet], addr p2p.EndPoint, index int) {
	defer runtime.KeepAlive(packet)

	if config.ShowDebugLog {
		defer logrus.Debugln("[listen] dispatched, unlock index", index)
		logrus.Debugln("[listen] start dispatching index", index)
	}
	pp := packet.Pointer()
	r := pp.Len() - pp.BodyLen()
	if r > 0 {
		logrus.Warnln("[listen] @", index, "packet from endpoint", addr, "len", pp.BodyLen(), "is smaller than it declared len", pp.Len(), ", drop it")
		return
	}
	p, ok := m.IsInPeer(pp.Src.String())
	if config.ShowDebugLog {
		logrus.Debugln("[listen] @", index, "recv from endpoint", addr, "src", pp.Src, "dst", pp.Dst)
	}
	if !ok {
		logrus.Warnln("[listen] @", index, "packet from", pp.Src, "to", pp.Dst, "is refused")
		return
	}
	if helper.IsNilInterface(p.endpoint) || !p.endpoint.Euqal(addr) {
		if m.ep.Network() == "tcp" && !addr.Euqal(p.endpoint) {
			logrus.Infoln("[listen] @", index, "set endpoint of peer", p.peerip, "to", addr.String())
			p.endpoint = addr
		} else { // others are all no status link
			logrus.Infoln("[listen] @", index, "set endpoint of peer", p.peerip, "to", addr.String())
			p.endpoint = addr
		}
	}
	now := time.Now()
	atomic.StorePointer((*unsafe.Pointer)(unsafe.Pointer(&p.lastalive)), unsafe.Pointer(&now))
	switch {
	case p.IsToMe(pp.Dst):
		if !p.Accept(pp.Src) {
			logrus.Warnln("[listen] @", index, "refused packet from", pp.Src.String()+":"+strconv.Itoa(int(pp.SrcPort)))
			return
		}
		addt := pp.AdditionalData()
		var err error
		data, err := p.Decode(pp.CipherIndex(), addt, pp.Body())
		if err != nil {
			if config.ShowDebugLog {
				logrus.Debugln("[listen] @", index, "drop invalid packet key idx:", pp.CipherIndex(), "addt:", addt, "err:", err)
			}
			return
		}
		pp.SetBody(data.Trans().Bytes())
		if p.usezstd {
			dec, _ := zstd.NewReader(bytes.NewReader(pp.Body()))
			var err error
			w := helper.SelectWriter()
			_, err = io.Copy(w, dec)
			dec.Close()
			if err != nil {
				if config.ShowDebugLog {
					logrus.Debugln("[listen] @", index, "drop invalid zstd packet:", err)
				}
				return
			}
			if config.ShowDebugLog {
				logrus.Debugln("[listen] @", index, "zstd decoded len:", w.Len())
			}
			pp.SetBody(w.TransBytes().Bytes())
		}
		if !pp.IsVaildHash() {
			if config.ShowDebugLog {
				logrus.Debugln("[listen] @", index, "drop invalid hash packet")
			}
			return
		}
		switch pp.Proto {
		case head.ProtoHello:
			switch {
			case len(pp.Body()) == 0:
				logrus.Warnln("[listen] @", index, "recv old hello packet, do nothing")
			case pp.Body()[0] == byte(head.HelloPing):
				n, err := p.WritePacket(head.NewPacketPartial(
					head.ProtoHello, m.SrcPort(), p.peerip, m.DstPort(), pbuf.ParseBytes(byte(head.HelloPong))), false)
				if err == nil {
					logrus.Infoln("[listen] @", index, "recv hello, send", n, "bytes hello ack packet")
				} else {
					logrus.Errorln("[listen] @", index, "send hello ack packet error:", err)
				}
			default:
				logrus.Infoln("[listen] @", index, "recv hello ack packet, do nothing")
			}
		case head.ProtoNotify:
			logrus.Infoln("[listen] @", index, "recv notify from", pp.Src)
			p.onNotify(pp.Body())
		case head.ProtoQuery:
			logrus.Infoln("[listen] @", index, "recv query from", pp.Src)
			p.onQuery(pp.Body())
		case head.ProtoData:
			if p.pipe != nil {
				p.pipe <- packet.Copy()
				if config.ShowDebugLog {
					logrus.Debugln("[listen] @", index, "deliver to pipe of", p.peerip)
				}
			} else {
				_, err := m.nic.Write(pp.Body())
				if err != nil {
					logrus.Errorln("[listen] @", index, "deliver", pp.BodyLen(), "bytes data to nic err:", err)
				} else if config.ShowDebugLog {
					logrus.Debugln("[listen] @", index, "deliver", pp.BodyLen(), "bytes data to nic")
				}
			}
		default:
			logrus.Warnln("[listen] @", index, "recv unknown proto:", pp.Proto)
		}
	case p.Accept(pp.Dst):
		if !p.allowtrans {
			logrus.Warnln("[listen] @", index, "refused to trans packet to", pp.Dst.String()+":"+strconv.Itoa(int(pp.DstPort)))
			return
		}
		// 转发
		lnk := m.router.NextHop(pp.Dst.String())
		if lnk == nil {
			logrus.Warnln("[listen] @", index, "transfer drop packet: nil nexthop")
			return
		}
		n, err := lnk.WritePacket(packet, true)
		if err == nil {
			if config.ShowDebugLog {
				logrus.Debugln("[listen] @", index, "trans", n, "bytes packet to", pp.Dst.String()+":"+strconv.Itoa(int(pp.DstPort)))
			}
		} else {
			logrus.Errorln("[listen] @", index, "trans packet to", pp.Dst.String()+":"+strconv.Itoa(int(pp.DstPort)), "err:", err)
		}
	default:
		logrus.Warnln("[listen] @", index, "packet dst", pp.Dst.String()+":"+strconv.Itoa(int(pp.DstPort)), "is not in peers")
	}
}

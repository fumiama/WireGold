package link

import (
	"bytes"
	"errors"
	"io"
	"net"
	"runtime"
	"strconv"
	"sync"
	"time"

	"github.com/klauspost/compress/zstd"
	"github.com/sirupsen/logrus"

	"github.com/fumiama/WireGold/config"
	"github.com/fumiama/WireGold/gold/head"
	"github.com/fumiama/WireGold/gold/p2p"
	"github.com/fumiama/WireGold/helper"
)

const lstnbufgragsz = 65536

type lstnq struct {
	index int
	addr  p2p.EndPoint
	buf   []byte
}

type listenqueue chan lstnq

func (q listenqueue) listen(m *Me, hasntfinished []sync.Mutex) {
	recvtotlcnt := uint64(0)
	recvloopcnt := uint16(0)
	recvlooptime := time.Now().UnixMilli()
	for lstn := range q {
		recvtotlcnt += uint64(len(lstn.buf))
		recvloopcnt++
		if recvloopcnt%m.speedloop == 0 {
			now := time.Now().UnixMilli()
			logrus.Infof("[listen] queue recv avg speed: %.2f KB/s", float64(recvtotlcnt)/float64(now-recvlooptime))
			recvtotlcnt = 0
			recvlooptime = now
		}
		packet := m.wait(lstn.buf[:len(lstn.buf):lstnbufgragsz])
		if packet == nil {
			if lstn.index < 0 {
				if config.ShowDebugLog {
					logrus.Debugln("[listen] queue waiting")
				}
				helper.PutBytes(lstn.buf)
				continue
			}
			if config.ShowDebugLog {
				logrus.Debugln("[listen] queue waiting, unlock index", lstn.index)
			}
			hasntfinished[lstn.index].Unlock()
			continue
		}
		if lstn.index >= 0 {
			go m.dispatch(packet, lstn.addr, lstn.index, hasntfinished[lstn.index].Unlock)
		} else {
			go m.dispatch(packet, lstn.addr, lstn.index, func() {
				helper.PutBytes(lstn.buf)
			})
		}
	}
}

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
		q := make(listenqueue, n)
		defer close(q)
		go q.listen(m, hasntfinished)
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
			var lbf []byte
			if usenewbuf {
				lbf = helper.MakeBytes(lstnbufgragsz)
			} else {
				lbf = listenbuf[i*lstnbufgragsz : (i+1)*lstnbufgragsz]
			}
			n, addr, err := conn.ReadFromPeer(lbf)
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
			lq := lstnq{
				index: -1,
				addr:  addr,
				buf:   lbf[:n],
			}
			if !usenewbuf {
				lq.index = int(i)
			}
			q <- lq
		}
	}()
	return
}

func (m *Me) dispatch(packet *head.Packet, addr p2p.EndPoint, index int, finish func()) {
	defer finish()
	if config.ShowDebugLog {
		defer logrus.Debugln("[listen] dispatched, unlock index", index)
		logrus.Debugln("[listen] start dispatching index", index)
	}
	r := packet.Len() - packet.BodyLen()
	if r > 0 {
		logrus.Warnln("[listen] @", index, "packet from endpoint", addr, "len", packet.BodyLen(), "is smaller than it declared len", packet.Len(), ", drop it")
		packet.Put()
		return
	}
	p, ok := m.IsInPeer(packet.Src.String())
	if config.ShowDebugLog {
		logrus.Debugln("[listen] @", index, "recv from endpoint", addr, "src", packet.Src, "dst", packet.Dst)
	}
	if !ok {
		logrus.Warnln("[listen] @", index, "packet from", packet.Src, "to", packet.Dst, "is refused")
		packet.Put()
		return
	}
	if p.endpoint == nil || !p.endpoint.Euqal(addr) {
		if m.ep.Network() == "tcp" && !addr.Euqal(p.endpoint) {
			logrus.Infoln("[listen] @", index, "set endpoint of peer", p.peerip, "to", addr.String())
			p.endpoint = addr
		} else { // others are all no status link
			logrus.Infoln("[listen] @", index, "set endpoint of peer", p.peerip, "to", addr.String())
			p.endpoint = addr
		}
	}
	switch {
	case p.IsToMe(packet.Dst):
		if !p.Accept(packet.Src) {
			logrus.Warnln("[listen] @", index, "refused packet from", packet.Src.String()+":"+strconv.Itoa(int(packet.SrcPort)))
			packet.Put()
			return
		}
		addt := packet.AdditionalData()
		var err error
		data, err := p.Decode(packet.CipherIndex(), addt, packet.Body())
		if err != nil {
			if config.ShowDebugLog {
				logrus.Debugln("[listen] @", index, "drop invalid packet key idx:", packet.CipherIndex(), "addt:", addt, "err:", err)
			}
			packet.Put()
			return
		}
		packet.SetBody(data, true)
		if p.usezstd {
			dec, _ := zstd.NewReader(bytes.NewReader(packet.Body()))
			var err error
			w := helper.SelectWriter()
			_, err = io.Copy(w, dec)
			dec.Close()
			if err != nil {
				if config.ShowDebugLog {
					logrus.Debugln("[listen] @", index, "drop invalid zstd packet:", err)
				}
				packet.Put()
				return
			}
			packet.SetBody(w.Bytes(), true)
		}
		if !packet.IsVaildHash() {
			if config.ShowDebugLog {
				logrus.Debugln("[listen] @", index, "drop invalid hash packet")
			}
			packet.Put()
			return
		}
		switch packet.Proto {
		case head.ProtoHello:
			switch p.status {
			case LINK_STATUS_DOWN:
				n, err := p.WriteAndPut(head.NewPacket(head.ProtoHello, m.SrcPort(), p.peerip, m.DstPort(), nil), false)
				if err == nil {
					if config.ShowDebugLog {
						logrus.Debugln("[listen] @", index, "send", n, "bytes hello ack packet")
					}
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
			go p.onNotify(packet.Body())
			packet.Put()
		case head.ProtoQuery:
			logrus.Infoln("[listen] @", index, "recv query from", packet.Src)
			go p.onQuery(packet.Body())
			packet.Put()
		case head.ProtoData:
			if p.pipe != nil {
				p.pipe <- packet.CopyWithBody()
				if config.ShowDebugLog {
					logrus.Debugln("[listen] @", index, "deliver to pipe of", p.peerip)
				}
			} else {
				_, err := m.nic.Write(packet.Body())
				if err != nil {
					logrus.Errorln("[listen] @", index, "deliver", packet.BodyLen(), "bytes data to nic err:", err)
				} else if config.ShowDebugLog {
					logrus.Debugln("[listen] @", index, "deliver", packet.BodyLen(), "bytes data to nic")
				}
			}
			packet.Put()
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
			if config.ShowDebugLog {
				logrus.Debugln("[listen] @", index, "trans", n, "bytes packet to", packet.Dst.String()+":"+strconv.Itoa(int(packet.DstPort)))
			}
		} else {
			logrus.Errorln("[listen] @", index, "trans packet to", packet.Dst.String()+":"+strconv.Itoa(int(packet.DstPort)), "err:", err)
		}
	default:
		logrus.Warnln("[listen] @", index, "packet dst", packet.Dst.String()+":"+strconv.Itoa(int(packet.DstPort)), "is not in peers")
		packet.Put()
	}
}

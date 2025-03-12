package link

import (
	"errors"
	"net"
	"strconv"
	"sync/atomic"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/fumiama/WireGold/config"
	"github.com/fumiama/WireGold/gold/head"
	"github.com/fumiama/WireGold/gold/p2p"
	"github.com/fumiama/WireGold/internal/algo"
	"github.com/fumiama/WireGold/internal/file"
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
		var (
			n    int
			addr p2p.EndPoint
			err  error
		)
		for {
			lbf := pbuf.NewBytes(lstnbufgragsz)
			lbf.V(func(b []byte) {
				n, addr, err = conn.ReadFromPeer(b)
			})
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
	buf.V(func(b []byte) {
		h := m.wait(b[:n], addr)
		if !h.HasInit() {
			if config.ShowDebugLog {
				logrus.Debugln("[listen] queue waiting")
			}
			return
		}
		h.B(func(b []byte, p *head.Packet) {
			if config.ShowDebugLog {
				logrus.Debugln("[listen] dispatch", len(b), "bytes packet")
			}
			m.dispatch(p, b, addr)
		})
	})
}

func (m *Me) dispatch(header *head.Packet, body []byte, addr p2p.EndPoint) {
	r := header.Size() - len(body)
	if r > 0 {
		logrus.Warnln("[listen] packet from endpoint", addr, "len", len(body), "is smaller than it declared len", header.Size(), ", drop it")
		return
	}
	srcip := header.Src()
	dstip := header.Dst()
	p := m.extractPeer(srcip, dstip, addr)
	if p == nil {
		return
	}
	if !p.Accept(srcip) {
		logrus.Warnln("[listen] refused packet from", srcip.String()+":"+strconv.Itoa(int(header.SrcPort)))
		return
	}
	if !p.IsToMe(dstip) {
		logrus.Warnln("[listen] unhandled trans packet from", srcip.String()+":"+strconv.Itoa(int(header.SrcPort)))
		return
	}
	addt := header.AdditionalData()
	var err error
	data, err := p.decode(header.CipherIndex(), addt, body)
	if err != nil {
		if config.ShowDebugLog {
			logrus.Debugln("[listen] drop invalid packet key idx:", header.CipherIndex(), "addt:", addt, "err:", err)
		}
		return
	}
	if data.Len() < 8 {
		if config.ShowDebugLog {
			logrus.Debugln("[listen] drop invalid data len packet key idx:", header.CipherIndex(), "addt:", addt, "len", data.Len())
		}
		return
	}
	ok := false
	data.V(func(b []byte) {
		ok = algo.IsVaildBlake2bHash8(header.PreCRC64(), b)
	})
	if !ok {
		if config.ShowDebugLog {
			logrus.Debugln("[listen] drop invalid hash packet")
		}
		return
	}
	data = data.SliceFrom(8)
	if p.usezstd {
		data.V(func(b []byte) {
			data, err = algo.DecodeZstd(b) // skip hash
		})
		if err != nil {
			if config.ShowDebugLog {
				logrus.Debugln("[listen] drop invalid zstd packet:", err)
			}
			return
		}
		if config.ShowDebugLog {
			logrus.Debugln("[listen] zstd decoded len:", data.Len())
		}
	}
	fn, ok := GetDispacher(header.Proto.Proto())
	if !ok {
		logrus.Warnln(file.Header(), "unsupported proto", header.Proto.Proto())
		return
	}
	fn(header, p, data)
}

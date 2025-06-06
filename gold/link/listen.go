package link

import (
	"errors"
	"net"
	"runtime"
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

type job struct {
	addr p2p.EndPoint
	buf  pbuf.Bytes
	n    int
	fil  *uintptr
}

func (m *Me) runworkers() {
	ncpu := runtime.NumCPU()
	m.jobs = make([]chan job, ncpu)
	for i := 0; i < ncpu; i++ {
		m.jobs[i] = make(chan job, 4096)
		go func(i int, jobs <-chan job) {
			for jb := range jobs {
				if config.ShowDebugLog {
					logrus.Debugln("[listen] job thread", i, "call waitordispatch")
				}
				m.waitordispatch(jb.addr, jb.buf, jb.n, jb.fil)
				if config.ShowDebugLog {
					logrus.Debugln("[listen] job thread", i, "fin waitordispatch")
				}
			}
		}(i, m.jobs[i])
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
	ncpu := runtime.NumCPU()
	bufs := make([]byte, lstnbufgragsz*ncpu)
	fils := make([]uintptr, ncpu)
	go m.runworkers()
	go func() {
		var (
			n    int
			addr p2p.EndPoint
			err  error
		)
		for {
			idx := -1
			for i := 0; i < ncpu; i++ {
				if !atomic.CompareAndSwapUintptr(&fils[i], 0, 1) {
					continue
				}
				idx = i
				break
			}

			var (
				lbf pbuf.Bytes
				fil *uintptr
			)
			if idx < 0 {
				lbf = pbuf.NewLargeBytes(lstnbufgragsz)
			} else {
				lbf = pbuf.ParseBytes(bufs[idx*lstnbufgragsz : (idx+1)*lstnbufgragsz : (idx+1)*lstnbufgragsz]...).Ignore()
				fil = &fils[idx]
			}

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
			if idx < 0 {
				if config.ShowDebugLog {
					logrus.Infoln("[listen] go dispatch")
				}
				go m.waitordispatch(addr, lbf, n, fil)
			} else {
				if config.ShowDebugLog {
					logrus.Debugln("[listen] send dispatch to cpu", idx)
				}
				m.jobs[idx] <- job{addr: addr, buf: lbf, n: n, fil: fil}
			}
		}
	}()
	return
}

func (m *Me) waitordispatch(addr p2p.EndPoint, buf pbuf.Bytes, n int, fil *uintptr) {
	defer func() {
		buf.ManualDestroy()
		if fil != nil {
			atomic.StoreUintptr(fil, 0)
		}
	}()

	recvtotlcnt := atomic.AddUint64(&m.recvtotlcnt, uint64(n))
	recvloopcnt := atomic.AddUintptr(&m.recvloopcnt, 1)
	recvlooptime := atomic.LoadInt64(&m.recvlooptime)
	if recvloopcnt%uintptr(m.speedloop) == 0 {
		now := time.Now().UnixMilli()
		kb := float64(recvtotlcnt) / float64(now-recvlooptime)
		if kb < 1024 {
			logrus.Infof("[listen] queue recv avg speed: %.2f KB/s", kb)
		} else {
			kb /= 1024
			if kb < 1024 {
				logrus.Infof("[listen] queue recv avg speed: %.2f MB/s", kb)
			} else {
				logrus.Infof("[listen] queue recv avg speed: %.2f GB/s", kb/1024)
			}
		}
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
			if !p.HasFinished() {
				panic("unexpected unfinished")
			}
			m.dispatch(p, b, addr)
		})
		h.ManualDestroy()
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
	if len(data) < 8 {
		if config.ShowDebugLog {
			logrus.Debugln("[listen] drop invalid data len packet key idx:", header.CipherIndex(), "addt:", addt, "len", len(data))
		}
		return
	}
	ok := false
	ok = algo.IsVaildBlake2bHash8(header.PreCRC64(), data)
	if !ok {
		if config.ShowDebugLog {
			logrus.Debugln("[listen] drop invalid hash packet")
		}
		return
	}
	data = data[8:]
	if p.usezstd {
		data, err = algo.DecodeZstd(data) // skip hash
		if err != nil {
			if config.ShowDebugLog {
				logrus.Debugln("[listen] drop invalid zstd packet:", err)
			}
			return
		}
		if config.ShowDebugLog {
			logrus.Debugln("[listen] zstd decoded len:", len(data))
		}
	}
	fn, ok := GetDispacher(header.Proto.Proto())
	if !ok {
		logrus.Warnln(file.Header(), "unsupported proto", header.Proto.Proto())
		return
	}
	fn(header, p, data)
}

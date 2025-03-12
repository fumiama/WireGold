package link

import (
	"bytes"
	"encoding/hex"
	"io"
	"strconv"

	"github.com/fumiama/WireGold/config"
	"github.com/fumiama/WireGold/gold/head"
	"github.com/fumiama/WireGold/internal/bin"
	base14 "github.com/fumiama/go-base16384"
	"github.com/sirupsen/logrus"
)

// Read 从 peer 收包
func (l *Link) Read() LinkData {
	return <-l.pipe
}

// wait TODO: 判断是否为 trans 并提前 call dispatch
func (m *Me) wait(data []byte) (h head.PacketBytes) {
	if len(data) < int(head.PacketHeadLen)+8 { // not a valid packet
		if config.ShowDebugLog {
			logrus.Debugln("[recv] invalid data len", len(data))
		}
		return
	}
	bound := 64
	endl := "..."
	if len(data) < bound {
		bound = len(data)
		endl = "."
	}
	if config.ShowDebugLog {
		logrus.Debugln("[recv] data bytes, len", len(data), "val", hex.EncodeToString(data[:bound]), endl)
	}
	if m.base14 {
		w := bin.SelectWriter()
		_, err := io.Copy(w, base14.NewDecoder(bytes.NewReader(data)))
		if err != nil { // not a valid packet
			if config.ShowDebugLog {
				logrus.Debugln("[recv] decode base14 err:", err)
			}
			return
		}
		data = w.ToBytes().Trans()
		if len(data) < bound {
			bound = len(data)
			endl = "."
		}
		if config.ShowDebugLog {
			logrus.Debugln("[recv] data b14ed, len", len(data), "val", hex.EncodeToString(data[:bound]), endl)
		}
		if len(data) < int(head.PacketHeadLen)+8 { // not a valid packet
			if config.ShowDebugLog {
				logrus.Debugln("[recv] invalid data len", len(data))
			}
			return
		}
	}
	seq, data := m.xordec(data) // inplace decoding
	if len(data) < bound {
		bound = len(data)
		endl = "."
	}
	if config.ShowDebugLog {
		logrus.Debugln("[recv] data xored, len", len(data), "val", hex.EncodeToString(data[:bound]), endl)
	}
	header, err := head.ParsePacketHeader(data)
	if err != nil { // not a valid packet
		if config.ShowDebugLog {
			logrus.Debugln("[recv] invalid packet header:", err)
		}
		return
	}
	if config.ShowDebugLog {
		logrus.Debugf("[recv] packet seq %08x", seq)
	}
	if _, got := m.recved.GetOrSet(seq, struct{}{}); got {
		if config.ShowDebugLog {
			logrus.Debugln("[recv] ignore duplicated seq packet", strconv.FormatUint(uint64(seq), 16))
		}
		return
	}
	if config.ShowDebugLog {
		header.B(func(_ []byte, p *head.Packet) {
			logrus.Debugln(
				"[recv]", strconv.FormatUint(uint64(seq), 16),
				len(data), "bytes data with protoflag", p.Proto,
				"offset", p.Offset,
			)
		})
	}

	header.B(func(buf []byte, p *head.Packet) {
		if !p.Proto.HasMore() {
			ok := p.WriteDataSegment(data, buf)
			if !ok {
				logrus.Errorln("[recv]", strconv.FormatUint(uint64(seq), 16), "unexpected !ok")
				return
			}
			if config.ShowDebugLog {
				logrus.Debugln("[recv]", strconv.FormatUint(uint64(seq), 16), len(data), "bytes full data waited")
			}
			h = header
			return
		}
	})

	if h.HasInit() {
		return
	}

	h, got := m.recving.GetOrSet(uint16(seq), header)
	if got && h == header {
		panic("unexpected multi-put found")
	}
	if config.ShowDebugLog {
		logrus.Debugln("[recv]", strconv.FormatUint(uint64(seq&0xffff), 16), "get frag part isnew:", !got)
	}
	h.B(func(buf []byte, p *head.Packet) {
		ok := p.WriteDataSegment(data, buf)
		if !ok {
			if config.ShowDebugLog {
				logrus.Debugln("[recv]", strconv.FormatUint(uint64(seq&0xffff), 16), "wait other frag parts isnew:", !got)
			}
			return
		}
		m.recving.Delete(uint16(seq))
		if config.ShowDebugLog {
			logrus.Debugln("[recv]", strconv.FormatUint(uint64(seq&0xffff), 16), "all parts has reached")
		}
	})
	return
}

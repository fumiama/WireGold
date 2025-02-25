package link

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"hash/crc64"
	"io"
	"strconv"

	"github.com/fumiama/WireGold/config"
	"github.com/fumiama/WireGold/gold/head"
	"github.com/fumiama/WireGold/helper"
	base14 "github.com/fumiama/go-base16384"
	"github.com/fumiama/orbyte"
	"github.com/sirupsen/logrus"
)

// Read 从 peer 收包
func (l *Link) Read() *orbyte.Item[head.Packet] {
	return <-l.pipe
}

func (m *Me) wait(data []byte) *orbyte.Item[head.Packet] {
	if len(data) < head.PacketHeadLen { // not a valid packet
		if config.ShowDebugLog {
			logrus.Debugln("[recv] invalid data len", len(data))
		}
		return nil
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
		w := helper.SelectWriter()
		_, err := io.Copy(w, base14.NewDecoder(bytes.NewReader(data)))
		if err != nil { // not a valid packet
			if config.ShowDebugLog {
				logrus.Debugln("[recv] decode base14 err:", err)
			}
			return nil
		}
		data = w.TransUnderlyingBytes()
		if len(data) < bound {
			bound = len(data)
			endl = "."
		}
		if config.ShowDebugLog {
			logrus.Debugln("[recv] data b14ed, len", len(data), "val", hex.EncodeToString(data[:bound]), endl)
		}
		if len(data) < head.PacketHeadLen { // not a valid packet
			if config.ShowDebugLog {
				logrus.Debugln("[recv] invalid data len", len(data))
			}
			return nil
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
		return nil
	}
	if !header.Pointer().Flags.IsValid() {
		if config.ShowDebugLog {
			logrus.Debugln("[recv] drop invalid flags packet:", header.Pointer().Flags)
		}
		return nil
	}
	crc := header.Pointer().CRC64()
	crclog := crc
	crc ^= (uint64(seq) << 16)
	if config.ShowDebugLog {
		logrus.Debugf("[recv] packet crc %016x, seq %08x, xored crc %016x", crclog, seq, crc)
	}
	if _, got := m.recved.GetOrSet(crc, struct{}{}); got {
		if config.ShowDebugLog {
			logrus.Debugln("[recv] ignore duplicated crc packet", strconv.FormatUint(crc, 16))
		}
		return nil
	}
	if config.ShowDebugLog {
		logrus.Debugln(
			"[recv]", strconv.FormatUint(crc, 16),
			len(data), "bytes data with flag", header.Pointer().Flags,
			"offset", header.Pointer().Flags.Offset(),
		)
	}
	if header.Pointer().Flags.IsSingle() || header.Pointer().Flags.NoFrag() {
		ok := header.Pointer().ParseData(data)
		if !ok {
			logrus.Errorln("[recv]", strconv.FormatUint(crc, 16), "unexpected !ok")
			return nil
		}
		if config.ShowDebugLog {
			logrus.Debugln("[recv]", strconv.FormatUint(crc, 16), len(data), "bytes full data waited")
		}
		return header
	}

	crchash := crc64.New(crc64.MakeTable(crc64.ISO))
	_, _ = crchash.Write(head.Hash(data))
	var buf [4]byte
	binary.LittleEndian.PutUint32(buf[:], seq)
	_, _ = crchash.Write(buf[:])
	hsh := crchash.Sum64()
	h, got := m.recving.GetOrSet(hsh, header)
	if got && h == header {
		panic("unexpected multi-put found")
	}
	if config.ShowDebugLog {
		logrus.Debugln("[recv]", strconv.FormatUint(crc, 16), "get frag part of", strconv.FormatUint(hsh, 16), "isnew:", !got)
	}
	ok := h.Pointer().ParseData(data)
	if !ok {
		if config.ShowDebugLog {
			logrus.Debugln("[recv]", strconv.FormatUint(crc, 16), "wait other frag parts of", strconv.FormatUint(hsh, 16), "isnew:", !got)
		}
		return nil
	}
	m.recving.Delete(hsh)
	if config.ShowDebugLog {
		logrus.Debugln("[recv]", strconv.FormatUint(crc, 16), "all parts of", strconv.FormatUint(hsh, 16), "has reached")
	}
	return h
}

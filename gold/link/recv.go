package link

import (
	"encoding/binary"
	"encoding/hex"
	"hash/crc64"
	"strconv"

	"github.com/fumiama/WireGold/config"
	"github.com/fumiama/WireGold/gold/head"
	base14 "github.com/fumiama/go-base16384"
	"github.com/fumiama/orbyte"
	"github.com/fumiama/orbyte/pbuf"
	"github.com/sirupsen/logrus"
)

// Read 从 peer 收包
func (l *Link) Read() *orbyte.Item[head.Packet] {
	return <-l.pipe
}

func (m *Me) wait(data pbuf.Bytes) *orbyte.Item[head.Packet] {
	if data.Len() < head.PacketHeadLen { // not a valid packet
		if config.ShowDebugLog {
			logrus.Debugln("[recv] invalid data len", data.Len())
		}
		return nil
	}
	bound := 64
	endl := "..."
	if data.Len() < bound {
		bound = data.Len()
		endl = "."
	}
	if config.ShowDebugLog {
		logrus.Debugln("[recv] data bytes, len", data.Len(), "val", hex.EncodeToString(data.Bytes()[:bound]), endl)
	}
	if m.base14 {
		data = pbuf.ParseBytes(base14.Decode(data.Bytes())...)
		if data.Len() < bound {
			bound = data.Len()
			endl = "."
		}
		if config.ShowDebugLog {
			logrus.Debugln("[recv] data b14ed, len", data.Len(), "val", hex.EncodeToString(data.Bytes()[:bound]), endl)
		}
		if data.Len() < head.PacketHeadLen { // not a valid packet
			if config.ShowDebugLog {
				logrus.Debugln("[recv] invalid data len", data.Len())
			}
			return nil
		}
	}
	seq, dat := m.xordec(data.Trans().Bytes())
	if len(dat) < bound {
		bound = len(dat)
		endl = "."
	}
	if config.ShowDebugLog {
		logrus.Debugln("[recv] data xored, len", len(dat), "val", hex.EncodeToString(dat[:bound]), endl)
	}
	header, err := head.ParsePacketHeader(dat)
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
			len(dat), "bytes data with flag", header.Pointer().Flags,
			"offset", header.Pointer().Flags.Offset(),
		)
	}
	if header.Pointer().Flags.IsSingle() || header.Pointer().Flags.NoFrag() {
		ok := header.Pointer().ParseData(dat)
		if !ok {
			logrus.Errorln("[recv]", strconv.FormatUint(crc, 16), "unexpected !ok")
			return nil
		}
		if config.ShowDebugLog {
			logrus.Debugln("[recv]", strconv.FormatUint(crc, 16), len(dat), "bytes full data waited")
		}
		return header
	}

	crchash := crc64.New(crc64.MakeTable(crc64.ISO))
	_, _ = crchash.Write(head.Hash(data.Bytes()))
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
	ok := h.Pointer().ParseData(dat)
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

package link

import (
	"encoding/binary"
	"encoding/hex"
	"hash/crc64"
	"strconv"

	"github.com/fumiama/WireGold/config"
	"github.com/fumiama/WireGold/gold/head"
	base14 "github.com/fumiama/go-base16384"
	"github.com/sirupsen/logrus"
)

// Read 从 peer 收包
func (l *Link) Read() *head.Packet {
	return <-l.pipe
}

func (m *Me) wait(data []byte) *head.Packet {
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
		data = base14.Decode(data)
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
	seq, data := m.xordec(data)
	if len(data) < bound {
		bound = len(data)
		endl = "."
	}
	if config.ShowDebugLog {
		logrus.Debugln("[recv] data xored, len", len(data), "val", hex.EncodeToString(data[:bound]), endl)
	}
	if len(data) < head.PacketHeadLen { // not a valid packet
		if config.ShowDebugLog {
			logrus.Debugln("[recv] invalid data len", len(data))
		}
		return nil
	}
	flags := head.Flags(data)
	if !flags.IsValid() {
		if config.ShowDebugLog {
			logrus.Debugln("[recv] drop invalid flags packet:", hex.EncodeToString(data[11:12]), hex.EncodeToString(data[10:11]))
		}
		return nil
	}
	crc := head.CRC64(data)
	crclog := crc
	crc ^= (uint64(seq) << 16)
	if config.ShowDebugLog {
		logrus.Debugf("[recv] packet crc %016x, seq %08x, xored crc %016x", crclog, seq, crc)
	}
	if m.recved.Get(crc) {
		if config.ShowDebugLog {
			logrus.Debugln("[recv] ignore duplicated crc packet", strconv.FormatUint(crc, 16))
		}
		return nil
	}
	m.recved.Set(crc, true)
	if config.ShowDebugLog {
		logrus.Debugln("[recv]", strconv.FormatUint(crc, 16), len(data), "bytes data with flag", hex.EncodeToString(data[11:12]), hex.EncodeToString(data[10:11]))
	}
	if flags.IsSingle() || flags.NoFrag() {
		h := head.SelectPacket()
		_, err := h.Unmarshal(data)
		if err != nil {
			logrus.Errorln("[recv]", strconv.FormatUint(crc, 16), "unmarshal err:", err)
			return nil
		}
		return h
	}

	crchash := crc64.New(crc64.MakeTable(crc64.ISO))
	_, _ = crchash.Write(head.Hash(data))
	var buf [4]byte
	binary.LittleEndian.PutUint32(buf[:], seq)
	_, _ = crchash.Write(buf[:])
	hsh := crchash.Sum64()
	h := m.recving.Get(hsh)
	if h != nil {
		if config.ShowDebugLog {
			logrus.Debugln("[recv]", strconv.FormatUint(crc, 16), "get another frag part of", strconv.FormatUint(hsh, 16))
		}
		ok, err := h.Unmarshal(data)
		if err == nil {
			if ok {
				m.recving.Delete(hsh)
				if config.ShowDebugLog {
					logrus.Debugln("[recv]", strconv.FormatUint(crc, 16), "all parts of", strconv.FormatUint(hsh, 16), "has reached")
				}
				return h
			}
		} else {
			h.Put()
			logrus.Errorln("[recv]", strconv.FormatUint(crc, 16), "unmarshal err:", err)
		}
		return nil
	}
	if config.ShowDebugLog {
		logrus.Debugln("[recv]", strconv.FormatUint(crc, 16), "get new frag part of", strconv.FormatUint(hsh, 16))
	}
	h = head.SelectPacket()
	_, err := h.Unmarshal(data)
	if err != nil {
		h.Put()
		logrus.Errorln("[recv]", strconv.FormatUint(crc, 16), "unmarshal err:", err)
		return nil
	}
	m.recving.Set(hsh, h)
	return nil
}

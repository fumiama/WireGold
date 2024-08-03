package link

import (
	"encoding/binary"
	"encoding/hex"
	"hash/crc64"
	"strconv"

	"github.com/fumiama/WireGold/config"
	"github.com/fumiama/WireGold/gold/head"
	"github.com/sirupsen/logrus"
)

// Read 从 peer 收包
func (l *Link) Read() *head.Packet {
	return <-l.pipe
}

func (m *Me) wait(data []byte) *head.Packet {
	if len(data) < head.PacketHeadLen { // not a valid packet
		return nil
	}
	bound := 64
	endl := "..."
	if len(data) < bound {
		bound = len(data)
		endl = "."
	}
	if config.ShowDebugLog {
		logrus.Debugln("[recv] data bytes", hex.EncodeToString(data[:bound]), endl)
	}
	seq, data := m.xordec(data)
	if config.ShowDebugLog {
		logrus.Debugln("[recv] data xored", hex.EncodeToString(data[:bound]), endl)
	}
	flags := head.Flags(data)
	if !flags.IsValid() {
		if config.ShowDebugLog {
			logrus.Debugln("[recv] drop invalid flags packet:", hex.EncodeToString(data[11:12]), hex.EncodeToString(data[10:11]))
		}
		return nil
	}
	crc := binary.LittleEndian.Uint64(data[52:head.PacketHeadLen])
	crclog := crc
	crc ^= (uint64(seq) << 16)
	if config.ShowDebugLog {
		logrus.Debugf("[recv] packet crc %016x, seq %08x, xored crc %016x", crclog, seq, crc)
	}
	if m.recved.Get(crc) {
		logrus.Warnln("[recv] ignore duplicated crc packet", strconv.FormatUint(crc, 16))
		return nil
	}
	if config.ShowDebugLog {
		logrus.Debugln("[recv]", len(data), "bytes data with flag", hex.EncodeToString(data[11:12]), hex.EncodeToString(data[10:11]))
	}
	if flags.IsSingle() || flags.NoFrag() {
		h := head.SelectPacket()
		_, err := h.Unmarshal(data)
		if err != nil {
			logrus.Errorln("[recv] unmarshal err:", err)
			return nil
		}
		m.recved.Set(crc, true)
		return h
	}

	crchash := crc64.New(crc64.MakeTable(crc64.ISO))
	_, _ = crchash.Write(data[20:52])
	var buf [4]byte
	binary.LittleEndian.PutUint32(buf[:], seq)
	_, _ = crchash.Write(buf[:])
	hsh := crchash.Sum64()
	h := m.recving.Get(hsh)
	if h != nil {
		if config.ShowDebugLog {
			logrus.Debugln("[recv] get another frag part of", strconv.FormatUint(hsh, 16))
		}
		ok, err := h.Unmarshal(data)
		if err == nil {
			if ok {
				m.recving.Delete(hsh)
				m.recved.Set(crc, true)
				if config.ShowDebugLog {
					logrus.Debugln("[recv] all parts of", strconv.FormatUint(hsh, 16), "has reached")
				}
				return h
			}
		} else {
			h.Put()
			logrus.Errorln("[recv] unmarshal err:", err)
		}
		return nil
	}
	if config.ShowDebugLog {
		logrus.Debugln("[recv] get new frag part of", strconv.FormatUint(hsh, 16))
	}
	h = head.SelectPacket()
	_, err := h.Unmarshal(data)
	if err != nil {
		h.Put()
		logrus.Errorln("[recv] unmarshal err:", err)
		return nil
	}
	m.recving.Set(hsh, h)
	m.recved.Set(crc, true)
	return nil
}

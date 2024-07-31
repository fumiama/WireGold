package link

import (
	"encoding/binary"
	"encoding/hex"
	"strconv"
	"unsafe"

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
	logrus.Debugln("[recv] data bytes", hex.EncodeToString(data[:bound]), endl)
	data = m.xordec(data)
	logrus.Debugln("[recv] data xored", hex.EncodeToString(data[:bound]), endl)
	flags := head.Flags(data)
	if !flags.IsValid() {
		logrus.Debugln("[recv] drop invalid flags packet:", hex.EncodeToString(data[11:12]), hex.EncodeToString(data[10:11]))
		return nil
	}
	crc := binary.LittleEndian.Uint64(data[52:head.PacketHeadLen])
	logrus.Debugf("[recv] packet crc %016x", crc)
	if m.recved.Get(crc) { // 是重放攻击
		logrus.Warnln("[recv] ignore duplicated crc packet", strconv.FormatUint(crc, 16))
		return nil
	}
	logrus.Debugln("[recv]", len(data), "bytes data with flag", hex.EncodeToString(data[11:12]), hex.EncodeToString(data[10:11]))
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

	hashd := data[20:52]
	hsh := *(*[32]byte)(*(*unsafe.Pointer)(unsafe.Pointer(&hashd)))
	h := m.recving.Get(hsh)
	if h != nil {
		logrus.Debugln("[recv] get another frag part of", hex.EncodeToString(hashd))
		ok, err := h.Unmarshal(data)
		if err == nil {
			if ok {
				m.recving.Delete(hsh)
				m.recved.Set(crc, true)
				logrus.Debugln("[recv] all parts of", hex.EncodeToString(hashd), "has reached")
				return h
			}
		} else {
			h.Put()
			logrus.Errorln("[recv] unmarshal err:", err)
		}
		return nil
	}
	logrus.Debugln("[recv] get new frag part of", hex.EncodeToString(hashd))
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

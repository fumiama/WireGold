package link

import (
	"encoding/binary"
	"encoding/hex"
	"unsafe"

	"github.com/fumiama/WireGold/gold/head"
	"github.com/sirupsen/logrus"
)

// Read 从 peer 收包
func (l *Link) Read() *head.Packet {
	return <-l.pipe
}

func (m *Me) wait(data []byte) *head.Packet {
	if len(data) < 60 { // not a valid packet
		return nil
	}
	data = m.xor(data)
	flags := binary.LittleEndian.Uint16(data[10:12])
	if flags&0x8000 == 0x8000 { // not a valid packet
		return nil
	}
	crc := binary.LittleEndian.Uint64(data[52:60])
	if m.recved.Get(crc) != 0 { // 是重放攻击
		return nil
	}
	logrus.Debugln("[recv]", len(data), "bytes data with flag", hex.EncodeToString(data[10:12]))
	if flags == 0 || flags == 0x4000 {
		h := head.SelectPacket()
		_, err := h.Unmarshal(data)
		if err != nil {
			logrus.Errorln("[recv] unmarshal err:", err)
			return nil
		}
		m.recved.Set(crc, 1)
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
				m.recved.Set(crc, 1)
				logrus.Debugln("[recv] all parts of", hex.EncodeToString(hashd), "is reached")
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
	return nil
}

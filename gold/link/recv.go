package link

import (
	"encoding/binary"
	"encoding/hex"
	"time"
	"unsafe"

	"github.com/fumiama/WireGold/gold/head"
	"github.com/fumiama/WireGold/helper"
	"github.com/sirupsen/logrus"
	"github.com/wdvxdr1123/ZeroBot/extension/ttl"
)

// Read 从 peer 收包
func (l *Link) Read() *head.Packet {
	return <-l.pipe
}

func (m *Me) initrecvpool() {
	if m.writer == nil {
		m.writer = helper.SelectWriter()
	}
	m.recving = ttl.NewCache[[32]byte, *head.Packet](time.Second * 128)
}

func (m *Me) wait(data []byte) *head.Packet {
	flags := binary.LittleEndian.Uint16(data[10:12])
	logrus.Debugln("[recv]", len(data), "bytes data with flag", hex.EncodeToString(data[10:12]))
	if flags == 0 || flags == 0x4000 {
		h := head.SelectPacket()
		_, err := h.Unmarshal(data)
		if err != nil {
			logrus.Errorln("[recv] unmarshal err:", err)
			return nil
		}
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
				logrus.Debugln("[recv] all parts of", hex.EncodeToString(hashd), "is reached")
				return h
			}
		} else {
			logrus.Errorln("[recv] unmarshal err:", err)
		}
		return nil
	}
	logrus.Debugln("[recv] get new frag part of", hex.EncodeToString(hashd))
	h = head.SelectPacket()
	_, err := h.Unmarshal(data)
	if err != nil {
		logrus.Errorln("[recv] unmarshal err:", err)
		return nil
	}
	m.recving.Set(hsh, h)
	return nil
}

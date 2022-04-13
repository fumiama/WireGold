package link

import (
	"encoding/binary"
	"encoding/hex"
	"time"
	"unsafe"

	"github.com/fumiama/WireGold/gold/head"
	"github.com/sirupsen/logrus"
)

// Read 从 peer 收包
func (l *Link) Read() *head.Packet {
	return <-l.pipe
}

func (m *Me) initrecvpool() {
	if m.recving == nil {
		m.recving = make(map[[32]byte]*head.Packet, 128)
	}
	// 超时定时器
	m.clock = make(map[*head.Packet]uint8, 128)
	var delhs []*head.Packet
	t := time.NewTicker(time.Second)
	for range t.C {
		m.recvmu.Lock()
		for k, v := range m.clock {
			if v > 10 { // 10s
				delete(m.recving, k.Hash)
				delhs = append(delhs, k)
			} else {
				m.clock[k]++
			}
		}
		for _, k := range delhs {
			delete(m.clock, k)
			logrus.Warnln("[recv] drop timeout packet from", k.Src)
		}
		delhs = delhs[:0]
		m.recvmu.Unlock()
	}
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

	m.recvmu.Lock()
	defer m.recvmu.Unlock()
	hashd := data[20:52]
	hsh := *(*[32]byte)(*(*unsafe.Pointer)(unsafe.Pointer(&hashd)))
	h, ok := m.recving[hsh]
	if ok {
		logrus.Debugln("[recv] get another frag part of", hex.EncodeToString(hashd))
		ok, err := h.Unmarshal(data)
		if err == nil {
			if ok {
				delete(m.clock, h)
				delete(m.recving, hsh)
				logrus.Debugln("[recv] all parts of", hex.EncodeToString(hashd), "is reached")
				return h
			}
			m.clock[h] = 0
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
	m.recving[hsh] = h
	m.clock[h] = 0
	return nil
}

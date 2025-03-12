package link

import (
	"github.com/fumiama/orbyte/pbuf"

	"github.com/fumiama/WireGold/internal/algo"
)

func (l *Link) randkeyidx() uint8 {
	if l.keys[1] == nil {
		return 0
	}
	return algo.RandKeyIndex()
}

// decode by aead and put b into pool
func (l *Link) decode(teatype uint8, additional uint16, b []byte) (db pbuf.Bytes, err error) {
	if len(b) == 0 || teatype >= 32 {
		return
	}
	if l.keys[0] == nil {
		return pbuf.ParseBytes(b...).Copy(), nil
	}
	aead := l.keys[teatype]
	if aead == nil {
		return
	}
	return algo.DecodeAEAD(aead, additional, b)
}

// xorenc 按 8 字节, 以初始 m.mask 循环异或编码 data
func (m *Me) xorenc(data []byte, seq uint32) pbuf.Bytes {
	return algo.EncodeXOR(data, m.mask, seq)
}

// xordec 按 8 字节, 以初始 m.mask 循环异或解码 data
func (m *Me) xordec(data []byte) (uint32, []byte) {
	return algo.DecodeXOR(data, m.mask)
}

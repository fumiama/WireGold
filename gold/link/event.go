package link

import (
	"github.com/RomiChan/syncx"
	"github.com/fumiama/WireGold/gold/head"
	"github.com/fumiama/orbyte/pbuf"
)

// 事件分发器
var dispachers syncx.Map[uint8, Dispacher]

type Dispacher func(header *head.Packet, peer *Link, data pbuf.Bytes)

// RegisterDispacher of proto
func RegisterDispacher(p uint8, d Dispacher) (actual Dispacher, hasexist bool) {
	return dispachers.LoadOrStore(p, d)
}

// GetDispacher fn, ok
func GetDispacher(p uint8) (Dispacher, bool) {
	return dispachers.Load(p)
}

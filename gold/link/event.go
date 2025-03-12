package link

import (
	"strconv"

	"github.com/fumiama/WireGold/gold/head"
	"github.com/fumiama/orbyte/pbuf"
)

// 事件分发器
var dispachers map[uint8]EventDispacher = make(map[uint8]EventDispacher)

type EventDispacher func(header *head.Packet, peer *Link, data pbuf.Bytes)

// AddProto is thread unsafe. Use in init() only.
func AddProto(p uint8, d EventDispacher) {
	_, ok := dispachers[p]
	if ok {
		panic("proto " + strconv.Itoa(int(p)) + " has been registered")
	}
	dispachers[p] = d
}

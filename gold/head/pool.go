package head

import (
	"sync"

	"github.com/fumiama/WireGold/helper"
)

var packetPool = sync.Pool{
	New: func() interface{} {
		return new(Packet)
	},
}

// SelectPacket 从池中取出一个 Packet
func SelectPacket() *Packet {
	return packetPool.Get().(*Packet)
}

// PutPacket 将 Packet 放回池中
func PutPacket(p *Packet) {
	p.idxdatsz = 0
	if p.buffered {
		helper.PutBytes(p.data)
		p.buffered = false
	}
	p.a, p.b = 0, 0
	p.data = nil
	p.rembytes = 0
	packetPool.Put(p)
}

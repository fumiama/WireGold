package head

import "sync"

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
	p.Data = nil
	packetPool.Put(p)
}

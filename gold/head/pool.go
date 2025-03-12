package head

import (
	"github.com/fumiama/orbyte/pbuf"
)

var packetPool = pbuf.NewBufferPool[Packet]()

// selectPacket 从池中取出一个 Packet
func selectPacket(buf ...byte) *PacketItem {
	return (*PacketItem)(packetPool.NewBuffer(buf))
}

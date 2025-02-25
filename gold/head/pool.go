package head

import (
	"github.com/fumiama/orbyte"
	"github.com/fumiama/orbyte/pbuf"
)

type packetPooler struct {
	orbyte.Pooler[Packet]
}

func (packetPooler) New(_ any, pooled Packet) Packet {
	return pooled
}

func (packetPooler) Parse(obj any, _ Packet) Packet {
	return obj.(Packet)
}

func (packetPooler) Reset(p *Packet) {
	p.idxdatsz = 0
	p.data = pbuf.Bytes{}
	p.a, p.b = 0, 0
	p.rembytes = 0
}

func (packetPooler) Copy(dst, src *Packet) {
	*dst = *src
	dst.data = src.data.Copy()
}

var packetPool = orbyte.NewPool[Packet](packetPooler{})

// selectPacket 从池中取出一个 Packet
func selectPacket() *orbyte.Item[Packet] {
	return packetPool.New(nil)
}

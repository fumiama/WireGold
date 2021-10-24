package head

import "encoding/json"

type Packet struct {
	DataSZ  uint32
	Proto   uint8
	TTL     uint8
	SrcPort uint16
	DstPort uint16
	Src     string
	Dst     string
	Data    []byte
}

func NewPacket(proto uint8, srcPort uint16, dstPort uint16, data []byte) *Packet {
	return &Packet{
		Proto:   proto,
		TTL:     255,
		SrcPort: srcPort,
		DstPort: dstPort,
		Data:    data,
	}
}

func (p *Packet) UnMashal(data []byte) error {
	return json.Unmarshal(data, p)
}

func (p *Packet) Mashal(src string, dst string) ([]byte, error) {
	p.DataSZ = uint32(len(p.Data))
	p.Src = src
	p.Dst = dst
	return json.Marshal(p)
}

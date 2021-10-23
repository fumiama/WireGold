package head

type Packet struct {
	Proto   uint8
	SrcPort uint16
	DstPort uint16
	TTL     uint8
	Data    []byte
}

func NewPacket(proto uint8, srcPort uint16, dstPort uint16, data []byte) *Packet {
	return &Packet{
		Proto:   proto,
		SrcPort: srcPort,
		DstPort: dstPort,
		TTL:     255,
		Data:    data,
	}
}

func (p *Packet) UnMashal(data []byte) {

}

func (p *Packet) Mashal() []byte {
	return nil
}

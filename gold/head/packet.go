package head

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

func (p *Packet) UnMashal(data []byte) {

}

func (p *Packet) Mashal(src string, dst string) []byte {
	return nil
}

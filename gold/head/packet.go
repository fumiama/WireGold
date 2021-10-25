package head

import "encoding/json"

// Packet 是发送和接收的最小单位
type Packet struct {
	// DataSZ len(Data)
	DataSZ uint32
	// Proto 详见 head
	Proto uint8
	// TTL is time to live
	TTL uint8
	// SrcPort 源端口
	SrcPort uint16
	// DstPort 目的端口
	DstPort uint16
	// Src 源 ip
	Src string
	// Dst 目的 ip
	Dst string
	// Hash 使用 BLAKE2 生成加密前 Packet 的摘要
	// 生成时 Hash 全 0
	// https://github.com/minio/blake2b-simd
	Hash [32]byte
	// Data 承载的数据
	Data []byte
}

// NewPacket 生成一个新包
func NewPacket(proto uint8, srcPort uint16, dstPort uint16, data []byte) *Packet {
	return &Packet{
		Proto:   proto,
		TTL:     255,
		SrcPort: srcPort,
		DstPort: dstPort,
		Data:    data,
	}
}

// UnMashal 将 data 的数据解码到自身
// 同时通过 Hash 验证数据完整性
func (p *Packet) UnMashal(data []byte) error {
	return json.Unmarshal(data, p)
}

// Mashal 将自身数据编码为 []byte
// 同时生成 Hash
func (p *Packet) Mashal(src string, dst string) ([]byte, error) {
	p.DataSZ = uint32(len(p.Data))
	p.Src = src
	p.Dst = dst
	return json.Marshal(p)
}

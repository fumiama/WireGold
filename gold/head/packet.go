package head

import (
	"encoding/binary"
	"errors"
	"net"
	"unsafe"

	"github.com/fumiama/WireGold/helper"
	blake2b "github.com/minio/blake2b-simd"
)

// Packet 是发送和接收的最小单位
type Packet struct {
	// DataSZ len(Data)
	// 不得超过 65507-head 字节
	DataSZ uint32
	// Proto 详见 head
	Proto uint8
	// TTL is time to live
	TTL uint8
	// SrcPort 源端口
	SrcPort uint16
	// DstPort 目的端口
	DstPort uint16
	// Flags 高3位为标志(xDM)，低13位为分片偏移
	Flags uint16
	// Src 源 ip (ipv4)
	Src net.IP
	// Dst 目的 ip (ipv4)
	Dst net.IP
	// Hash 使用 BLAKE2 生成加密前 Packet 的摘要
	// 生成时 Hash 全 0
	// https://github.com/minio/blake2b-simd
	Hash [32]byte
	// Data 承载的数据
	Data []byte
	// 记录还有多少字节未到达
	rembytes uint32
}

// NewPacket 生成一个新包
func NewPacket(proto uint8, srcPort uint16, dst net.IP, dstPort uint16, data []byte) *Packet {
	// logrus.Debugln("[packet] new: [proto:", proto, ", srcport:", srcPort, ", dstport:", dstPort, ", dst:", dst, ", data:", data)
	return &Packet{
		Proto:   proto,
		TTL:     16,
		SrcPort: srcPort,
		DstPort: dstPort,
		Dst:     dst,
		Data:    data,
	}
}

// Unmarshal 将 data 的数据解码到自身
func (p *Packet) Unmarshal(data []byte) (complete bool, err error) {
	if len(data) < 12 {
		err = errors.New("data len < 12")
		return
	}

	if p.DataSZ == 0 && len(p.Data) == 0 {
		p.DataSZ = binary.LittleEndian.Uint32(data[:4])
		if int(p.DataSZ)+52 == len(data) {
			p.Data = data[52:]
			p.rembytes = 0
		} else {
			p.Data = make([]byte, p.DataSZ)
			p.rembytes = p.DataSZ
		}
		pt := binary.LittleEndian.Uint16(data[4:6])
		p.Proto = uint8(pt)
		p.TTL = uint8(pt >> 8)
		p.SrcPort = binary.LittleEndian.Uint16(data[6:8])
		p.DstPort = binary.LittleEndian.Uint16(data[8:10])
	}

	flags := binary.LittleEndian.Uint16(data[10:12])

	if flags&0x1fff == 0 {
		p.Flags = flags
		p.Src = make(net.IP, 4)
		copy(p.Src, data[12:16])
		p.Dst = make(net.IP, 4)
		copy(p.Dst, data[16:20])
		copy(p.Hash[:], data[20:52])
	}

	if p.rembytes > 0 {
		p.rembytes -= uint32(copy(p.Data[flags<<3:], data[52:]))
	}

	complete = p.rembytes == 0

	return
}

// Marshal 将自身数据编码为 []byte
// offset 必须为 8 的倍数，表示偏移的 8 位
func (p *Packet) Marshal(src net.IP, datasz uint32, offset uint16, dontfrag, hasmore bool) ([]byte, func()) {
	p.TTL--
	if p.TTL == 0 {
		return nil, nil
	}

	if src != nil {
		p.DataSZ = datasz
		p.Src = src
		if dontfrag {
			offset |= 0x4000
		}
		if hasmore {
			offset |= 0x2000
		}
		p.Flags = offset
	}

	return helper.OpenWriterF(func(w *helper.Writer) {
		w.WriteUInt32(p.DataSZ)
		w.WriteUInt16((uint16(p.TTL) << 8) | uint16(p.Proto))
		w.WriteUInt16(p.SrcPort)
		w.WriteUInt16(p.DstPort)
		w.WriteUInt16(p.Flags)
		w.Write(p.Src.To4())
		w.Write(p.Dst.To4())
		w.Write(p.Hash[:])
		w.Write(p.Data)
	})
}

// FillHash 生成 p.Data 的 Hash
func (p *Packet) FillHash() {
	sum := blake2b.New256().Sum(p.Data)
	p.Hash = *(*[32]byte)(*(*unsafe.Pointer)(unsafe.Pointer(&sum)))
}

// IsVaildHash 验证 packet 合法性
func (p *Packet) IsVaildHash() bool {
	sum := blake2b.New256().Sum(p.Data)
	return *(*[32]byte)(*(*unsafe.Pointer)(unsafe.Pointer(&sum))) == p.Hash
}

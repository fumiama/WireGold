package head

import (
	"encoding/binary"
	"errors"
	"net"
	"unsafe"

	blake2b "github.com/minio/blake2b-simd"
	"github.com/sirupsen/logrus"
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
	// Src 源 ip
	Src net.IP
	// Dst 目的 ip
	Dst net.IP
	// Hash 使用 BLAKE2 生成加密前 Packet 的摘要
	// 生成时 Hash 全 0
	// https://github.com/minio/blake2b-simd
	Hash [32]byte
	// Data 承载的数据
	Data []byte
}

// NewPacket 生成一个新包
func NewPacket(proto uint8, srcPort uint16, dst net.IP, dstPort uint16, data []byte) *Packet {
	logrus.Debugln("[packet] new: [proto:", proto, ", srcport:", srcPort, ", dstport:", dstPort, ", dst:", dst, ", data:", data)
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
func (p *Packet) Unmarshal(data []byte) error {
	if len(data) < 12 {
		return errors.New("data len < 12")
	}
	p.DataSZ = binary.LittleEndian.Uint32(data[:4])
	pt := binary.LittleEndian.Uint16(data[4:6])
	p.Proto = uint8(pt)
	p.TTL = uint8(pt >> 8)
	p.SrcPort = binary.LittleEndian.Uint16(data[6:8])
	p.DstPort = binary.LittleEndian.Uint16(data[8:10])
	sdl := binary.LittleEndian.Uint16(data[10:12])
	srclen := uint8(sdl)
	dstlen := uint8(sdl >> 8)
	if len(data) < int(12+srclen+dstlen) {
		return errors.New("data src or dst len mismatch")
	}
	if srclen > 0 {
		p.Src = make(net.IP, srclen)
		copy(p.Src, data[12:12+srclen])
	}
	if dstlen > 0 {
		p.Dst = make(net.IP, dstlen)
		copy(p.Dst, data[12+srclen:12+srclen+dstlen])
	}
	copy(p.Hash[:], data[12+srclen+dstlen:12+srclen+dstlen+32])
	p.Data = data[12+srclen+dstlen+32:]
	return nil
}

// Marshal 将自身数据编码为 []byte
func (p *Packet) Marshal(src net.IP) []byte {
	p.TTL--
	if p.TTL == 0 {
		return nil
	}

	p.DataSZ = uint32(len(p.Data))
	if src != nil {
		p.Src = src
	}

	packet := make([]byte, 52+len(p.Data))
	binary.LittleEndian.PutUint32(packet[:4], p.DataSZ)
	binary.LittleEndian.PutUint16(packet[4:6], (uint16(p.TTL)<<8)|uint16(p.Proto))
	binary.LittleEndian.PutUint16(packet[6:8], p.SrcPort)
	binary.LittleEndian.PutUint16(packet[8:10], p.DstPort)
	binary.LittleEndian.PutUint16(packet[10:12], 0x0404)
	copy(packet[12:16], p.Src.To4())
	copy(packet[16:20], p.Dst.To4())
	copy(packet[20:52], p.Hash[:])
	copy(packet[52:], p.Data)

	// logrus.Debugln("[packet] marshaled packet:", hex.EncodeToString(packet))

	return packet
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

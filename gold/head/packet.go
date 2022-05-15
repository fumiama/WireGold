package head

import (
	"encoding/binary"
	"encoding/hex"
	"errors"
	"hash/crc64"
	"net"

	"github.com/fumiama/WireGold/helper"
	blake2b "github.com/fumiama/blake2b-simd"
	"github.com/sirupsen/logrus"
)

// Packet 是发送和接收的最小单位
type Packet struct {
	// TeaTypeDataSZ len(Data)
	// 高 8 位指定加密所用 tea key
	// 不得超过 65507-head 字节
	TeaTypeDataSZ uint32
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
	// https://github.com/fumiama/blake2b-simd
	Hash [32]byte
	// Data 承载的数据
	Data []byte
	// 记录还有多少字节未到达
	rembytes uint32
}

// NewPacket 生成一个新包
func NewPacket(proto uint8, srcPort uint16, dst net.IP, dstPort uint16, data []byte) (p *Packet) {
	// logrus.Debugln("[packet] new: [proto:", proto, ", srcport:", srcPort, ", dstport:", dstPort, ", dst:", dst, ", data:", data)
	p = SelectPacket()
	p.Proto = proto
	p.TTL = 16
	p.SrcPort = srcPort
	p.DstPort = dstPort
	p.Dst = dst
	p.Data = data
	return
}

// Unmarshal 将 data 的数据解码到自身
func (p *Packet) Unmarshal(data []byte) (complete bool, err error) {
	if len(data) < 60 {
		err = errors.New("data len < 60")
		return
	}
	if crc64.Checksum(data[:52], crc64.MakeTable(crc64.ISO)) != binary.LittleEndian.Uint64(data[52:60]) {
		err = errors.New("bad crc checksum")
		return
	}

	sz := p.TeaTypeDataSZ & 0x00ffffff
	if sz == 0 && len(p.Data) == 0 {
		p.TeaTypeDataSZ = binary.LittleEndian.Uint32(data[:4])
		sz = p.TeaTypeDataSZ & 0x00ffffff
		if int(sz)+52 == len(data) {
			p.Data = data[52:]
			p.rembytes = 0
		} else {
			p.Data = make([]byte, sz)
			p.rembytes = sz
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
		p.rembytes -= uint32(copy(p.Data[flags<<3:], data[60:]))
	}

	complete = p.rembytes == 0

	return
}

// Marshal 将自身数据编码为 []byte
// offset 必须为 8 的倍数，表示偏移的 8 位
func (p *Packet) Marshal(src net.IP, teatype uint8, datasz uint32, offset uint16, dontfrag, hasmore bool) ([]byte, func()) {
	p.TTL--
	if p.TTL == 0 {
		return nil, nil
	}

	if src != nil {
		p.TeaTypeDataSZ = uint32(teatype)<<24 | datasz
		p.Src = src
		offset &= 0x1fff
		if dontfrag {
			offset |= 0x4000
		}
		if hasmore {
			offset |= 0x2000
		}
		p.Flags = offset
	}

	return helper.OpenWriterF(func(w *helper.Writer) {
		w.WriteUInt32(p.TeaTypeDataSZ)
		w.WriteUInt16((uint16(p.TTL) << 8) | uint16(p.Proto))
		w.WriteUInt16(p.SrcPort)
		w.WriteUInt16(p.DstPort)
		w.WriteUInt16(p.Flags)
		w.Write(p.Src.To4())
		w.Write(p.Dst.To4())
		w.Write(p.Hash[:])
		w.WriteUInt64(crc64.Checksum(w.Bytes(), crc64.MakeTable(crc64.ISO)))
		w.Write(p.Data)
	})
}

// FillHash 生成 p.Data 的 Hash
func (p *Packet) FillHash() {
	h := blake2b.New256()
	_, err := h.Write(p.Data)
	if err != nil {
		logrus.Error("[packet] err when fill hash:", err)
		return
	}
	logrus.Debugln("[packet] sum calulated:", hex.EncodeToString(h.Sum(p.Hash[:0])))
}

// IsVaildHash 验证 packet 合法性
func (p *Packet) IsVaildHash() bool {
	h := blake2b.New256()
	_, err := h.Write(p.Data)
	if err != nil {
		logrus.Error("[packet] err when check hash:", err)
		return false
	}
	var sum [32]byte
	logrus.Debugln("[packet] sum calulated:", hex.EncodeToString(h.Sum(sum[:0])))
	logrus.Debugln("[packet] sum in packet:", hex.EncodeToString(p.Hash[:]))
	return sum == p.Hash
}

// Put 将自己放回池中
func (p *Packet) Put() {
	PutPacket(p)
}

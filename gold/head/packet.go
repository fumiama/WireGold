package head

import (
	"encoding/binary"
	"encoding/hex"
	"errors"
	"hash/crc64"
	"net"

	blake2b "github.com/fumiama/blake2b-simd"
	"github.com/sirupsen/logrus"

	"github.com/fumiama/WireGold/config"
	"github.com/fumiama/WireGold/helper"
)

const PacketHeadLen = 60

var (
	ErrBadCRCChecksum = errors.New("bad crc checksum")
	ErrDataLenLT60    = errors.New("data len < 60")
)

type PacketFlags uint16

func (pf PacketFlags) IsValid() bool {
	return pf&0x8000 == 0
}

func (pf PacketFlags) DontFrag() bool {
	return pf&0x4000 == 0x4000
}

func (pf PacketFlags) NoFrag() bool {
	return pf == 0x4000
}

func (pf PacketFlags) IsSingle() bool {
	return pf == 0
}

func (pf PacketFlags) ZeroOffset() bool {
	return pf&0x1fff == 0
}

func (pf PacketFlags) Offset() uint16 {
	return uint16(pf << 3)
}

// Flags extract flags from raw data
func Flags(data []byte) PacketFlags {
	return PacketFlags(binary.LittleEndian.Uint16(data[10:12]))
}

// Packet 是发送和接收的最小单位
type Packet struct {
	// idxdatsz len(Data)
	// 高 5 位指定加密所用 key index
	// 高 5-16 位是递增值, 用于 xchacha20 验证 additionalData
	// 不得超过 65507-head 字节
	idxdatsz uint32
	// Proto 详见 head
	Proto uint8
	// TTL is time to live
	TTL uint8
	// SrcPort 源端口
	SrcPort uint16
	// DstPort 目的端口
	DstPort uint16
	// Flags 高3位为标志(xDM)，低13位为分片偏移
	Flags PacketFlags
	// Src 源 ip (ipv4)
	Src net.IP
	// Dst 目的 ip (ipv4)
	Dst net.IP
	// Hash 使用 BLAKE2 生成加密前 Packet 的摘要
	// 生成时 Hash 全 0
	// https://github.com/fumiama/blake2b-simd
	Hash [32]byte
	// crc64 包头字段的 checksum 值，可以认为在一定时间内唯一
	crc64 uint64
	// data 承载的数据
	data []byte
	// Data 当前的偏移
	a, b int
	// 记录还有多少字节未到达
	rembytes int
	// 是否经由 helper.MakeBytes 创建 Data
	buffered bool
}

// NewPacket 生成一个新包
func NewPacket(proto uint8, srcPort uint16, dst net.IP, dstPort uint16, data []byte) (p *Packet) {
	p = SelectPacket()
	p.Proto = proto
	p.TTL = 16
	p.SrcPort = srcPort
	p.DstPort = dstPort
	p.Dst = dst
	p.data = data
	p.b = len(data)
	return
}

// Unmarshal 将 data 的数据解码到自身
func (p *Packet) Unmarshal(data []byte) (complete bool, err error) {
	if len(data) < 60 {
		err = ErrDataLenLT60
		return
	}
	p.crc64 = binary.LittleEndian.Uint64(data[52:PacketHeadLen])
	if crc64.Checksum(data[:52], crc64.MakeTable(crc64.ISO)) != p.crc64 {
		err = ErrBadCRCChecksum
		return
	}

	sz := p.Len()
	if sz == 0 && len(p.data) == 0 {
		p.idxdatsz = binary.LittleEndian.Uint32(data[:4])
		sz = p.Len()
		if sz+52 == len(data) {
			p.data = data[52:]
			p.b = len(p.data)
			p.rembytes = 0
		} else {
			p.data = helper.MakeBytes(sz)
			p.buffered = true
			p.b = sz
			p.rembytes = sz
		}
		pt := binary.LittleEndian.Uint16(data[4:6])
		p.Proto = uint8(pt)
		p.TTL = uint8(pt >> 8)
		p.SrcPort = binary.LittleEndian.Uint16(data[6:8])
		p.DstPort = binary.LittleEndian.Uint16(data[8:10])
	}

	flags := PacketFlags(binary.LittleEndian.Uint16(data[10:12]))

	if flags.ZeroOffset() {
		p.Flags = flags
		p.Src = make(net.IP, 4)
		copy(p.Src, data[12:16])
		p.Dst = make(net.IP, 4)
		copy(p.Dst, data[16:20])
		copy(p.Hash[:], data[20:52])
	}

	if p.rembytes > 0 {
		p.rembytes -= copy(p.data[flags.Offset():], data[PacketHeadLen:])
		if config.ShowDebugLog {
			logrus.Debugln("[packet] copied frag", hex.EncodeToString(p.Hash[:]), "rembytes:", p.rembytes)
		}
	}

	complete = p.rembytes == 0

	return
}

// DecreaseAndGetTTL TTL 自减后返回
func (p *Packet) DecreaseAndGetTTL() uint8 {
	p.TTL--
	return p.TTL
}

// Marshal 将自身数据编码为 []byte
// offset 必须为 8 的倍数，表示偏移的 8 位
func (p *Packet) Marshal(src net.IP, teatype uint8, additional uint16, datasz uint32, offset uint16, dontfrag, hasmore bool) ([]byte, func()) {
	if src != nil {
		p.Src = src
		p.idxdatsz = (uint32(teatype) << 27) | (uint32(additional&0x07ff) << 16) | datasz&0xffff
	}

	offset &= 0x1fff
	if dontfrag {
		offset |= 0x4000
	}
	if hasmore {
		offset |= 0x2000
	}

	return helper.OpenWriterF(func(w *helper.Writer) {
		w.WriteUInt32(p.idxdatsz)
		w.WriteUInt16((uint16(p.TTL) << 8) | uint16(p.Proto))
		w.WriteUInt16(p.SrcPort)
		w.WriteUInt16(p.DstPort)
		w.WriteUInt16(uint16(PacketFlags(offset)))
		w.Write(p.Src.To4())
		w.Write(p.Dst.To4())
		w.Write(p.Hash[:])
		w.WriteUInt64(crc64.Checksum(w.Bytes(), crc64.MakeTable(crc64.ISO)))
		w.Write(p.Body())
	})
}

// FillHash 生成 p.Data 的 Hash
func (p *Packet) FillHash() {
	h := blake2b.New256()
	_, err := h.Write(p.Body())
	if err != nil {
		logrus.Error("[packet] err when fill hash:", err)
		return
	}
	hsh := h.Sum(p.Hash[:0])
	if config.ShowDebugLog {
		logrus.Debugln("[packet] sum calulated:", hex.EncodeToString(hsh))
	}
}

// IsVaildHash 验证 packet 合法性
func (p *Packet) IsVaildHash() bool {
	h := blake2b.New256()
	_, err := h.Write(p.Body())
	if err != nil {
		logrus.Error("[packet] err when check hash:", err)
		return false
	}
	var sum [32]byte
	_ = h.Sum(sum[:0])
	if config.ShowDebugLog {
		logrus.Debugln("[packet] sum calulated:", hex.EncodeToString(sum[:]))
		logrus.Debugln("[packet] sum in packet:", hex.EncodeToString(p.Hash[:]))
	}
	return sum == p.Hash
}

// AdditionalData 获得 packet 的 additionalData
func (p *Packet) AdditionalData() uint16 {
	return uint16((p.idxdatsz >> 16) & 0x07ff)
}

// CipherIndex packet 加密使用的密钥集目录
func (p *Packet) CipherIndex() uint8 {
	return uint8(p.idxdatsz >> 27)
}

// Len is packet size
func (p *Packet) Len() int {
	return int(p.idxdatsz & 0xffff)
}

// Put 将自己放回池中
func (p *Packet) Put() {
	PutPacket(p)
}

// Body returns data
func (p *Packet) Body() []byte {
	return p.data[p.a:p.b]
}

func (p *Packet) BodyLen() int {
	return p.b - p.a
}

func (p *Packet) SetBody(b []byte, buffered bool) {
	p.a = 0
	p.b = len(b)
	if len(b) <= cap(p.data) {
		p.data = p.data[:len(b)]
		copy(p.data, b)
		if buffered {
			helper.PutBytes(b)
		}
		return
	}
	if p.buffered {
		helper.PutBytes(p.data)
	}
	p.data = b
	p.buffered = buffered
}

func (p *Packet) CropBody(a, b int) {
	if b > len(p.data) {
		b = len(p.data)
	}
	if a < 0 || b < 0 || a > b {
		return
	}
	p.a, p.b = a, b
}

func (p *Packet) Copy() *Packet {
	newp := SelectPacket()
	*newp = *p
	newp.buffered = false
	return newp
}

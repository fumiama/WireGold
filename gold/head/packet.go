package head

import (
	"encoding/binary"
	"encoding/hex"
	"errors"
	"net"
	"sync/atomic"

	blake2b "github.com/fumiama/blake2b-simd"
	"github.com/fumiama/orbyte"
	"github.com/fumiama/orbyte/pbuf"
	"github.com/sirupsen/logrus"

	"github.com/fumiama/WireGold/config"
	"github.com/fumiama/WireGold/helper"
)

const PacketHeadLen = 60

var (
	ErrBadCRCChecksum = errors.New("bad crc checksum")
	ErrDataLenLT60    = errors.New("data len < 60")
)

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
	// 记录还有多少字节未到达
	rembytes int32
	// Src 源 ip (ipv4)
	Src net.IP
	// Dst 目的 ip (ipv4)
	Dst net.IP
	// Hash 使用 BLAKE2 生成加密前 Packet 的摘要
	// 生成时 Hash 全 0
	// https://github.com/fumiama/blake2b-simd
	Hash [32]byte
	// crc64 包头字段的 checksum 值，可以认为在一定时间内唯一 (现已更改算法为 md5 但名字未变)
	crc64 uint64
	// data 承载的数据
	data pbuf.Bytes
	// Data 当前的偏移
	a, b int
}

// NewPacketPartial 从一些预设参数生成一个新包
func NewPacketPartial(
	proto uint8, srcPort uint16,
	dst net.IP, dstPort uint16,
	data pbuf.Bytes,
) *orbyte.Item[Packet] {
	p := selectPacket()
	pp := p.Pointer()
	pp.Proto = proto
	pp.TTL = 16
	pp.SrcPort = srcPort
	pp.DstPort = dstPort
	pp.Dst = dst
	pp.data = data
	pp.b = data.Len()
	return p
}

func ParsePacket(p Packet) *orbyte.Item[Packet] {
	return packetPool.Parse(nil, p)
}

func ParsePacketHeader(data []byte) (p *orbyte.Item[Packet], err error) {
	if len(data) < 60 {
		err = ErrDataLenLT60
		return
	}
	p = selectPacket()
	pp := p.Pointer()
	pp.crc64 = CRC64(data)
	if CalcCRC64(data) != pp.crc64 {
		err = ErrBadCRCChecksum
		return
	}

	pp.idxdatsz = binary.LittleEndian.Uint32(data[:4])
	sz := pp.Len()
	if config.ShowDebugLog {
		logrus.Debugln("[packet] header data len", sz, "read data len", len(data))
	}
	pt := binary.LittleEndian.Uint16(data[4:6])
	pp.Proto = uint8(pt)
	pp.TTL = uint8(pt >> 8)
	pp.SrcPort = binary.LittleEndian.Uint16(data[6:8])
	pp.DstPort = binary.LittleEndian.Uint16(data[8:10])

	flags := PacketFlags(binary.LittleEndian.Uint16(data[10:12]))
	pp.Flags = flags
	pp.Src = make(net.IP, 4)
	copy(pp.Src, data[12:16])
	pp.Dst = make(net.IP, 4)
	copy(pp.Dst, data[16:20])
	copy(pp.Hash[:], data[20:52])

	switch {
	case sz+PacketHeadLen == len(data):
		pp.b = sz
		pp.rembytes = -1
	case pp.rembytes == 0:
		pp.data = pbuf.NewBytes(sz)
		pp.b = sz
		pp.rembytes = int32(sz)
	}

	return
}

// ParseData 将 data 的数据解码到自身
//
// 必须先调用 ParsePacketHeader
func (p *Packet) ParseData(data []byte) (complete bool) {
	sz := p.Len()
	if sz+PacketHeadLen == len(data) {
		p.data = pbuf.ParseBytes(data[PacketHeadLen:]...).Copy()
		return true
	}

	flags := PacketFlags(binary.LittleEndian.Uint16(data[10:12]))
	if config.ShowDebugLog {
		logrus.Debugln("[packet] parse data flags", flags, "off", flags.Offset())
	}
	if flags.ZeroOffset() {
		p.Flags = flags
		if config.ShowDebugLog {
			logrus.Debugln("[packet] parse data set zero offset flags", flags)
		}
	}

	rembytes := atomic.LoadInt32(&p.rembytes)
	if rembytes > 0 {
		n := int32(copy(p.data.Bytes()[flags.Offset():], data[PacketHeadLen:]))
		newrem := rembytes - n
		for !atomic.CompareAndSwapInt32(&p.rembytes, rembytes, newrem) {
			rembytes = atomic.LoadInt32(&p.rembytes)
			newrem = rembytes - n
		}
		if config.ShowDebugLog {
			logrus.Debugln("[packet] copied frag", hex.EncodeToString(data[20:52]), "rembytes:", p.rembytes)
		}
	}

	return p.rembytes <= 0
}

// DecreaseAndGetTTL TTL 自减后返回
func (p *Packet) DecreaseAndGetTTL() uint8 {
	p.TTL--
	return p.TTL
}

// MarshalWith 补全剩余参数, 将自身数据编码为 []byte
// offset 必须为 8 的倍数，表示偏移的 8 位
func (p *Packet) MarshalWith(
	src net.IP, teatype uint8, additional uint16,
	datasz uint32, offset uint16,
	dontfrag, hasmore bool,
) pbuf.Bytes {
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
	p.Flags = PacketFlags(offset)
	return helper.NewWriterF(func(w *helper.Writer) {
		w.WriteUInt32(p.idxdatsz)
		w.WriteUInt16((uint16(p.TTL) << 8) | uint16(p.Proto))
		w.WriteUInt16(p.SrcPort)
		w.WriteUInt16(p.DstPort)
		w.WriteUInt16(uint16(p.Flags))
		w.Write(p.Src.To4())
		w.Write(p.Dst.To4())
		w.Write(p.Hash[:])
		p.crc64 = CalcCRC64(w.UnsafeBytes())
		w.WriteUInt64(p.crc64)
		w.Write(p.UnsafeBody())
	})
}

// FillHash 生成 p.Data 的 Hash
func (p *Packet) FillHash() {
	h := blake2b.New256()
	_, err := h.Write(p.UnsafeBody())
	if err != nil {
		logrus.Errorln("[packet] err when fill hash:", err)
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
	_, err := h.Write(p.UnsafeBody())
	if err != nil {
		logrus.Errorln("[packet] err when check hash:", err)
		return false
	}
	var sum [32]byte
	_ = h.Sum(sum[:0])
	if config.ShowDebugLog {
		logrus.Debugln("[packet] sum data len:", len(p.UnsafeBody()))
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

func (p *Packet) CRC64() uint64 {
	return p.crc64
}

// TransBody returns item.Trans().Slice()
func (p *Packet) TransBody() pbuf.Bytes {
	d := p.data.Trans().Slice(p.a, p.b)
	p.data = pbuf.Bytes{}
	return d
}

// UnsafeBody returns data
func (p *Packet) UnsafeBody() []byte {
	return p.data.Bytes()[p.a:p.b]
}

func (p *Packet) BodyLen() int {
	return p.b - p.a
}

func (p *Packet) SetBody(b []byte) {
	p.a = 0
	p.b = len(b)
	p.data = pbuf.ParseBytes(b...)
}

func (p *Packet) CropBody(a, b int) {
	if b > p.data.Len() {
		b = p.data.Len()
	}
	if a < 0 || b < 0 || a > b {
		return
	}
	p.a, p.b = a, b
}

func (p *Packet) ShallowCopy() (newp Packet) {
	newp = *p
	newp.data = p.data.Ref()
	return newp
}

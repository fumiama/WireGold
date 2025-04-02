package head

import (
	"errors"
	"net"
	"sync/atomic"
	"unsafe"

	"github.com/fumiama/orbyte"
	"github.com/fumiama/orbyte/pbuf"
)

const (
	// PacketHeadPreCRCIdx skip idxdatsz, which will be set at Seal().
	PacketHeadPreCRCIdx = unsafe.Offsetof(Packet{}.randn)
	// PacketHeadNoCRCLen without final crc
	PacketHeadNoCRCLen = unsafe.Offsetof(Packet{}.md5h8)
	PacketHeadLen      = unsafe.Offsetof(Packet{}.hashrem)
)

var (
	ErrBadCRCChecksum  = errors.New("bad crc checksum")
	ErrDataLenLEHeader = errors.New("data len <= header len")
	ErrInvalidOffset   = errors.New("invalid offset")
)

type (
	PacketBuf   = pbuf.UserBuffer[Packet]
	PacketItem  = orbyte.Item[PacketBuf]
	PacketBytes = pbuf.UserBytes[Packet]
)

// Packet 是发送和接收的最小单位
type Packet struct {
	// idxdatsz
	//
	// idx
	// 高 5 位指定加密所用 key index
	// 高 5-16 位是递增值, 用于 xchacha20 验证 additionalData
	//
	// datsz
	// 不得超过 65507-head 字节
	idxdatsz uint32
	// randn
	// 在发送报文时填入随机值.
	randn int32
	// Proto 高3位为标志(xDM)，低5位为协议类型
	Proto FlagsProto
	// TTL is time to live
	TTL uint8
	// SrcPort 源端口
	SrcPort uint16
	// DstPort 目的端口
	DstPort uint16
	// Offset 分片偏移量
	Offset uint16
	// src 源 ip (ipv4)
	src [4]byte
	// dst 目的 ip (ipv4)
	dst [4]byte
	// md5h8 发送时记录包头字段除自身外的 checksum 值.
	//
	// 可以认为在一定时间内唯一 (现已更改算法为 md5 但名字未变)。
	md5h8 uint64

	// 以下字段为包体, 与 data 一起加密

	// hashrem 使用 BLAKE2B 生成加密前 packet data+crc64 的摘要,
	// 取其前 8 字节, 小端序读写. 接收时记录剩余字节数.
	//
	// https://github.com/fumiama/blake2b-simd
	hashrem int64

	// Buffer 用于 builder with 暂存原始包体数据
	// 以及接收时保存 body, 通过 PacketBytes 截取偏移.
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
func (p *Packet) Size() int {
	return int(p.idxdatsz & 0xffff)
}

// CRC64 extract md5h8rem field
func (p *Packet) CRC64() uint64 {
	return p.md5h8
}

func (p *Packet) Src() net.IP {
	return append(net.IP{}, p.src[:]...)
}

func (p *Packet) Dst() net.IP {
	return append(net.IP{}, p.dst[:]...)
}

func (p *Packet) HasFinished() bool {
	return atomic.LoadInt64(&p.hashrem) <= 0
}

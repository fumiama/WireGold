package head

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"unsafe"

	"github.com/fumiama/orbyte/pbuf"
	"github.com/sirupsen/logrus"

	"github.com/fumiama/WireGold/config"
	"github.com/fumiama/WireGold/internal/algo"
	"github.com/fumiama/WireGold/internal/bin"
)

// PreCRC64 calculate crc64 checksum without idxdatsz.
func (p *Packet) PreCRC64() (crc uint64) {
	w := bin.SelectWriter()
	// 固定 TTL 为 0, flag 为空 计算
	if bin.IsLittleEndian {
		ttl := p.TTL
		f := p.Proto
		p.TTL = 0
		p.Proto &= protobit
		w.Write((*[PacketHeadNoCRCLen]byte)(
			(unsafe.Pointer)(p),
		)[:])
		p.TTL = ttl
		p.Proto = f
	} else {
		w.WriteUInt32(p.idxdatsz)
		w.WriteUInt32(uint32(p.randn))
		w.WriteUInt16(uint16(p.Proto & protobit)) // TTL, flags is set to 0
		w.WriteUInt16(p.SrcPort)
		w.WriteUInt16(p.DstPort)
		w.WriteUInt16(p.Offset)
		w.Write(p.src[:])
		w.Write(p.dst[:])
	}
	w.P(func(b *pbuf.Buffer) {
		crc = algo.MD5Hash8(b.Bytes()[PacketHeadPreCRCIdx:])
		if config.ShowDebugLog {
			logrus.Debugf(
				"[box] calc pre-crc64 %016x, dat %s", crc,
				hex.EncodeToString(b.Bytes()[PacketHeadPreCRCIdx:]),
			)
		}
	})
	return
}

// WriteHeaderTo write header bytes to buf
// with crc64 checksum.
func (p *Packet) WriteHeaderTo(buf *bytes.Buffer) {
	// 固定 TTL 为 0 计算
	if bin.IsLittleEndian {
		buf.Write((*[PacketHeadNoCRCLen]byte)(
			(unsafe.Pointer)(p),
		)[:])
		pbuf.NewBytes(buf.Len()).V(func(b []byte) {
			copy(b, buf.Bytes())
			ClearTTL(b)
			p.md5h8 = algo.MD5Hash8(b)
		})
		_ = binary.Write(buf, binary.LittleEndian, p.md5h8)
		return
	}
	w := bin.SelectWriter()
	w.WriteUInt32(p.idxdatsz)
	w.WriteUInt32(uint32(p.randn))
	w.WriteUInt16((uint16(p.TTL) << 8) | uint16(p.Proto))
	w.WriteUInt16(p.SrcPort)
	w.WriteUInt16(p.DstPort)
	w.WriteUInt16(p.Offset)
	w.Write(p.src[:])
	w.Write(p.dst[:])
	w.P(func(buf *pbuf.Buffer) {
		pbuf.NewBytes(buf.Len()).V(func(b []byte) {
			copy(b, buf.Bytes())
			ClearTTL(b)
			p.md5h8 = algo.MD5Hash8(b)
		})
	})
	w.WriteUInt64(p.md5h8)
	w.P(func(b *pbuf.Buffer) {
		_, _ = buf.ReadFrom(b)
	})
}

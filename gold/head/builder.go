package head

import (
	"crypto/cipher"
	"crypto/rand"
	"encoding/binary"
	"net"
	"strconv"

	"github.com/fumiama/orbyte/pbuf"
	"github.com/sirupsen/logrus"

	"github.com/fumiama/WireGold/config"
	"github.com/fumiama/WireGold/internal/algo"
	"github.com/fumiama/WireGold/internal/bin"
	"github.com/fumiama/WireGold/internal/file"
)

type (
	HeaderBuilder PacketItem
	DataBuilder   PacketItem
	PacketBuilder PacketItem
)

func NewPacketBuilder() *HeaderBuilder {
	p := selectPacket()
	p.P(func(ub *PacketBuf) {
		err := binary.Read(
			rand.Reader, binary.LittleEndian, &ub.DAT.randn,
		)
		if err != nil {
			panic(err)
		}
	})
	return (*HeaderBuilder)(p)
}

func (pb *HeaderBuilder) p(f func(*PacketBuf)) *HeaderBuilder {
	(*PacketItem)(pb).P(f)
	return pb
}

func (pb *HeaderBuilder) Proto(proto uint8) *HeaderBuilder {
	return pb.p(func(ub *PacketBuf) {
		ub.DAT.Proto |= FlagsProto(proto) & protobit
	})
}

func (pb *HeaderBuilder) TTL(ttl uint8) *HeaderBuilder {
	return pb.p(func(ub *PacketBuf) {
		ub.DAT.TTL = ttl
	})
}

func (pb *HeaderBuilder) Src(ip net.IP, p uint16) *HeaderBuilder {
	return pb.p(func(ub *PacketBuf) {
		copy(ub.DAT.src[:], ip.To4())
		ub.DAT.SrcPort = p
	})
}

func (pb *HeaderBuilder) Dst(ip net.IP, p uint16) *HeaderBuilder {
	return pb.p(func(ub *PacketBuf) {
		copy(ub.DAT.dst[:], ip.To4())
		ub.DAT.DstPort = p
	})
}

func (pb *HeaderBuilder) With(data []byte) *DataBuilder {
	return (*DataBuilder)(pb.p(func(ub *PacketBuf) {
		// header crc64 except idxdatasz
		ub.DAT.md5h8rem = int64(ub.DAT.PreCRC64())
		// plain data
		ub.Buffer.Write(data)
		if config.ShowDebugLog {
			logrus.Debugln(file.Header(), strconv.FormatUint(uint64(ub.DAT.md5h8rem), 16), "build with data", file.ToLimitHexString(data, 64))
		}
	}))
}

func (pb *DataBuilder) p(f func(*PacketBuf)) *DataBuilder {
	(*PacketItem)(pb).P(f)
	return pb
}

func (pb *DataBuilder) Zstd() *DataBuilder {
	return pb.p(func(ub *PacketBuf) {
		data := algo.EncodeZstd(ub.Bytes())
		ub.Reset()
		data.V(func(b []byte) { ub.Write(b) })
		if config.ShowDebugLog {
			logrus.Debugln(file.Header(), strconv.FormatUint(uint64(ub.DAT.md5h8rem), 16), "data after zstd", file.ToLimitHexString(ub.Bytes(), 64))
		}
	})
}

func (pb *DataBuilder) Hash() *DataBuilder {
	return pb.p(func(ub *PacketBuf) {
		ub.DAT.hash = algo.Blake2bHash8(
			uint64(ub.DAT.md5h8rem), ub.Bytes(),
		)
	})
}

func (pb *DataBuilder) tea(typ uint8) *DataBuilder {
	return pb.p(func(ub *PacketBuf) {
		ub.DAT.idxdatsz |= (uint32(typ) << 27)
	})
}

func (pb *DataBuilder) additional(additional uint16) *DataBuilder {
	return pb.p(func(ub *PacketBuf) {
		ub.DAT.idxdatsz |= (uint32(additional&0x07ff) << 16)
	})
}

func (pb *DataBuilder) Seal(aead cipher.AEAD, teatyp uint8, additional uint16) *PacketBuilder {
	return (*PacketBuilder)(pb.tea(teatyp).additional(additional).
		p(func(ub *PacketBuf) {
			// encrypted data: chacha20(hash + plain)
			w := bin.SelectWriter()
			w.WriteUInt64(ub.DAT.hash)
			w.Write(ub.Bytes())
			w.P(func(b *pbuf.Buffer) {
				data := algo.EncodeAEAD(aead, additional, b.Bytes())
				ub.Reset()
				data.V(func(b []byte) { ub.Write(b) })
			})
		}))
}

func (pb *DataBuilder) Plain(teatyp uint8, additional uint16) *PacketBuilder {
	return (*PacketBuilder)(pb.tea(teatyp).additional(additional).
		p(func(ub *PacketBuf) {
			w := bin.SelectWriter()
			w.WriteUInt64(ub.DAT.hash)
			w.Write(ub.Bytes())
			w.P(func(b *pbuf.Buffer) {
				ub.Reset()
				ub.ReadFrom(b)
			})
		}))
}

func (pb *DataBuilder) Trans(teatyp uint8, additional uint16) *PacketBuilder {
	return (*PacketBuilder)(pb.tea(teatyp).additional(additional))
}

func (pb *PacketBuilder) copy() *PacketBuilder {
	return (*PacketBuilder)((*PacketItem)(pb).Copy())
}

func (pb *PacketBuilder) p(f func(*PacketBuf)) *PacketBuilder {
	(*PacketItem)(pb).P(f)
	return pb
}

// datasize fill encrypted datasize by calling data.Len().
func (pb *PacketBuilder) datasize() *PacketBuilder {
	return pb.p(func(ub *PacketBuf) {
		l := uint32(ub.Len()) & 0xffff
		ub.DAT.idxdatsz |= l
	})
}

func (pb *PacketBuilder) noFrag(on bool) *PacketBuilder {
	return pb.p(func(ub *PacketBuf) {
		if on {
			ub.DAT.Proto |= nofragbit
		} else {
			ub.DAT.Proto &= ^nofragbit
		}
	})
}

func (pb *PacketBuilder) hasMore(on bool) *PacketBuilder {
	return pb.p(func(ub *PacketBuf) {
		if on {
			ub.DAT.Proto |= hasmorebit
		} else {
			ub.DAT.Proto &= ^hasmorebit
		}
	})
}

func (pb *PacketBuilder) offset(off uint16) *PacketBuilder {
	return pb.p(func(ub *PacketBuf) {
		ub.DAT.Offset = off
	})
}

// Split mtu based on the total len, which includes
// header and body and padding after outer xor.
func (pb *PacketBuilder) Split(mtu int, nofrag bool) (pbs []PacketBytes) {
	pb.datasize().p(func(ub *PacketBuf) {
		bodylen := ub.Len()
		datalen := bodylen + int(PacketHeadLen)
		udplen := algo.EncodeXORLen(datalen)
		if udplen <= mtu { // can be sent in a single packet
			pbs = []PacketBytes{
				pbuf.BufferItemToBytes((*PacketItem)(
					pb.copy().noFrag(nofrag).hasMore(false).offset(0),
				)),
			}
			return
		}
		if nofrag { // drop oversized packet
			return
		}
		pb.noFrag(false).hasMore(true)
		datalim := mtu - 9 - int(PacketHeadLen)
		n := bodylen / datalim
		r := bodylen % datalim
		if r > 0 {
			n++
		}
		pbs = make([]PacketBytes, n)
		for i := 0; i < n; i++ {
			a, b := i*datalim, (i+1)*datalim
			if b > bodylen {
				b = bodylen
			}
			pbs[i] = pbuf.BufferItemToBytes((*PacketItem)(
				pb.copy().offset(uint16(i*datalim)),
			)).Slice(a, b)
		}
	})
	return
}

func BuildPacketFromBytes(pb PacketBytes) pbuf.Bytes {
	w := bin.SelectWriter()
	pb.B(func(_ []byte, p *Packet) {
		w.P(func(b *pbuf.Buffer) {
			p.WriteHeaderTo(&b.Buffer)
		})
	})
	pb.V(func(b []byte) {
		w.Write(b)
	})
	return w.ToBytes()
}

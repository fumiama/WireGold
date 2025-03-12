package head

import (
	"encoding/binary"
	"errors"
	"sync/atomic"
	"unsafe"

	"github.com/sirupsen/logrus"

	"github.com/fumiama/WireGold/config"
	"github.com/fumiama/WireGold/internal/algo"
	"github.com/fumiama/WireGold/internal/bin"
	"github.com/fumiama/orbyte/pbuf"
)

func ParsePacketHeader(data []byte) (pbytes PacketBytes, err error) {
	if len(data) <= int(PacketHeadLen) {
		err = ErrDataLenLEHeader
		return
	}
	p := selectPacket()
	sz := 0
	p.P(func(pb *PacketBuf) {
		if bin.IsLittleEndian {
			copy((*[PacketHeadLen]byte)(
				(unsafe.Pointer)(&pb.DAT),
			)[:], data)
		} else {
			pb.DAT.idxdatsz = binary.LittleEndian.Uint32(data[:4])
			pb.DAT.randn = int32(binary.LittleEndian.Uint32(data[4:8]))
			pt := binary.LittleEndian.Uint16(data[8:10])
			pb.DAT.Proto = FlagsProto(pt)
			pb.DAT.TTL = uint8(pt >> 8)
			pb.DAT.SrcPort = binary.LittleEndian.Uint16(data[10:12])
			pb.DAT.DstPort = binary.LittleEndian.Uint16(data[12:14])
			pb.DAT.Offset = binary.LittleEndian.Uint16(data[14:16])
			copy(pb.DAT.src[:], data[16:20])
			copy(pb.DAT.dst[:], data[20:24])
			pb.DAT.md5h8rem = int64(binary.LittleEndian.Uint64(data[24:32]))
		}
		sz = pb.DAT.Size()
		if !pb.DAT.Proto.IsValid() {
			err = errors.New("invalid proto " + pb.DAT.Proto.String())
			return
		}
		if (!pb.DAT.Proto.HasMore() && (pb.DAT.Offset != 0 ||
			sz+int(PacketHeadLen) != len(data))) ||
			(pb.DAT.Proto.HasMore() && pb.DAT.Offset+
				uint16(len(data[PacketHeadLen:])) > uint16(sz)) {
			err = ErrInvalidOffset
			if config.ShowDebugLog {
				logrus.Warnf("[unbox] invalid offset %04x size %04x", pb.DAT.Offset, sz)
			}
			return
		}
		crc := algo.MD5Hash8(data[:PacketHeadNoCRCLen])
		if crc != uint64(pb.DAT.md5h8rem) {
			err = ErrBadCRCChecksum
			if config.ShowDebugLog {
				logrus.Warnf("[unbox] exp crc %016x but got %016x", pb.DAT.md5h8rem, crc)
			}
			return
		}
		if config.ShowDebugLog {
			logrus.Debugln("[unbox] header data len", sz, "read data len", len(data)-int(PacketHeadLen))
		}
		if sz+int(PacketHeadLen) == len(data) {
			pb.Buffer.Write(data[PacketHeadLen:])
			pb.DAT.md5h8rem = -1
			return
		}
		pb.Buffer.Grow(sz)
		pb.Buffer.Write(make([]byte, sz))
		pb.DAT.md5h8rem = int64(sz)
	})
	if err != nil {
		return
	}
	pbytes = pbuf.BufferItemToBytes(p)
	return
}

// WriteDataSegment 将 data 的数据并发解码到自身 buf.
//
// 必须先调用 ParsePacketHeader 获得 packet.
//
// return: complete.
func (p *Packet) WriteDataSegment(data, buf []byte) bool {
	if atomic.LoadInt64(&p.md5h8rem) <= 0 {
		return true
	}

	flags := FlagsProto(data[8])
	offset := binary.LittleEndian.Uint16(data[14:16])
	if config.ShowDebugLog {
		logrus.Debugln("[unbox] parse data flags", flags, "off", offset)
	}

	if offset == 0 {
		p.Proto = flags
		p.Offset = 0
		if config.ShowDebugLog {
			logrus.Debugln("[unbox] parse data set zero offset flags", flags)
		}
	}

	rembytes := atomic.LoadInt64(&p.md5h8rem)
	if rembytes > 0 {
		n := int64(copy(buf[offset:], data[PacketHeadLen:]))
		newrem := rembytes - n
		for !atomic.CompareAndSwapInt64(&p.md5h8rem, rembytes, newrem) {
			rembytes = atomic.LoadInt64(&p.md5h8rem)
			newrem = rembytes - n
		}
	}
	return atomic.LoadInt64(&p.md5h8rem) <= 0
}

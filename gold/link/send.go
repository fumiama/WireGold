package link

import (
	crand "crypto/rand"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"math/rand"

	"github.com/sirupsen/logrus"

	"github.com/fumiama/WireGold/config"
	"github.com/fumiama/WireGold/gold/head"
	"github.com/fumiama/WireGold/internal/bin"
	base14 "github.com/fumiama/go-base16384"
	"github.com/fumiama/orbyte/pbuf"
)

var (
	ErrDropBigDontFragPkt = errors.New("drop big don't fragmnet packet")
	ErrTTL                = errors.New("ttl exceeded")
)

func randseq(i uint16) uint32 {
	var buf [4]byte
	_, _ = crand.Read(buf[:2])
	binary.BigEndian.PutUint16(buf[2:4], i)
	return binary.BigEndian.Uint32(buf[:])
}

// WritePacket 基于 data 向 peer 发包
//
// data 可为空, 因为不保证可达所以不返回错误。
func (l *Link) WritePacket(proto uint8, data []byte, ttl uint8) {
	teatype := l.randkeyidx()
	sndcnt := uint16(l.incgetsndcnt())
	mtu := l.mtu
	if l.mturandomrange > 0 {
		mtu -= uint16(rand.Intn(int(l.mturandomrange)))
	}
	if config.ShowDebugLog {
		logrus.Debugln("[send] write mtu:", mtu, ", addt:", sndcnt&0x07ff, ", key index:", teatype, ", data len:", len(data))
	}
	pb := head.NewPacketBuilder().
		Src(l.me.me, l.me.srcport).Dst(l.peerip, l.me.dstport).
		Proto(proto).TTL(ttl).With(data)
	if l.usezstd {
		pb.Zstd()
	}
	pb = pb.Hash()
	var pktb *head.PacketBuilder
	if l.keys[0] == nil {
		pktb = pb.Plain(teatype, sndcnt&0x07ff)
	} else {
		pktb = pb.Seal(l.keys[teatype], teatype, sndcnt&0x07ff)
	}
	bs := pktb.Split(int(mtu), false)
	if config.ShowDebugLog {
		logrus.Debugln("[send] split packet into", len(bs), "parts")
	}
	for _, b := range bs { //TODO: impl. nofrag
		go l.write2peer(head.BuildPacketFromBytes(b), randseq(sndcnt))
	}
}

// write2peer 计算 xor + b14 后向 peer 发包
//
// 因为不保证可达所以不返回错误。
func (l *Link) write2peer(b pbuf.Bytes, seq uint32) {
	if l.doublepacket {
		err := l.write2peer1(b, seq)
		if err != nil {
			if config.ShowDebugLog {
				logrus.Warnln("[send] double wr2peer", l.peerip, "err:", err)
			}
		}
	}
	err := l.write2peer1(b, seq)
	if err != nil {
		if config.ShowDebugLog {
			logrus.Warnln("[send] wr2peer", l.peerip, "err:", err)
		}
	}
}

// write2peer1 计算 xor + b14 后向 peer 发一个包
func (l *Link) write2peer1(b pbuf.Bytes, seq uint32) (err error) {
	peerep := l.endpoint
	if bin.IsNilInterface(peerep) {
		return errors.New("nil endpoint of " + l.peerip.String())
	}

	conn := l.me.conn
	if conn == nil {
		return io.ErrClosedPipe
	}
	b.V(func(data []byte) {
		if config.ShowDebugLog {
			bound := 64
			endl := "..."
			if len(data) < bound {
				bound = len(data)
				endl = "."
			}
			logrus.Debugln("[send] crc seq", fmt.Sprintf("%08x", seq), "raw data bytes", hex.EncodeToString(data[:bound]), endl)
		}
		b = l.me.xorenc(data, seq)
		if config.ShowDebugLog {
			bound := 64
			endl := "..."
			if b.Len() < bound {
				bound = b.Len()
				endl = "."
			}
			b.V(func(b []byte) {
				logrus.Debugln("[send] crc seq", fmt.Sprintf("%08x", seq), "xored data bytes", hex.EncodeToString(b[:bound]), endl)
			})
		}
	})
	if l.me.base14 {
		b.V(func(data []byte) {
			b = pbuf.ParseBytes(base14.Encode(data)...)
			if config.ShowDebugLog {
				bound := 64
				endl := "..."
				if b.Len() < bound {
					bound = b.Len()
					endl = "."
				}
				b.V(func(b []byte) {
					logrus.Debugln("[send] crc seq", fmt.Sprintf("%08x", seq), "b14ed data bytes", hex.EncodeToString(b[:bound]), endl)
				})
			}
		})
	}
	b.V(func(b []byte) {
		if config.ShowDebugLog {
			logrus.Debugln("[send] crc seq", fmt.Sprintf("%08x", seq), "write", len(b), "bytes data from ep", conn.LocalAddr(), "to", peerep)
		}
		_, err = conn.WriteToPeer(b, peerep)
	})
	return
}

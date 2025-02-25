package link

import (
	crand "crypto/rand"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"runtime"

	"github.com/sirupsen/logrus"

	"github.com/fumiama/WireGold/config"
	"github.com/fumiama/WireGold/gold/head"
	"github.com/fumiama/WireGold/helper"
	base14 "github.com/fumiama/go-base16384"
	"github.com/fumiama/orbyte"
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

// WritePacket 向 peer 发包
func (l *Link) WritePacket(p *orbyte.Item[head.Packet], istransfer bool) (n int, err error) {
	pp := p.Pointer()
	teatype := l.randkeyidx()
	sndcnt := uint16(l.incgetsndcnt())
	seq := randseq(sndcnt)
	mtu := l.mtu
	if l.mturandomrange > 0 {
		mtu -= uint16(rand.Intn(int(l.mturandomrange)))
	}
	if config.ShowDebugLog {
		logrus.Debugln("[send] mtu:", mtu, ", addt:", sndcnt&0x07ff, ", key index:", teatype)
	}
	if !istransfer {
		l.encrypt(pp, sndcnt, teatype)
	}
	delta := (int(mtu) - head.PacketHeadLen) & 0x0000fff8
	if delta <= 0 {
		logrus.Warnln("[send] reset invalid data frag len", delta, "to 8")
		delta = 8
	}
	remlen := pp.BodyLen()
	if remlen <= delta {
		return l.write(p, teatype, sndcnt, uint32(remlen), 0, istransfer, false, seq)
	}
	if istransfer && pp.Flags.DontFrag() && remlen > delta {
		return 0, ErrDropBigDontFragPkt
	}
	ttl := pp.TTL
	totl := uint32(remlen)
	pos := 0
	packet := head.ParsePacket(pp.ShallowCopy())
	for remlen > delta {
		remlen -= delta
		if config.ShowDebugLog {
			logrus.Debugln("[send] split frag [", pos, "~", pos+delta, "], remain:", remlen)
		}
		packet.Pointer().CropBody(pos, pos+delta)
		cnt, err := l.write(packet, teatype, sndcnt, totl, uint16(pos>>3), istransfer, true, seq)
		n += cnt
		if err != nil {
			return n, err
		}
		packet.Pointer().TTL = ttl
		pos += delta
	}
	if remlen > 0 {
		if config.ShowDebugLog {
			logrus.Debugln("[send] last frag [", pos, "~", pos+remlen, "]")
		}
		pp.CropBody(pos, pos+remlen)
		cnt := 0
		cnt, err = l.write(p, teatype, sndcnt, totl, uint16(pos>>3), istransfer, false, seq)
		n += cnt
	}
	runtime.KeepAlive(p)
	return n, err
}

func (l *Link) encrypt(p *head.Packet, sndcnt uint16, teatype uint8) {
	p.FillHash()
	if config.ShowDebugLog {
		logrus.Debugln("[send] data len before encrypt:", p.BodyLen())
	}
	data := p.Body()
	if l.usezstd {
		data = encodezstd(data).Trans().Bytes()
		if config.ShowDebugLog {
			logrus.Debugln("[send] data len after zstd:", len(data))
		}
	}
	p.SetBody(l.encode(teatype, sndcnt&0x07ff, data).Trans().Bytes())
	if config.ShowDebugLog {
		logrus.Debugln("[send] data len after xchacha20:", p.BodyLen(), "addt:", sndcnt)
	}
}

// write 向 peer 发包
func (l *Link) write(
	p *orbyte.Item[head.Packet], teatype uint8, additional uint16,
	datasz uint32, offset uint16, istransfer,
	hasmore bool, seq uint32,
) (int, error) {
	if p.Pointer().DecreaseAndGetTTL() <= 0 {
		return 0, ErrTTL
	}
	if l.doublepacket {
		_, _ = l.writeonce(p, teatype, additional, datasz, offset, istransfer, hasmore, seq)
	}
	return l.writeonce(p, teatype, additional, datasz, offset, istransfer, hasmore, seq)
}

// write 向 peer 发一个包
func (l *Link) writeonce(
	p *orbyte.Item[head.Packet], teatype uint8, additional uint16,
	datasz uint32, offset uint16,
	istransfer, hasmore bool, seq uint32,
) (int, error) {
	peerep := l.endpoint
	if helper.IsNilInterface(peerep) {
		return 0, errors.New("nil endpoint of " + p.Pointer().Dst.String())
	}

	var d pbuf.Bytes
	// TODO: now all packet allow frag, adapt to DF
	if istransfer {
		d = p.Pointer().MarshalWith(nil, 0, 0, 0, offset, false, hasmore)
	} else {
		d = p.Pointer().MarshalWith(l.me.me, teatype, additional, datasz, offset, false, hasmore)
	}

	bound := 64
	endl := "..."
	if d.Len() < bound {
		bound = d.Len()
		endl = "."
	}
	conn := l.me.conn
	if conn == nil {
		return 0, io.ErrClosedPipe
	}
	if config.ShowDebugLog {
		logrus.Debugln("[send] write", d.Len(), "bytes data from ep", conn.LocalAddr(), "to", peerep, "offset", fmt.Sprintf("%04x", offset), "crc", fmt.Sprintf("%016x", p.Pointer().CRC64()))
		logrus.Debugln("[send] data bytes", hex.EncodeToString(d.Bytes()[:bound]), endl)
	}
	d = l.me.xorenc(d.Bytes(), seq)
	if l.me.base14 {
		d = pbuf.ParseBytes(base14.Encode(d.Bytes())...)
	}
	if config.ShowDebugLog {
		logrus.Debugln("[send] data xored", hex.EncodeToString(d.Bytes()[:bound]), endl)
	}
	return conn.WriteToPeer(d.Trans().Bytes(), peerep)
}

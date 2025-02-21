package link

import (
	"bytes"
	crand "crypto/rand"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"math/rand"

	"github.com/klauspost/compress/zstd"
	"github.com/sirupsen/logrus"

	"github.com/fumiama/WireGold/config"
	"github.com/fumiama/WireGold/gold/head"
	"github.com/fumiama/WireGold/helper"
	base14 "github.com/fumiama/go-base16384"
)

var (
	ErrDropBigDontFragPkt = errors.New("drop big don't fragmnet packet")
	ErrTTL                = errors.New("ttl exceeded")
)

// WriteAndPut 向 peer 发包并将包放回缓存池
func (l *Link) WriteAndPut(p *head.Packet, istransfer bool) (n int, err error) {
	defer p.Put()
	teatype := l.randkeyidx()
	sndcnt := uint16(l.incgetsndcnt())
	var buf [4]byte
	_, _ = crand.Read(buf[:2])
	binary.BigEndian.PutUint16(buf[2:4], sndcnt)
	seq := binary.BigEndian.Uint32(buf[:])
	mtu := l.mtu
	if l.mturandomrange > 0 {
		mtu -= uint16(rand.Intn(int(l.mturandomrange)))
	}
	if config.ShowDebugLog {
		logrus.Debugln("[send] mtu:", mtu, ", addt:", sndcnt&0x07ff, ", key index:", teatype)
	}
	if !istransfer {
		l.encrypt(p, sndcnt, teatype)
	}
	delta := (int(mtu) - head.PacketHeadLen) & 0x0000fff8
	if delta <= 0 {
		logrus.Warnln("[send] reset invalid data frag len", delta, "to 8")
		delta = 8
	}
	remlen := p.BodyLen()
	if remlen <= delta {
		return l.write(p, teatype, sndcnt, uint32(remlen), 0, istransfer, false, seq)
	}
	if istransfer && p.Flags.DontFrag() && remlen > delta {
		return 0, ErrDropBigDontFragPkt
	}
	ttl := p.TTL
	totl := uint32(remlen)
	pos := 0
	packet := p.Copy()
	for remlen > delta {
		remlen -= delta
		if config.ShowDebugLog {
			logrus.Debugln("[send] split frag [", pos, "~", pos+delta, "], remain:", remlen)
		}
		packet.CropBody(pos, pos+delta)
		cnt, err := l.write(packet, teatype, sndcnt, totl, uint16(pos>>3), istransfer, true, seq)
		n += cnt
		if err != nil {
			return n, err
		}
		packet.TTL = ttl
		pos += delta
	}
	packet.Put()
	if remlen > 0 {
		if config.ShowDebugLog {
			logrus.Debugln("[send] last frag [", pos, "~", pos+remlen, "]")
		}
		p.CropBody(pos, pos+remlen)
		cnt := 0
		cnt, err = l.write(p, teatype, sndcnt, totl, uint16(pos>>3), istransfer, false, seq)
		n += cnt
	}
	return n, err
}

func (l *Link) encrypt(p *head.Packet, sndcnt uint16, teatype uint8) {
	p.FillHash()
	if config.ShowDebugLog {
		logrus.Debugln("[send] data len before encrypt:", p.BodyLen())
	}
	data := p.Body()
	if l.usezstd {
		w := helper.SelectWriter()
		defer helper.PutWriter(w)
		enc, _ := zstd.NewWriter(w, zstd.WithEncoderLevel(zstd.SpeedFastest))
		_, _ = io.Copy(enc, bytes.NewReader(data))
		enc.Close()
		data = w.Bytes()
		if config.ShowDebugLog {
			logrus.Debugln("[send] data len after zstd:", len(data))
		}
	}
	p.SetBody(l.Encode(teatype, sndcnt&0x07ff, data), true)
	if config.ShowDebugLog {
		logrus.Debugln("[send] data len after xchacha20:", p.BodyLen(), "addt:", sndcnt)
	}
}

// write 向 peer 发包
func (l *Link) write(p *head.Packet, teatype uint8, additional uint16, datasz uint32, offset uint16, istransfer, hasmore bool, seq uint32) (int, error) {
	if p.DecreaseAndGetTTL() <= 0 {
		return 0, ErrTTL
	}
	if l.doublepacket {
		_, _ = l.writeonce(p, teatype, additional, datasz, offset, istransfer, hasmore, seq)
	}
	return l.writeonce(p, teatype, additional, datasz, offset, istransfer, hasmore, seq)
}

// write 向 peer 发一个包
func (l *Link) writeonce(p *head.Packet, teatype uint8, additional uint16, datasz uint32, offset uint16, istransfer, hasmore bool, seq uint32) (int, error) {
	peerep := l.endpoint
	if peerep == nil {
		return 0, errors.New("nil endpoint of " + p.Dst.String())
	}

	var d []byte
	var cl func()
	// TODO: now all packet allow frag, adapt to DF
	if istransfer {
		d, cl = p.Marshal(nil, 0, 0, 0, offset, false, hasmore)
	} else {
		d, cl = p.Marshal(l.me.me, teatype, additional, datasz, offset, false, hasmore)
	}
	defer cl()

	bound := 64
	endl := "..."
	if len(d) < bound {
		bound = len(d)
		endl = "."
	}
	conn := l.me.conn
	if conn == nil {
		return 0, io.ErrClosedPipe
	}
	if config.ShowDebugLog {

		logrus.Debugln("[send] write", len(d), "bytes data from ep", conn.LocalAddr(), "to", peerep, "offset", fmt.Sprintf("%04x", offset), "crc", fmt.Sprintf("%016x", p.CRC64()))
		logrus.Debugln("[send] data bytes", hex.EncodeToString(d[:bound]), endl)
	}
	d = l.me.xorenc(d, seq)
	if l.me.base14 {
		d = base14.Encode(d)
	}
	if config.ShowDebugLog {
		logrus.Debugln("[send] data xored", hex.EncodeToString(d[:bound]), endl)
	}
	defer helper.PutBytes(d)
	return conn.WriteToPeer(d, peerep)
}

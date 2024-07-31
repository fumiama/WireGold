package link

import (
	"bytes"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"time"

	"github.com/fumiama/WireGold/gold/head"
	"github.com/fumiama/WireGold/helper"
	"github.com/klauspost/compress/zstd"
	"github.com/sirupsen/logrus"
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
	mtu := l.mtu
	if l.mturandomrange > 0 {
		mtu -= uint16(rand.Intn(int(l.mturandomrange)))
	}
	logrus.Debugln("[send] mtu:", mtu, ", addt:", sndcnt&0x07ff, ", key index:", teatype)
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
		return l.write(p, teatype, sndcnt, uint32(remlen), 0, istransfer, false)
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
		logrus.Debugln("[send] split frag [", pos, "~", pos+delta, "], remain:", remlen)
		packet.CropBody(pos, pos+delta)
		cnt, err := l.write(packet, teatype, sndcnt, totl, uint16(pos>>3), istransfer, true)
		n += cnt
		if err != nil {
			return n, err
		}
		packet.TTL = ttl
		pos += delta
	}
	packet.Put()
	if remlen > 0 {
		logrus.Debugln("[send] last frag [", pos, "~", pos+remlen, "]")
		p.CropBody(pos, pos+remlen)
		cnt := 0
		cnt, err = l.write(p, teatype, sndcnt, totl, uint16(pos>>3), istransfer, false)
		n += cnt
	}
	return n, err
}

func (l *Link) encrypt(p *head.Packet, sndcnt uint16, teatype uint8) {
	p.FillHash()
	logrus.Debugln("[send] data len before encrypt:", p.BodyLen())
	data := p.Body()
	if l.usezstd {
		w := helper.SelectWriter()
		defer helper.PutWriter(w)
		enc, _ := zstd.NewWriter(w, zstd.WithEncoderLevel(zstd.SpeedFastest))
		_, _ = io.Copy(enc, bytes.NewReader(data))
		enc.Close()
		data = w.Bytes()
		logrus.Debugln("[send] data len after zstd:", len(data))
	}
	p.SetBody(l.Encode(teatype, sndcnt&0x07ff, data), true)
	logrus.Debugln("[send] data len after xchacha20:", p.BodyLen(), "addt:", sndcnt)
}

// write 向 peer 发包
func (l *Link) write(p *head.Packet, teatype uint8, additional uint16, datasz uint32, offset uint16, istransfer, hasmore bool) (int, error) {
	if p.DecreaseAndGetTTL() <= 0 {
		return 0, ErrTTL
	}
	if l.doublepacket {
		cpp := p.Copy()
		_ = time.AfterFunc(time.Millisecond*(100+time.Duration(rand.Intn(50))), func() {
			defer cpp.Put()
			_, _ = l.writeonce(cpp, teatype, additional, datasz, offset, istransfer, hasmore)
		})
	}
	return l.writeonce(p, teatype, additional, datasz, offset, istransfer, hasmore)
}

// write 向 peer 发一个包
func (l *Link) writeonce(p *head.Packet, teatype uint8, additional uint16, datasz uint32, offset uint16, istransfer, hasmore bool) (int, error) {
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
	logrus.Debugln("[send] write", len(d), "bytes data from ep", l.me.conn.LocalAddr(), "to", peerep, "offset:", fmt.Sprintf("%04x", offset))
	logrus.Debugln("[send] data bytes", hex.EncodeToString(d[:bound]), endl)
	d = l.me.xorenc(d)
	logrus.Debugln("[send] data xored", hex.EncodeToString(d[:bound]), endl)
	defer helper.PutBytes(d)
	return l.me.conn.WriteToPeer(d, peerep)
}

package link

import (
	"bytes"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"math/rand"

	"github.com/fumiama/WireGold/gold/head"
	"github.com/fumiama/WireGold/helper"
	"github.com/klauspost/compress/zstd"
	"github.com/sirupsen/logrus"
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
	delta := (int(mtu) - 60) & 0x0000fff8
	if delta <= 0 {
		logrus.Warnln("[send] reset invalid data frag len", delta, "to 8")
		delta = 8
	}
	if len(p.Data) <= delta {
		return l.write(p, teatype, sndcnt, uint32(len(p.Data)), 0, istransfer, false)
	}
	if istransfer && p.Flags.DontFrag() && len(p.Data) > delta {
		return 0, errors.New("drop don't fragmnet big trans packet")
	}
	data := p.Data
	ttl := p.TTL
	totl := uint32(len(data))
	pos := 0
	packet := head.SelectPacket()
	*packet = *p
	for ; int(totl)-pos > delta; pos += delta {
		logrus.Debugln("[send] split frag [", pos, "~", pos+delta, "], remain:", int(totl)-pos-delta)
		packet.Data = data[:delta]
		cnt, err := l.write(packet, teatype, sndcnt, totl, uint16(pos>>3), istransfer, true)
		n += cnt
		if err != nil {
			return n, err
		}
		data = data[delta:]
		packet.TTL = ttl
	}
	packet.Put()
	if len(data) > 0 {
		p.Data = data
		cnt := 0
		cnt, err = l.write(p, teatype, sndcnt, totl, uint16(pos>>3), istransfer, false)
		n += cnt
	}
	return n, err
}

func (l *Link) encrypt(p *head.Packet, sndcnt uint16, teatype uint8) {
	p.FillHash()
	logrus.Debugln("[send] data len before encrypt:", len(p.Data))
	if l.usezstd {
		w := helper.SelectWriter()
		defer helper.PutWriter(w)
		enc, _ := zstd.NewWriter(w, zstd.WithEncoderLevel(zstd.SpeedFastest))
		_, _ = io.Copy(enc, bytes.NewReader(p.Data))
		enc.Close()
		p.Data = w.Bytes()
		logrus.Debugln("[send] data len after zstd:", len(p.Data))
	}
	p.Data = l.Encode(teatype, sndcnt&0x07ff, p.Data)
	logrus.Debugln("[send] data len after xchacha20:", len(p.Data), "addt:", sndcnt)
}

// write 向 peer 发一个包
func (l *Link) write(p *head.Packet, teatype uint8, additional uint16, datasz uint32, offset uint16, istransfer, hasmore bool) (n int, err error) {
	var d []byte
	var cl func()
	if istransfer {
		d, cl = p.Marshal(nil, teatype, additional, 0, 0, false, false)
	} else {
		d, cl = p.Marshal(l.me.me, teatype, additional, datasz, offset, false, hasmore)
	}
	if d == nil {
		return 0, errors.New("[send] ttl exceeded")
	}
	peerep := l.endpoint
	if peerep == nil {
		return 0, errors.New("[send] nil endpoint of " + p.Dst.String())
	}
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
	n, err = l.me.conn.WriteToPeer(d, peerep)
	cl()
	return
}

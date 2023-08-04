package link

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"sync/atomic"

	"github.com/fumiama/WireGold/gold/head"
	"github.com/fumiama/WireGold/helper"
	"github.com/klauspost/compress/zstd"
	"github.com/sirupsen/logrus"
)

// WriteAndPut 向 peer 发包并将包放回缓存池
func (l *Link) WriteAndPut(p *head.Packet, istransfer bool) (n int, err error) {
	teatype := uint8(rand.Intn(16))
	sndcnt := atomic.AddUintptr(&l.sendcount, 1)
	mtu := l.mtu
	if l.mturandomrange > 0 {
		mtu -= uint16(rand.Intn(int(l.mturandomrange))) & 0xfff8
	}
	logrus.Debugln("[send] mtu:", mtu, ", count:", sndcnt, ", additional data:", uint16(sndcnt))
	if len(p.Data) <= int(mtu) {
		if !istransfer {
			l.encrypt(p, uint16(sndcnt), teatype)
		}
		defer p.Put()
		return l.write(p, teatype, uint16(sndcnt), mtu, uint32(len(p.Data)), 0, istransfer, false)
	}
	if !istransfer {
		l.encrypt(p, uint16(sndcnt), teatype)
	}
	data := p.Data
	ttl := p.TTL
	totl := uint32(len(data))
	i := 0
	packet := head.SelectPacket()
	*packet = *p
	for ; int(totl)-i > int(mtu); i += int(mtu) {
		logrus.Debugln("[send] split frag [", i, "~", i+int(mtu), "], remain:", int(totl)-i-int(mtu))
		packet.Data = data[:int(mtu)]
		cnt, err := l.write(packet, teatype, uint16(sndcnt), mtu, totl, uint16(i>>3), istransfer, true)
		n += cnt
		if err != nil {
			return n, err
		}
		data = data[int(mtu):]
		packet.TTL = ttl
	}
	packet.Put()
	p.Data = data
	cnt, err := l.write(p, teatype, uint16(sndcnt), mtu, totl, uint16(i>>3), istransfer, false)
	p.Put()
	n += cnt
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
	if l.aead != nil {
		p.Data = l.EncodePreshared(sndcnt, p.Data)
		logrus.Debugln("[send] data len after xchacha20:", len(p.Data))
	}
	p.Data = l.Encode(teatype, p.Data)
	logrus.Debugln("[send] data len after tea:", len(p.Data))
}

// write 向 peer 发一个包
func (l *Link) write(p *head.Packet, teatype uint8, additional, mtu uint16, datasz uint32, offset uint16, istransfer, hasmore bool) (n int, err error) {
	var d []byte
	var cl func()
	if istransfer {
		if p.Flags&0x4000 == 0x4000 && len(p.Data) > int(mtu) {
			return len(p.Data), errors.New("drop dont fragmnet big trans packet")
		}
		d, cl = p.Marshal(nil, teatype, additional, 0, 0, false, false)
	} else {
		d, cl = p.Marshal(l.me.me, teatype, additional, datasz, offset, false, hasmore)
	}
	if d == nil {
		return 0, errors.New("[send] ttl exceeded")
	}
	if err == nil {
		peerep := l.endpoint
		if peerep == nil {
			return 0, errors.New("[send] nil endpoint of " + p.Dst.String())
		}
		logrus.Debugln("[send] write", len(d), "bytes data from ep", l.me.myep.LocalAddr(), "to", peerep, "offset:", fmt.Sprintf("%04x", offset))
		n, err = l.me.myep.WriteToUDP(l.me.xor(d), peerep)
		cl()
	}
	return
}

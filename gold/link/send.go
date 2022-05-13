package link

import (
	"errors"
	"fmt"
	"math/rand"

	"github.com/fumiama/WireGold/gold/head"
	"github.com/sirupsen/logrus"
)

// WriteAndPut 向 peer 发包并将包放回缓存池
func (l *Link) WriteAndPut(p *head.Packet, istransfer bool) (n int, err error) {
	teatype := uint8(rand.Intn(16))
	if len(p.Data) <= int(l.mtu) {
		if !istransfer {
			p.FillHash()
			p.Data = l.Encode(teatype, p.Data)
		}
		defer p.Put()
		return l.write(p, teatype, uint32(len(p.Data)), 0, istransfer, false)
	}
	if !istransfer {
		p.FillHash()
		p.Data = l.Encode(teatype, p.Data)
	}
	data := p.Data
	ttl := p.TTL
	totl := uint32(len(data))
	i := 0
	packet := head.SelectPacket()
	*packet = *p
	for ; int(totl)-i > int(l.mtu); i += int(l.mtu) {
		logrus.Debugln("[link] split frag", i, ":", i+int(l.mtu), ", remain:", int(totl)-i-int(l.mtu))
		packet.Data = data[:int(l.mtu)]
		cnt, err := l.write(packet, teatype, totl, uint16(i>>3), istransfer, true)
		n += cnt
		if err != nil {
			return n, err
		}
		data = data[int(l.mtu):]
		packet.TTL = ttl
	}
	packet.Put()
	p.Data = data
	cnt, err := l.write(p, teatype, totl, uint16(i>>3), istransfer, false)
	p.Put()
	n += cnt
	return n, err
}

// write 向 peer 发一个包
func (l *Link) write(p *head.Packet, teatype uint8, datasz uint32, offset uint16, istransfer, hasmore bool) (n int, err error) {
	var d []byte
	var cl func()
	if istransfer {
		if p.Flags&0x4000 == 0x4000 && len(p.Data) > int(l.mtu) {
			return len(p.Data), errors.New("drop dont fragmnet big trans packet")
		}
		d, cl = p.Marshal(nil, teatype, 0, 0, false, false)
	} else {
		d, cl = p.Marshal(l.me.me, teatype, datasz, offset, false, hasmore)
	}
	if d == nil {
		return 0, errors.New("[link] ttl exceeded")
	}
	if err == nil {
		peerep := l.endpoint
		if peerep == nil {
			return 0, errors.New("[link] nil endpoint of " + p.Dst.String())
		}
		logrus.Debugln("[link] write", len(d), "bytes data from ep", l.me.myep.LocalAddr(), "to", peerep, "offset:", fmt.Sprintf("%04x", offset))
		n, err = l.me.myep.WriteToUDP(d, peerep)
		cl()
	}
	return
}

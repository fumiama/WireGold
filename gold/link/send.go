package link

import (
	"errors"
	"fmt"

	"github.com/fumiama/WireGold/gold/head"
	"github.com/sirupsen/logrus"
)

// Write 向 peer 发包
func (l *Link) Write(p *head.Packet, istransfer bool) (n int, err error) {
	if len(p.Data) <= int(l.me.mtu) {
		if !istransfer {
			p.FillHash()
			p.Data = l.Encode(p.Data)
		}
		return l.write(p, uint32(len(p.Data)), 0, istransfer, false)
	}
	if !istransfer {
		p.FillHash()
		p.Data = l.Encode(p.Data)
	}
	data := p.Data
	totl := uint32(len(data))
	i := 0
	for ; int(totl)-i > int(l.me.mtu); i += int(l.me.mtu) {
		logrus.Debugln("[link] split frag", i, ":", i+int(l.me.mtu), ", remain:", int(totl)-i-int(l.me.mtu))
		packet := *p
		packet.Data = data[:int(l.me.mtu)]
		cnt, err := l.write(&packet, totl, uint16(uint(i)>>3), istransfer, true)
		n += cnt
		if err != nil {
			return n, err
		}
		data = data[int(l.me.mtu):]
	}
	p.Data = data
	cnt, err := l.write(p, totl, uint16(uint(i)>>3), istransfer, false)
	n += cnt
	if err != nil {
		return n, err
	}
	return n, nil
}

// write 向 peer 发一个包
func (l *Link) write(p *head.Packet, datasz uint32, offset uint16, istransfer, hasmore bool) (n int, err error) {
	var d []byte
	var cl func()
	if istransfer {
		if p.Flags&0x4000 == 0x4000 && len(p.Data) > int(l.me.mtu) {
			return len(p.Data), errors.New("drop dont fragmnet big trans packet")
		}
		d, cl = p.Marshal(nil, 0, 0, false, false)
	} else {
		d, cl = p.Marshal(l.me.me, datasz, offset, false, hasmore)
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

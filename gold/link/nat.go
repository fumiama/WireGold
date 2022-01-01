package link

import (
	"time"

	"github.com/sirupsen/logrus"

	"github.com/fumiama/WireGold/gold/head"
)

// 保持 NAT
func (l *Link) keepAlive() {
	if l.keepalive > 0 {
		logrus.Infoln("[link.nat] start to keep alive")
		t := time.NewTicker(time.Second * time.Duration(l.keepalive))
		for range t.C {
			n, err := l.Write(head.NewPacket(head.ProtoHello, l.me.srcport, l.peerip, l.me.dstport, nil), false)
			if err == nil {
				logrus.Infoln("[link] send", n, "bytes keep alive packet")
			} else {
				logrus.Errorln("[link] send keep alive packet error:", err)
			}
		}
	}
}

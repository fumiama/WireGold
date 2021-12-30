package link

import (
	"time"

	"github.com/sirupsen/logrus"

	"github.com/fumiama/WireGold/gold/head"
)

// 保持 NAT
func (l *Link) keepAlive() {
	if l.keepalive > 0 && !l.haskeepruning {
		l.haskeepruning = true
		go func() {
			t := time.NewTicker(time.Second * time.Duration(l.keepalive))
			for range t.C {
				_, _ = l.Write(head.NewPacket(head.ProtoHello, 0, l.peerip, 0, nil), false)
				logrus.Infoln("[link.nat] send keep alive packet")
			}
		}()
		logrus.Infoln("[link.nat] start to keep alive")
	}
}

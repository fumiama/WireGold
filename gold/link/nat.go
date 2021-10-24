package link

import (
	"time"

	"github.com/sirupsen/logrus"

	"github.com/fumiama/WireGold/gold/head"
)

func (l *Link) keepAlive() {
	if l.KeepAlive > 0 && !l.hasKeepRuning {
		l.hasKeepRuning = true
		go func() {
			t := time.NewTicker(time.Second * time.Duration(l.KeepAlive))
			for range t.C {
				_, _ = l.Write(head.NewPacket(head.ProtoHello, 0, 0, nil))
				logrus.Infoln("[link.nat] send keep alive packet")
			}
		}()
		logrus.Infoln("[link.nat] start to keep alive")
	}
}

func onQuery(packet *head.Packet) {
	// TODO: 完成data解包与notify分发
}

func onNotify(packet *head.Packet) {
	// TODO: 完成data解包与endpoint注册
}

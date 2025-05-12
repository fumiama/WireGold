package hello

import (
	"github.com/sirupsen/logrus"

	"github.com/fumiama/WireGold/gold/head"
	"github.com/fumiama/WireGold/gold/link"
	"github.com/fumiama/WireGold/internal/file"
)

func init() {
	link.RegisterDispacher(head.ProtoHello, func(_ *head.Packet, peer *link.Link, data []byte) {
		switch {
		case len(data) == 0:
			logrus.Warnln(file.Header(), "recv old packet, do nothing")
		case data[0] == byte(head.HelloPing):
			go peer.WritePacket(head.ProtoHello, []byte{byte(head.HelloPong)}, peer.Me().TTL())
			logrus.Infoln(file.Header(), "recv, send ack")
		default:
			logrus.Infoln(file.Header(), "recv ack, do nothing")
		}
	})
}

package proto

import (
	"github.com/fumiama/orbyte/pbuf"
	"github.com/sirupsen/logrus"

	"github.com/fumiama/WireGold/gold/head"
	"github.com/fumiama/WireGold/gold/link"
	"github.com/fumiama/WireGold/internal/file"
)

func init() {
	link.AddProto(head.ProtoHello, func(_ *head.Packet, peer *link.Link, data pbuf.Bytes) {
		onHello(data, peer)
	})
}

func onHello(data pbuf.Bytes, p *link.Link) {
	data.V(func(b []byte) {
		switch {
		case len(b) == 0:
			logrus.Warnln(file.Header(), "recv old packet, do nothing")
		case b[0] == byte(head.HelloPing):
			go p.WritePacket(head.ProtoHello, []byte{byte(head.HelloPong)})
			logrus.Infoln(file.Header(), "recv, send ack")
		default:
			logrus.Infoln(file.Header(), "recv ack, do nothing")
		}
	})
}

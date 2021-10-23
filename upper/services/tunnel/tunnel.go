package tunnel

import (
	"github.com/sirupsen/logrus"

	"github.com/fumiama/WireGold/gold/head"
	"github.com/fumiama/WireGold/gold/link"
)

type Tunnel struct {
	l    *link.Link
	In   *chan []byte
	Out  *chan []byte
	src  uint16
	dest uint16
}

func Create(peer string, srcport uint16, destport uint16) (s Tunnel, err error) {
	logrus.Infoln("[tunnel] create from", srcport, "to", destport)
	var l link.Link
	l, err = link.Connect(peer)
	if err == nil {
		s.l = &l
		s.In = new(chan []byte)
		s.Out = new(chan []byte)
		s.src = srcport
		s.dest = destport
		go s.handleWrite()
	} else {
		logrus.Errorln("[tunnel] create err:", err)
	}
	return
}

func (s *Tunnel) handleWrite() {
	for b := range *s.In {
		_, err := s.l.Write(head.NewPacket(head.ProtoData, s.src, s.dest, b))
		if err != nil {
			logrus.Errorln("[tunnel] write err:", err)
		}
	}
}

func (s *Tunnel) Handle(srcport uint16, destport uint16, data *[]byte) {

}

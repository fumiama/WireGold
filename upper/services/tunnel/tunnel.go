package tunnel

import (
	"errors"
	"net"

	"github.com/sirupsen/logrus"

	"github.com/fumiama/WireGold/gold/head"
	"github.com/fumiama/WireGold/gold/link"
)

type Tunnel struct {
	l        *link.Link
	in       chan []byte
	out      chan []byte
	outcache []byte
	peerip   net.IP
	src      uint16
	dest     uint16
	mtu      uint16
}

func Create(me *link.Me, peer string, srcport, destport, mtu uint16) (s Tunnel, err error) {
	logrus.Infoln("[tunnel] create from", srcport, "to", destport)
	s.l, err = me.Connect(peer)
	if err == nil {
		s.in = make(chan []byte, 4)
		s.out = make(chan []byte, 4)
		s.peerip = net.ParseIP(peer)
		s.src = srcport
		s.dest = destport
		s.mtu = mtu
		go s.handleWrite()
		go s.handleRead()
	} else {
		logrus.Errorln("[tunnel] create err:", err)
	}
	return
}

func (s *Tunnel) Write(p []byte) (int, error) {
	s.in <- p
	return len(p), nil
}

func (s *Tunnel) Read(p []byte) (int, error) {
	var d []byte
	if s.outcache != nil {
		d = s.outcache
	} else {
		d = <-s.out
	}
	if d != nil {
		if len(p) >= len(d) {
			s.outcache = nil
			return copy(p, d), nil
		} else {
			s.outcache = d[len(p):]
			return copy(p, d[:len(p)]), nil
		}
	}
	return 0, errors.New("reading reaches nil")
}

func (s *Tunnel) Close() error {
	s.l.Close()
	close(s.in)
	return nil
}

func (s *Tunnel) handleWrite() {
	for b := range s.in {
		logrus.Debugln("[tunnel] write recv", b)
		if b == nil {
			logrus.Errorln("[tunnel] write recv nil")
			break
		}
		logrus.Debugln("[tunnel] writing", len(b), "bytes...")
		for len(b) > int(s.mtu) {
			logrus.Infoln("[tunnel] split buffer")
			_, err := s.l.Write(head.NewPacket(head.ProtoData, s.src, s.peerip, s.dest, b[:s.mtu]), false)
			if err != nil {
				logrus.Errorln("[tunnel] write err:", err)
				return
			}
			logrus.Debugln("[tunnel] write succeeded")
			b = b[s.mtu:]
		}
		_, err := s.l.Write(head.NewPacket(head.ProtoData, s.src, s.peerip, s.dest, b), false)
		if err != nil {
			logrus.Errorln("[tunnel] write err:", err)
			break
		}
		logrus.Debugln("[tunnel] write succeeded")
	}
}

func (s *Tunnel) handleRead() {
	for {
		p := s.l.Read()
		if p == nil {
			logrus.Errorln("[tunnel] read recv nil")
			break
		}
		logrus.Debugln("[tunnel] read recv", p.Data)
		s.out <- p.Data
	}
}

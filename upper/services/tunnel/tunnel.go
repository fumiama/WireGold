package tunnel

import (
	"encoding/hex"
	"io"
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

func Create(me *link.Me, peer string) (s Tunnel, err error) {
	s.l, err = me.Connect(peer)
	if err == nil {
		s.in = make(chan []byte, 4)
		s.out = make(chan []byte, 4)
		s.peerip = net.ParseIP(peer)
	} else {
		logrus.Errorln("[tunnel] create err:", err)
	}
	return
}

func (s *Tunnel) Start(srcport, destport, mtu uint16) {
	logrus.Infoln("[tunnel] start from", srcport, "to", destport)
	s.src = srcport
	s.dest = destport
	s.mtu = mtu
	go s.handleWrite()
	go s.handleRead()
}

func (s *Tunnel) Run(srcport, destport, mtu uint16) {
	logrus.Infoln("[tunnel] start from", srcport, "to", destport)
	s.src = srcport
	s.dest = destport
	s.mtu = mtu
	go s.handleWrite()
	s.handleRead()
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
	return 0, io.EOF
}

func (s *Tunnel) Stop() {
	s.l.Close()
	close(s.in)
}

func (s *Tunnel) handleWrite() {
	for b := range s.in {
		end := 64
		endl := "..."
		if len(b) < 64 {
			end = len(b)
			endl = "."
		}
		logrus.Debugln("[tunnel] write send", hex.EncodeToString(b[:end]), endl)
		if b == nil {
			logrus.Errorln("[tunnel] write recv nil")
			break
		}
		logrus.Debugln("[tunnel] writing", len(b), "bytes...")
		for len(b) > int(s.mtu) {
			logrus.Infoln("[tunnel] split buffer")
			_, err := s.l.WriteAndPut(head.NewPacket(head.ProtoData, s.src, s.peerip, s.dest, b[:s.mtu]), false)
			if err != nil {
				logrus.Errorln("[tunnel] write err:", err)
				return
			}
			logrus.Debugln("[tunnel] write succeeded")
			b = b[s.mtu:]
		}
		_, err := s.l.WriteAndPut(head.NewPacket(head.ProtoData, s.src, s.peerip, s.dest, b), false)
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
		end := 64
		endl := "..."
		if len(p.Data) < 64 {
			end = len(p.Data)
			endl = "."
		}
		logrus.Debugln("[tunnel] read recv", hex.EncodeToString(p.Data[:end]), endl)
		s.out <- p.Data
		p.Put()
	}
}

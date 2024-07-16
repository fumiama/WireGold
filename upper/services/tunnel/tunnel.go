package tunnel

import (
	"encoding/binary"
	"encoding/hex"
	"io"
	"net"

	"github.com/sirupsen/logrus"

	_ "github.com/fumiama/WireGold/gold/p2p/tcp" // support tcp connection
	_ "github.com/fumiama/WireGold/gold/p2p/udp" // support udp connection

	"github.com/fumiama/WireGold/gold/head"
	"github.com/fumiama/WireGold/gold/link"
)

type Tunnel struct {
	l        *link.Link
	in       chan []byte
	out      chan *head.Packet
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
		s.out = make(chan *head.Packet, 4)
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
		pkt := <-s.out
		if pkt == nil {
			return 0, io.EOF
		}
		defer pkt.Put()
		if pkt.BodyLen() < 4 {
			logrus.Warnln("[tunnel] unexpected packet data len", pkt.BodyLen(), "content", hex.EncodeToString(pkt.Body()))
			return 0, io.EOF
		}
		d = pkt.Body()[4:]
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
	close(s.out)
}

func (s *Tunnel) handleWrite() {
	seq := uint32(0)
	buf := make([]byte, s.mtu)
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
		for len(b) > int(s.mtu)-4 {
			logrus.Debugln("[tunnel] seq", seq, "split buffer")
			binary.LittleEndian.PutUint32(buf[:4], seq)
			seq++
			copy(buf[4:], b[:s.mtu-4])
			_, err := s.l.WriteAndPut(
				head.NewPacket(head.ProtoData, s.src, s.peerip, s.dest, buf), false,
			)
			if err != nil {
				logrus.Errorln("[tunnel] seq", seq-1, "write err:", err)
				return
			}
			logrus.Debugln("[tunnel] seq", seq-1, "write succeeded")
			b = b[s.mtu-4:]
		}
		binary.LittleEndian.PutUint32(buf[:4], seq)
		seq++
		copy(buf[4:], b)
		_, err := s.l.WriteAndPut(
			head.NewPacket(head.ProtoData, s.src, s.peerip, s.dest, buf[:len(b)+4]), false,
		)
		if err != nil {
			logrus.Errorln("[tunnel] seq", seq-1, "write err:", err)
			break
		}
		logrus.Debugln("[tunnel] seq", seq-1, "write succeeded")
	}
}

func (s *Tunnel) handleRead() {
	seq := uint32(0)
	seqmap := make(map[uint32]*head.Packet)
	for {
		if p, ok := seqmap[seq]; ok {
			logrus.Debugln("[tunnel] dispatch cached seq", seq)
			delete(seqmap, seq)
			seq++
			s.out <- p
			continue
		}
		p := s.l.Read()
		if p == nil {
			logrus.Errorln("[tunnel] read recv nil")
			break
		}
		end := 64
		endl := "..."
		if p.BodyLen() < 64 {
			end = p.BodyLen()
			endl = "."
		}
		logrus.Debugln("[tunnel] read recv", hex.EncodeToString(p.Body()[:end]), endl)
		recvseq := binary.LittleEndian.Uint32(p.Body()[:4])
		if recvseq == seq {
			logrus.Debugln("[tunnel] dispatch seq", seq)
			seq++
			s.out <- p
			continue
		}
		seqmap[recvseq] = p
		logrus.Debugln("[tunnel] cache seq", recvseq)
	}
}

package tunnel

import (
	"encoding/binary"
	"encoding/hex"
	"io"
	"net"

	"github.com/sirupsen/logrus"

	_ "github.com/fumiama/WireGold/gold/p2p/ip"      // support ip connection
	_ "github.com/fumiama/WireGold/gold/p2p/tcp"     // support tcp connection
	_ "github.com/fumiama/WireGold/gold/p2p/udp"     // support udp connection
	_ "github.com/fumiama/WireGold/gold/p2p/udplite" // support udplite connection
	_ "github.com/fumiama/WireGold/gold/proto/data"  // support data proto
	_ "github.com/fumiama/WireGold/gold/proto/hello" // support hello proto
	_ "github.com/fumiama/WireGold/gold/proto/nat"   // support nat proto

	"github.com/fumiama/WireGold/config"
	"github.com/fumiama/WireGold/gold/head"
	"github.com/fumiama/WireGold/gold/link"
)

type Tunnel struct {
	l        *link.Link
	in       chan []byte
	out      chan link.LinkData
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
		s.out = make(chan link.LinkData, 4)
		s.peerip = net.ParseIP(peer)
	} else {
		logrus.Errorln("[tunnel] create err:", err)
	}
	return
}

func (s *Tunnel) Start(srcport, destport, mtu uint16) {
	logrus.Infoln("[tunnel] start port through", srcport, "->", destport, "mtu", mtu)
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
		if !pkt.D.HasInit() {
			return 0, io.EOF
		}
		if pkt.H.Size() < 4 {
			logrus.Warnln("[tunnel] unexpected packet data len", pkt.H.Size(), "content", hex.EncodeToString(pkt.D.Trans()))
			return 0, io.EOF
		}
		d = pkt.D.Trans()[4:]
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
		if config.ShowDebugLog {
			logrus.Debugln("[tunnel] write send", hex.EncodeToString(b[:end]), endl)
		}
		if b == nil {
			logrus.Errorln("[tunnel] write recv nil")
			break
		}
		if config.ShowDebugLog {
			logrus.Debugln("[tunnel] writing", len(b), "bytes...")
		}
		for len(b) > int(s.mtu)-4 {
			if config.ShowDebugLog {
				logrus.Debugln("[tunnel] seq", seq, "split buffer")
			}
			binary.LittleEndian.PutUint32(buf[:4], seq)
			seq++
			copy(buf[4:], b[:s.mtu-4])
			s.l.WritePacket(head.ProtoData, buf, s.l.Me().TTL())
			if config.ShowDebugLog {
				logrus.Debugln("[tunnel] seq", seq-1, "written")
			}
			b = b[s.mtu-4:]
		}
		binary.LittleEndian.PutUint32(buf[:4], seq)
		seq++
		copy(buf[4:], b)
		s.l.WritePacket(head.ProtoData, buf[:len(b)+4], s.l.Me().TTL())
		if config.ShowDebugLog {
			logrus.Debugln("[tunnel] seq", seq-1, "written")
		}
	}
}

func (s *Tunnel) handleRead() {
	seq := uint32(0)
	seqmap := make(map[uint32]link.LinkData)
	for {
		if p, ok := seqmap[seq]; ok {
			if config.ShowDebugLog {
				logrus.Debugln("[tunnel] dispatch cached seq", seq)
			}
			delete(seqmap, seq)
			seq++
			s.out <- p
			continue
		}
		p := s.l.Read()
		if !p.D.HasInit() {
			logrus.Errorln("[tunnel] read recv nil")
			break
		}
		end := 64
		endl := "..."
		pp := &p.H
		if pp.Size() < 64 {
			end = pp.Size()
			endl = "."
		}
		var recvseq uint32
		p.D.V(func(b []byte) {
			if config.ShowDebugLog {
				logrus.Debugln("[tunnel] read recv", hex.EncodeToString(b[:end]), endl)
			}
			recvseq = binary.LittleEndian.Uint32(b[:4])
		})
		if recvseq == seq {
			if config.ShowDebugLog {
				logrus.Debugln("[tunnel] dispatch seq", seq)
			}
			seq++
			s.out <- p
			continue
		}
		seqmap[recvseq] = p
		if config.ShowDebugLog {
			logrus.Debugln("[tunnel] cache seq", recvseq)
		}
	}
}

package tcp

import (
	"encoding/binary"
	"errors"
	"io"
	"net"
	"time"

	"github.com/fumiama/WireGold/config"
	"github.com/fumiama/WireGold/helper"
	"github.com/sirupsen/logrus"
)

var (
	ErrInvalidMagic = errors.New("invalid magic")
)

type packetType uint8

const (
	packetTypeKeepAlive packetType = iota
	packetTypeNormal
	packetTypeSubKeepAlive
	packetTypeTop
)

var (
	magicbuf = []byte("GET ")
	magic    = binary.LittleEndian.Uint32(magicbuf)
)

type packet struct {
	typ packetType
	len uint16
	dat []byte
	io.ReaderFrom
	io.WriterTo
}

func (p *packet) pack() (net.Buffers, func()) {
	d, cl := helper.OpenWriterF(func(w *helper.Writer) {
		w.WriteByte(byte(p.typ))
		w.WriteUInt16(p.len)
	})
	return net.Buffers{magicbuf, d, p.dat}, cl
}

func (p *packet) Read(_ []byte) (int, error) {
	panic("stub")
}

func (p *packet) Write(_ []byte) (int, error) {
	panic("stub")
}

func (p *packet) ReadFrom(r io.Reader) (n int64, err error) {
	var buf [4]byte
	cnt, err := io.ReadFull(r, buf[:])
	n = int64(cnt)
	if err != nil {
		return
	}
	if binary.LittleEndian.Uint32(buf[:]) != magic {
		err = ErrInvalidMagic
		if config.ShowDebugLog {
			logrus.Debugf("[tcp] expect magic %08x but got %08x", magic, binary.LittleEndian.Uint32(buf[:]))
		}
		return
	}
	cnt, err = io.ReadFull(r, buf[:3])
	n += int64(cnt)
	if err != nil {
		return
	}
	p.typ = packetType(buf[0])
	p.len = binary.LittleEndian.Uint16(buf[1:3])
	w := helper.SelectWriter()
	copied, err := io.CopyN(w, r, int64(p.len))
	n += copied
	if err != nil {
		return
	}
	p.dat = w.Bytes()
	return
}

func (p *packet) WriteTo(w io.Writer) (n int64, err error) {
	buf, cl := p.pack()
	defer cl()
	return io.Copy(w, &buf)
}

func isvalid(tcpconn *net.TCPConn, timeout time.Duration) (issub, ok bool) {
	pckt := packet{}

	stopch := make(chan struct{})
	t := time.AfterFunc(timeout, func() {
		stopch <- struct{}{}
	})

	var err error
	copych := make(chan struct{})
	go func() {
		_, err = io.Copy(&pckt, tcpconn)
		copych <- struct{}{}
	}()

	select {
	case <-stopch:
		if config.ShowDebugLog {
			logrus.Debugln("[tcp] validate recv from", tcpconn.RemoteAddr(), "timeout")
		}
		return
	case <-copych:
		t.Stop()
	}

	if err != nil {
		if config.ShowDebugLog {
			logrus.Debugln("[tcp] validate recv from", tcpconn.RemoteAddr(), "err:", err)
		}
		return
	}
	if pckt.typ != packetTypeKeepAlive && pckt.typ != packetTypeSubKeepAlive {
		if config.ShowDebugLog {
			logrus.Debugln("[tcp] validate got invalid typ", pckt.typ, "from", tcpconn.RemoteAddr())
		}
		return
	}

	if config.ShowDebugLog {
		logrus.Debugln("[tcp] passed validate recv from", tcpconn.RemoteAddr())
	}
	return pckt.typ == packetTypeSubKeepAlive, true
}

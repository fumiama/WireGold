package tcp

import (
	"encoding/binary"
	"io"
	"net"

	"github.com/fumiama/WireGold/helper"
)

type packetType uint8

const (
	packetTypeKeepAlive packetType = iota
	packetTypeNormal
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
	return net.Buffers{d, p.dat}, cl
}

func (p *packet) Read(_ []byte) (int, error) {
	panic("stub")
}

func (p *packet) Write(_ []byte) (int, error) {
	panic("stub")
}

func (p *packet) ReadFrom(r io.Reader) (n int64, err error) {
	var buf [3]byte
	cnt, err := io.ReadFull(r, buf[:])
	n = int64(cnt)
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

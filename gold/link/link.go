package link

import (
	"errors"
	"net"

	"github.com/fumiama/WireGold/gold/head"
)

type Link struct {
	conn          net.Conn
	peer          *Identity
	hasKeepRuning bool
}

func Connect(peer string) (l Link, err error) {
	p, ok := peers[peer]
	if ok {
		l.conn, err = net.Dial("udp", peer)
		l.peer = p
	} else {
		err = errors.New("peer not exist")
	}
	return
}

func (l *Link) Close() {
	l.conn.Close()
}

func (l *Link) Read(p *head.Packet) (n int, err error) {
	d := make([]byte, 1024)
	n, err = l.conn.Read(d)
	if err == nil {
		n, err = l.peer.Decode(d)
		if err == nil {
			p.UnMashal(d)
		}
	}
	return
}

func (l *Link) Write(p *head.Packet) (n int, err error) {
	d := p.Mashal()
	_, err = l.peer.Encode(d)
	if err == nil {
		n, err = l.conn.Write(d)
	}
	return
}

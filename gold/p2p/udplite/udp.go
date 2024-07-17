//go:build !darwin

package udplite

import (
	"net"

	"github.com/fumiama/WireGold/gold/p2p"
)

type EndPoint net.UDPAddr

func (ep *EndPoint) String() string {
	return (*net.UDPAddr)(ep).String()
}

func (ep *EndPoint) Network() string {
	return (*net.UDPAddr)(ep).Network()
}

func (ep *EndPoint) Euqal(ep2 p2p.EndPoint) bool {
	if ep == nil || ep2 == nil {
		return ep == nil && ep2 == nil
	}
	udpep2, ok := ep2.(*EndPoint)
	if !ok {
		return false
	}
	udpep1 := ep
	return udpep1.IP.Equal(udpep2.IP) && udpep1.Port == udpep2.Port && udpep1.Zone == udpep2.Zone
}

func (ep *EndPoint) Listen() (p2p.Conn, error) {
	conn, err := listenUDPLite((*net.UDPAddr)(ep).Network(), (*net.UDPAddr)(ep))
	return (*Conn)(conn), err
}

type Conn net.UDPConn

func (conn *Conn) Close() error {
	return (*net.UDPConn)(conn).Close()
}

func (conn *Conn) String() string {
	return (*net.UDPConn)(conn).LocalAddr().String()
}

func (conn *Conn) LocalAddr() p2p.EndPoint {
	ep, _ := NewEndpoint((*net.UDPConn)(conn).LocalAddr().String())
	return ep
}

func (conn *Conn) ReadFromPeer(b []byte) (int, p2p.EndPoint, error) {
	n, addr, err := (*net.UDPConn)(conn).ReadFromUDP(b)
	return n, (*EndPoint)(addr), err
}

func (conn *Conn) WriteToPeer(b []byte, ep p2p.EndPoint) (int, error) {
	udpep, ok := ep.(*EndPoint)
	if !ok {
		return 0, p2p.ErrEndpointTypeMistatch
	}
	return (*net.UDPConn)(conn).WriteTo(b, (*net.UDPAddr)(udpep))
}

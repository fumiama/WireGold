package ip

import (
	"net"
	"strconv"

	"github.com/fumiama/WireGold/gold/p2p"
)

type EndPoint struct {
	addr *net.IPAddr
	ptcl uint
}

func (ep *EndPoint) String() string {
	return ep.addr.String()
}

func (ep *EndPoint) Network() string {
	return ep.addr.Network()
}

func (ep *EndPoint) Euqal(ep2 p2p.EndPoint) bool {
	if ep == nil || ep2 == nil {
		return ep == nil && ep2 == nil
	}
	ipep2, ok := ep2.(*EndPoint)
	if !ok {
		return false
	}
	ipep1 := ep
	return ipep1.addr.IP.Equal(ipep2.addr.IP) &&
		ipep1.addr.Zone == ipep2.addr.Zone
}

func (ep *EndPoint) Listen() (p2p.Conn, error) {
	conn, err := net.ListenIP(
		"ip:"+strconv.Itoa(int(ep.ptcl)),
		ep.addr,
	)
	return &Conn{
		ep:   ep,
		conn: conn,
	}, err
}

type Conn struct {
	ep   *EndPoint
	conn *net.IPConn
}

func (conn *Conn) Close() error {
	return conn.conn.Close()
}

func (conn *Conn) String() string {
	return conn.conn.LocalAddr().String()
}

func (conn *Conn) LocalAddr() p2p.EndPoint {
	ep, _ := NewEndpoint(conn.conn.LocalAddr().String())
	return ep
}

func (conn *Conn) ReadFromPeer(b []byte) (int, p2p.EndPoint, error) {
	n, addr, err := conn.conn.ReadFromIP(b)
	return n, &EndPoint{
		addr: addr,
		ptcl: conn.ep.ptcl,
	}, err
}

func (conn *Conn) WriteToPeer(b []byte, ep p2p.EndPoint) (int, error) {
	ipep, ok := ep.(*EndPoint)
	if !ok {
		return 0, p2p.ErrEndpointTypeMistatch
	}
	return conn.conn.WriteToIP(b, ipep.addr)
}

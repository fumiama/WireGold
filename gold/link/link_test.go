package link

import (
	"net"
	"testing"
)

func TestUDP(t *testing.T) {
	t.Log("test start")
	lconn, err := net.ListenUDP("udp", &net.UDPAddr{Port: 1234})
	if err == nil {
		dconn, err := net.DialUDP("udp", &net.UDPAddr{Port: 1235}, &net.UDPAddr{Port: 1234})
		if err != nil {
			t.Fatal(err)
		}
		_, err = dconn.Write(([]byte)("1234567890"))
		t.Log("write succ")
		if err != nil {
			t.Fatal(err)
		}
		d := make([]byte, 10)
		_, err = lconn.Read(d)
		t.Log("read succ")
		if err != nil {
			t.Fatal(err)
		}
		t.Log(d)
	} else {
		t.Fatal(err)
	}
}

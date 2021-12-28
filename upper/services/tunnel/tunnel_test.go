package tunnel

import (
	"testing"

	curve "github.com/fumiama/go-x25519"
	"github.com/sirupsen/logrus"

	"github.com/fumiama/WireGold/gold/link"
)

func TestTunnel(t *testing.T) {
	logrus.SetLevel(logrus.DebugLevel)
	selfpk, err := curve.New(nil)
	if err != nil {
		panic(err)
	}
	peerpk, err := curve.New(nil)
	if err != nil {
		panic(err)
	}
	m := link.NewMe(selfpk.Private(), "192.168.1.2", "127.0.0.1:1236")
	m.AddPeer("192.168.1.3", peerpk.Public(), "127.0.0.1:1237", nil, 0, false)
	p := link.NewMe(peerpk.Private(), "192.168.1.3", "127.0.0.1:1237")
	p.AddPeer("192.168.1.2", selfpk.Public(), "127.0.0.1:1236", nil, 0, false)
	tunnme, err := Create(&m, "192.168.1.3", 1, 1)
	if err != nil {
		t.Fatal(err)
	}
	tunnpeer, err := Create(&p, "192.168.1.2", 1, 1)
	if err != nil {
		t.Fatal(err)
	}
	sendb := ([]byte)("1234")
	tunnme.Write(sendb)
	buf := make([]byte, 4)
	tunnpeer.Read(buf)
	if string(sendb) != string(buf) {
		t.Log("error: recv", buf)
		t.Fail()
	}
}

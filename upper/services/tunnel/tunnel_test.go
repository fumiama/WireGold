package tunnel

import (
	"crypto/rand"
	"encoding/hex"
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
	t.Log("my priv key:", hex.EncodeToString(selfpk.Private()[:]))
	t.Log("my publ key:", hex.EncodeToString(selfpk.Public()[:]))
	t.Log("peer priv key:", hex.EncodeToString(peerpk.Private()[:]))
	t.Log("peer publ key:", hex.EncodeToString(peerpk.Public()[:]))

	m := link.NewMe(selfpk.Private(), "192.168.1.2/32", "127.0.0.1:1236", nil, 1, 1, 4096)
	m.AddPeer("192.168.1.3", peerpk.Public(), "127.0.0.1:1237", []string{"192.168.1.3/32"}, nil, 0, 0, false, false)
	p := link.NewMe(peerpk.Private(), "192.168.1.3/32", "127.0.0.1:1237", nil, 1, 1, 4096)
	p.AddPeer("192.168.1.2", selfpk.Public(), "127.0.0.1:1236", []string{"192.168.1.2/32"}, nil, 0, 0, false, false)
	tunnme, err := Create(&m, "192.168.1.3")
	if err != nil {
		t.Fatal(err)
	}
	tunnme.Start(1, 1, 4096)
	tunnpeer, err := Create(&p, "192.168.1.2")
	if err != nil {
		t.Fatal(err)
	}
	tunnpeer.Start(1, 1, 4096)

	sendb := ([]byte)("1234")
	tunnme.Write(sendb)
	buf := make([]byte, 4)
	tunnpeer.Read(buf)
	if string(sendb) != string(buf) {
		t.Log("error: recv", buf)
		t.Fail()
	}

	sendb = make([]byte, 4096)
	rand.Read(sendb)
	tunnme.Write(sendb)
	buf = make([]byte, 4096)
	tunnpeer.Read(buf)
	if string(sendb) != string(buf) {
		t.Fatal("error: recv 4096 bytes data")
	}

	sendb = make([]byte, 131072)
	rand.Read(sendb)
	tunnme.Write(sendb)
	buf = make([]byte, 131072)
	for i := 0; i < 32; i++ {
		tunnpeer.Read(buf[i*4096:])
	}
	if string(sendb) != string(buf) {
		t.Fatal("error: recv 131072 bytes data")
	}

	tunnme.Stop()
	tunnpeer.Stop()
}

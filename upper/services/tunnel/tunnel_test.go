package tunnel

import (
	"crypto/rand"
	"encoding/hex"
	"io"
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

	m := link.NewMe(&link.MyConfig{
		MyIPwithMask: "192.168.1.2/32",
		MyEndpoint:   "127.0.0.1:1236",
		PrivateKey:   selfpk.Private(),
		SrcPort:      1,
		DstPort:      1,
		MTU:          4096,
	})
	m.AddPeer(&link.PeerConfig{
		PeerIP:     "192.168.1.3",
		EndPoint:   "127.0.0.1:1237",
		AllowedIPs: []string{"192.168.1.3/32"},
		PubicKey:   peerpk.Public(),
		MTU:        4096,
	})
	p := link.NewMe(&link.MyConfig{
		MyIPwithMask: "192.168.1.3/32",
		MyEndpoint:   "127.0.0.1:1237",
		PrivateKey:   peerpk.Private(),
		SrcPort:      1,
		DstPort:      1,
		MTU:          4096,
	})
	p.AddPeer(&link.PeerConfig{
		PeerIP:     "192.168.1.2",
		EndPoint:   "127.0.0.1:1236",
		AllowedIPs: []string{"192.168.1.2/32"},
		PubicKey:   selfpk.Public(),
		MTU:        4096,
	})
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

	sendb = make([]byte, 65535)
	rand.Read(sendb)
	n, _ := tunnme.Write(sendb)
	t.Log("write", n, "bytes")
	buf = make([]byte, 65535)
	n, _ = io.ReadFull(&tunnpeer, buf)
	t.Log("read", n, "bytes")
	if string(sendb) != string(buf) {
		t.Fatal("error: recv 65535 bytes data")
	}

	tunnme.Stop()
	tunnpeer.Stop()
}

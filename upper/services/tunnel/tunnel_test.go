package tunnel

import (
	"testing"

	"github.com/fumiama/WireGold/gold/link"
	"github.com/sirupsen/logrus"
)

func TestTunnel(t *testing.T) {
	logrus.SetLevel(logrus.DebugLevel)
	link.SetMyself([32]byte{}, "192.168.1.2", "127.0.0.1:1236")
	link.AddPeer("192.168.1.2", [32]byte{}, "127.0.0.1:1236", nil, 0)
	tunn, err := Create("192.168.1.2", 1, 1)
	if err != nil {
		t.Error(err)
	} else {
		sendb := ([]byte)("1234")
		tunn.Write(sendb)
		p := make([]byte, 4)
		tunn.Read(p)
		if string(sendb) != string(p) {
			t.Log("error: recv", p)
			t.Fail()
		}
	}
}

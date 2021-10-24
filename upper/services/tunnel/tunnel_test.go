package tunnel

import (
	"testing"

	"github.com/fumiama/WireGold/gold/link"
)

func TestTunnel(t *testing.T) {
	link.SetMyself([32]byte{}, "127.0.0.1:1234")
	link.AddPeer("192.168.1.1", [32]byte{}, "127.0.0.1:1235", 0)
	link.Listen("127.0.0.1:1234")
	link.Listen("127.0.0.1:1235")
	tunn, err := Create("192.168.1.1", 1, 1)
	if err != nil {
		t.Error(err)
	} else {
		tunn.Write(([]byte)("1234"))
		p := make([]byte, 4)
		tunn.Read(p)
		t.Log(p)
	}
}

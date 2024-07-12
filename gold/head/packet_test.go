package head

import (
	crand "crypto/rand"
	"math/rand"
	"net"
	"testing"
)

func TestMarshalUnmarshal(t *testing.T) {
	data := make([]byte, 4096)
	_, err := crand.Read(data)
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 0x7ff; i++ {
		proto := uint8(rand.Intn(255))
		teatype := uint8(rand.Intn(32))
		srcPort := uint16(rand.Intn(65535))
		dstPort := uint16(rand.Intn(65535))
		src := make(net.IP, 4)
		_, err = crand.Read(src)
		if err != nil {
			t.Fatal(err)
		}
		dst := make(net.IP, 4)
		_, err = crand.Read(dst)
		if err != nil {
			t.Fatal(err)
		}
		p := NewPacket(proto, srcPort, dst, dstPort, data)
		p.FillHash()
		d, cl := p.Marshal(src, teatype, uint16(i), uint32(len(data)), 0, true, false)
		p = SelectPacket()
		ok, err := p.Unmarshal(d)
		cl()
		if !ok {
			t.Fatal("index", i)
		}
		if err != nil {
			t.Fatal(err)
		}
		if !p.IsVaildHash() {
			t.Fatal("index", i)
		}
		if p.Proto != proto {
			t.Fatal("index", i)
		}
		if p.CipherIndex() != teatype {
			t.Fatal("index", i, "expect", teatype, "got", p.CipherIndex())
		}
		if p.SrcPort != srcPort {
			t.Fatal("index", i)
		}
		if p.DstPort != dstPort {
			t.Fatal("index", i)
		}
		if !p.Src.Equal(src) {
			t.Fatal("index", i)
		}
		if !p.Dst.Equal(dst) {
			t.Fatal("index", i)
		}
		if p.AdditionalData() != uint16(i) {
			t.Fatal("index", i)
		}
	}
}

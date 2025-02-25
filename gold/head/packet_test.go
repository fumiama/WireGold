package head

import (
	crand "crypto/rand"
	"encoding/hex"
	"math/rand"
	"net"
	"testing"

	"github.com/fumiama/orbyte/pbuf"
)

func TestMarshalUnmarshal(t *testing.T) {
	data := pbuf.NewBytes(4096)
	n, err := crand.Read(data.Bytes())
	if n != 4096 {
		t.Fatal("unexpected")
	}
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
		p := NewPacketPartial(proto, srcPort, dst, dstPort, data.SliceTo(i))
		p.Pointer().FillHash()
		d := p.Pointer().MarshalWith(src, teatype, uint16(i), uint32(i), 0, true, false)
		t.Log("data:", hex.EncodeToString(d.Bytes()))
		p, err := ParsePacketHeader(d.Bytes())
		if err != nil {
			t.Fatal("index", i, err)
		}
		ok := p.Pointer().ParseData(d.Bytes())
		if !ok {
			t.Fatal("index", i)
		}
		if !p.Pointer().IsVaildHash() {
			t.Fatal("index", i, "expect body", hex.EncodeToString(data.SliceTo(i).Bytes()), "got", hex.EncodeToString(p.Pointer().Body()))
		}
		if p.Pointer().Proto != proto {
			t.Fatal("index", i)
		}
		if p.Pointer().CipherIndex() != teatype {
			t.Fatal("index", i, "expect", teatype, "got", p.Pointer().CipherIndex())
		}
		if p.Pointer().SrcPort != srcPort {
			t.Fatal("index", i)
		}
		if p.Pointer().DstPort != dstPort {
			t.Fatal("index", i)
		}
		if !p.Pointer().Src.Equal(src) {
			t.Fatal("index", i)
		}
		if !p.Pointer().Dst.Equal(dst) {
			t.Fatal("index", i)
		}
		if p.Pointer().AdditionalData() != uint16(i) {
			t.Fatal("index", i)
		}
	}
}

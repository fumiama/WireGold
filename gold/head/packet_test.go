package head

import (
	"bytes"
	crand "crypto/rand"
	"encoding/hex"
	"math/rand"
	"net"
	"runtime"
	"sync"
	"testing"

	"github.com/fumiama/WireGold/internal/algo"
	"github.com/fumiama/WireGold/internal/bin"
)

func TestBuilderNative(t *testing.T) {
	wg := sync.WaitGroup{}
	wg.Add(4096)
	for i := 0; i < 4096; i++ {
		go func(i int) {
			defer runtime.GC()
			defer wg.Done()
			dat := BuildPacketFromBytes(NewPacketBuilder().Proto(3).TTL(0xff).
				Src(net.IPv4(1, 2, 3, 4), 5).Dst(net.IPv4(6, 7, 8, 9), 10).
				With([]byte("0123456789")).Hash().Plain(0x12, 0x0345).
				Split(16384, false)[0]).Trans()
			s := hex.EncodeToString(dat)
			if s[:8] != "12004593" {
				panic(s[:8])
			}
			if s[16:48] != "03ff05000a0000000102030406070809" {
				panic(s[16:48])
			}
			if s[80:] != "30313233343536373839" {
				panic(s[80:])
			}
			p, err := ParsePacketHeader(dat)
			if err != nil {
				panic(err)
			}
			p.B(func(buf []byte, p *Packet) {
				ok := p.WriteDataSegment(dat, buf)
				if !ok {
					panic(i)
				}
				if !algo.IsVaildBlake2bHash8(p.PreCRC64(), buf) {
					panic(i)
				}
				if p.Proto != 3 {
					panic(i)
				}
				if p.CipherIndex() != 0x12 {
					panic(i)
				}
				if p.SrcPort != 5 {
					panic(i)
				}
				if p.DstPort != 10 {
					panic(i)
				}
				if !bytes.Equal(p.src[:], net.IPv4(1, 2, 3, 4).To4()) {
					panic(i)
				}
				if !bytes.Equal(p.dst[:], net.IPv4(6, 7, 8, 9).To4()) {
					panic(i)
				}
				if p.AdditionalData() != 0x0345 {
					panic(i)
				}
			})
		}(i)
	}
	wg.Wait()
}

func TestBuilderBE(t *testing.T) {
	wg := sync.WaitGroup{}
	wg.Add(4096)
	bin.IsLittleEndian = false
	for i := 0; i < 4096; i++ {
		go func(i int) {
			defer runtime.GC()
			defer wg.Done()
			dat := BuildPacketFromBytes(NewPacketBuilder().Proto(3).TTL(0xff).
				Src(net.IPv4(1, 2, 3, 4), 5).Dst(net.IPv4(6, 7, 8, 9), 10).
				With([]byte("0123456789")).Hash().Plain(0x12, 0x0345).
				Split(16384, false)[0]).Trans()
			s := hex.EncodeToString(dat)
			if s[:8] != "12004593" {
				panic(s[:8])
			}
			if s[16:48] != "03ff05000a0000000102030406070809" {
				panic(s[16:48])
			}
			if s[80:] != "30313233343536373839" {
				panic(s[80:])
			}
			p, err := ParsePacketHeader(dat)
			if err != nil {
				panic(err)
			}
			p.B(func(buf []byte, p *Packet) {
				ok := p.WriteDataSegment(dat, buf)
				if !ok {
					panic(i)
				}
				if !algo.IsVaildBlake2bHash8(p.PreCRC64(), buf) {
					panic(i)
				}
				if p.Proto != 3 {
					panic(i)
				}
				if p.CipherIndex() != 0x12 {
					panic(i)
				}
				if p.SrcPort != 5 {
					panic(i)
				}
				if p.DstPort != 10 {
					panic(i)
				}
				if !bytes.Equal(p.src[:], net.IPv4(1, 2, 3, 4).To4()) {
					panic(i)
				}
				if !bytes.Equal(p.dst[:], net.IPv4(6, 7, 8, 9).To4()) {
					panic(i)
				}
				if p.AdditionalData() != 0x0345 {
					panic(i)
				}
			})
		}(i)
	}
	wg.Wait()
}

func TestMarshalUnmarshal(t *testing.T) {
	// logrus.SetLevel(logrus.DebugLevel)
	data := make([]byte, 4096)
	n, err := crand.Read(data)
	if n != 4096 {
		t.Fatal("unexpected")
	}
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 4096; i++ {
		proto := uint8(rand.Intn(int(ProtoTop)))
		teatype := uint8(rand.Intn(32))
		srcPort := uint16(rand.Intn(65535))
		dstPort := uint16(rand.Intn(65535))
		src := make(net.IP, 4)
		_, err := crand.Read(src)
		if err != nil {
			t.Fatal(err)
		}
		dst := make(net.IP, 4)
		_, err = crand.Read(dst)
		if err != nil {
			t.Fatal(err)
		}
		dat := BuildPacketFromBytes(NewPacketBuilder().Proto(proto).
			Src(src, srcPort).Dst(dst, dstPort).
			With(data[:i]).Hash().Plain(teatype, uint16(i&0x7ff)).
			Split(16384, false)[0]).Trans()
		t.Log("pkt:", hex.EncodeToString(dat))
		p, err := ParsePacketHeader(dat)
		if err != nil {
			t.Fatal("index", i, err)
		}
		p.B(func(buf []byte, p *Packet) {
			ok := p.WriteDataSegment(dat, buf)
			if !ok {
				t.Fatal("index", i)
			}
			if !algo.IsVaildBlake2bHash8(p.PreCRC64(), buf) {
				t.Fatal("index", i, "expect body", hex.EncodeToString(data[:i]), "got", hex.EncodeToString(buf[8:]))
			}
			if p.Proto != FlagsProto(proto) {
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
			if !bytes.Equal(p.src[:], src) {
				t.Fatal("index", i)
			}
			if !bytes.Equal(p.dst[:], dst) {
				t.Fatal("index", i)
			}
			if p.AdditionalData() != uint16(i&0x7ff) {
				t.Fatal("index", i)
			}
			if !bytes.Equal(buf[8:], data[:i]) {
				t.Fatal("index", i)
			}
		})
	}
}

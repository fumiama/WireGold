package link

import (
	"bytes"
	"crypto/rand"
	"io"
	"testing"

	"golang.org/x/crypto/chacha20poly1305"
)

func TestXOR(t *testing.T) {
	m := Me{
		mask: 0x12345678_90abcdef,
	}
	buf := make([]byte, 4096)
	buf2 := make([]byte, 4096)
	for i := 1; i < 4096; i++ {
		data := buf[:i]
		orgdata := buf2[:i]
		r1 := bytes.NewBuffer(data[:0])
		r2 := bytes.NewBuffer(orgdata[:0])
		w := io.MultiWriter(r1, r2)
		_, err := io.CopyN(w, rand.Reader, int64(i))
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(m.xordec(m.xorenc(r1.Bytes())), r2.Bytes()) {
			t.Fatal("unexpected xor at", i)
		}
	}
}

func TestXChacha20(t *testing.T) {
	l := Link{}
	k := make([]byte, 32)
	_, err := rand.Read(k)
	if err != nil {
		t.Fatal(err)
	}
	l.aead, err = chacha20poly1305.NewX(k)
	if err != nil {
		t.Fatal(err)
	}
	data := []byte("12345678")
	for i := uint64(0); i < 100000; i++ {
		if !bytes.Equal(l.DecodePreshared(uint16(i), l.EncodePreshared(uint16(i), data)), data) {
			t.Fatal("unexpected preshared at", i, "addt", uint16(i))
		}
	}
}

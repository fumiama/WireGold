package link

import (
	"bytes"
	"crypto/rand"
	"io"
	"testing"
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

package link

import (
	"bytes"
	"crypto/rand"
	"testing"
)

func TestXOR(t *testing.T) {
	m := Me{
		mask: 0x12345678_90abcdef,
	}
	buf := make([]byte, 65535)
	for i := 1; i < 65536; i++ {
		data := buf[:i]
		_, err := rand.Read(data)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(m.xor(m.xor(data)), data) {
			t.Fatal("unexpected xor at ", i)
		}
	}
}

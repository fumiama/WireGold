package algo

import (
	"bytes"
	"crypto/rand"
	"encoding/binary"
	"encoding/hex"
	"io"
	"testing"

	"golang.org/x/crypto/chacha20poly1305"
)

func TestXOR(t *testing.T) {
	mask := uint64(0x12345678_90abcdef)
	buf := make([]byte, 4096)
	buf2 := make([]byte, 4096)
	for i := 0; i < 4096; i++ {
		data := buf[:i]
		orgdata := buf2[:i]
		r1 := bytes.NewBuffer(data[:0])
		r2 := bytes.NewBuffer(orgdata[:0])
		w := io.MultiWriter(r1, r2)
		_, err := io.CopyN(w, rand.Reader, int64(i))
		if err != nil {
			t.Fatal(err)
		}
		seq, dec := DecodeXOR(EncodeXOR(r1.Bytes(), mask, uint32(i)), mask)
		if !bytes.Equal(dec, r2.Bytes()) {
			t.Fatal("unexpected xor at", i, "except", hex.EncodeToString(r2.Bytes()), "got", hex.EncodeToString(dec))
		}
		if seq != uint32(i) {
			t.Fatal("unexpected xor at", i, "seq", seq)
		}
	}
}

func TestXChacha20(t *testing.T) {
	k := make([]byte, 32)
	_, err := rand.Read(k)
	if err != nil {
		t.Fatal(err)
	}
	aead, err := chacha20poly1305.NewX(k)
	if err != nil {
		t.Fatal(err)
	}
	data := make([]byte, 4096)
	_, err = rand.Read(data)
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 4096; i++ {
		db, err := DecodeAEAD(aead, uint16(i), EncodeAEAD(aead, uint16(i), data[:i]))
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(db, data[:i]) {
			t.Fatal("unexpected preshared at idx(len)", i, "addt", uint16(i))
		}
	}
}

func TestExpandKeyUnit(t *testing.T) {
	k1 := byte(0b10001010)
	k2 := byte(0b10111010)     // rev 01011101
	v := expandkeyunit(k1, k2) // x1x0x0x0x1x0x1x0 | 0x1x0x1x1x1x0x1x = 0110001011100110
	if v != 0b0110001011100110 {
		buf := [2]byte{}
		binary.BigEndian.PutUint16(buf[:], v)
		t.Fatal(hex.EncodeToString(buf[:]))
	}
}

func TestMixKeys(t *testing.T) {
	k1, _ := hex.DecodeString("ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff")
	k2, _ := hex.DecodeString("0000000000000000000000000000000000000000000000000000000000000000")
	k := MixKeys(k1, k2)
	kexp, _ := hex.DecodeString("55555555555555555555555555555555555555555555555555555555555555555555555555555555555555555555555555555555555555555555555555555555")
	if !bytes.Equal(k, kexp) {
		t.Fatal(hex.EncodeToString(k))
	}
	k1, _ = hex.DecodeString("1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef")
	k2, _ = hex.DecodeString("deadbeef1239876540deadbeef1239876540deadbeef1239876540abcdef4567")
	k = MixKeys(k1, k2)
	kexp, _ = hex.DecodeString("2ca9188d3ebb4a9f22e34d4479d857fca48390253ebbe23f22cbcf6e59507ddc06a9b08794316abfa26b67cedb7a5d542c8912adb493c0352aebe76e73dadf7e")
	if !bytes.Equal(k, kexp) {
		t.Fatal(hex.EncodeToString(k))
	}
}

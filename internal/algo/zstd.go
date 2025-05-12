package algo

import (
	"bytes"
	"io"

	"github.com/fumiama/WireGold/internal/bin"
	"github.com/fumiama/orbyte/pbuf"
	"github.com/klauspost/compress/zstd"
)

func EncodeZstd(data []byte) []byte {
	return bin.SelectWriter().P(func(w *pbuf.Buffer) {
		enc, err := zstd.NewWriter(w, zstd.WithEncoderLevel(zstd.SpeedFastest))
		if err != nil {
			panic(err)
		}
		_, err = io.Copy(enc, bytes.NewReader(data))
		if err != nil {
			panic(err)
		}
		err = enc.Close()
		if err != nil {
			panic(err)
		}
	}).ToBytes().Copy().Ignore().Trans()
}

func DecodeZstd(data []byte) (b []byte, err error) {
	dec, err := zstd.NewReader(bytes.NewReader(data))
	if err != nil {
		return
	}

	b = bin.SelectWriter().P(func(w *pbuf.Buffer) {
		_, err = io.Copy(w, dec)
		dec.Close()
	}).ToBytes().Copy().Ignore().Trans()

	return
}

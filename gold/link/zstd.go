package link

import (
	"bytes"
	"io"

	"github.com/fumiama/WireGold/helper"
	"github.com/fumiama/orbyte/pbuf"
	"github.com/klauspost/compress/zstd"
)

func encodezstd(data []byte) pbuf.Bytes {
	w := helper.SelectWriter()
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
	return w.TransBytes()
}

func decodezstd(data []byte) (pbuf.Bytes, error) {
	dec, err := zstd.NewReader(bytes.NewReader(data))
	if err != nil {
		return pbuf.Bytes{}, err
	}
	w := helper.SelectWriter()
	_, err = io.Copy(w, dec)
	dec.Close()
	if err != nil {
		return pbuf.Bytes{}, err
	}
	return w.TransBytes(), nil
}

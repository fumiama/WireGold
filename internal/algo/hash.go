package algo

import (
	"bytes"
	"crypto/md5"
	"encoding/binary"
	"encoding/hex"

	"github.com/fumiama/WireGold/config"
	"github.com/fumiama/blake2b-simd"
	"github.com/sirupsen/logrus"
)

// Blake2bHash8 生成 data 的 blake2b hash, 返回前八位
func Blake2bHash8(precrc64 uint64, data []byte) uint64 {
	var tgt [32]byte
	h := blake2b.New256()
	binary.LittleEndian.PutUint64(tgt[:8], precrc64)
	_, _ = h.Write(tgt[:8])
	_, _ = h.Write(data)
	b := h.Sum(tgt[:0])[:8]
	if config.ShowDebugLog {
		logrus.Debugln("[algo] blk2b hash:", hex.EncodeToString(b))
	}
	return binary.LittleEndian.Uint64(b)
}

// IsVaildBlake2bHash8 在收齐全部分片并解密后验证 packet 合法性
func IsVaildBlake2bHash8(precrc64 uint64, hash8data []byte) bool {
	var tgt [32]byte
	h := blake2b.New256()
	binary.LittleEndian.PutUint64(tgt[:8], precrc64)
	_, _ = h.Write(tgt[:8])
	_, _ = h.Write(hash8data[8:])
	b := h.Sum(tgt[:0])[:8]
	if config.ShowDebugLog {
		logrus.Debugln("[algo] blk2b sum calulated:", hex.EncodeToString(b))
		logrus.Debugln("[algo] blk2b sum in packet:", hex.EncodeToString(hash8data[:8]))
	}
	return bytes.Equal(b, hash8data[:8])
}

// MD5Hash8 calculate packet header checksum
func MD5Hash8(data []byte) uint64 {
	m := md5.Sum(data)
	return binary.LittleEndian.Uint64(m[:8])
}

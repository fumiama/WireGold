package head

import (
	"fmt"
)

const (
	hasmorebit FlagsProto = 0x20 << iota
	nofragbit
	topbit
)

const (
	impossiblebit = hasmorebit | nofragbit
	flagsbit      = topbit | impossiblebit
	protobit      = ^flagsbit
)

type FlagsProto uint8

func (pf FlagsProto) String() string {
	return fmt.Sprintf("%02x", uint8(pf))
}

func (pf FlagsProto) IsValid() bool {
	return pf&topbit == 0 &&
		pf&impossiblebit != impossiblebit &&
		pf.Proto() < ProtoTop
}

func (pf FlagsProto) HasMore() bool {
	return pf&hasmorebit != 0
}

func (pf FlagsProto) NoFrag() bool {
	return pf&nofragbit != 0
}

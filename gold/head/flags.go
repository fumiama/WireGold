package head

import (
	"encoding/binary"
	"fmt"
)

type PacketFlags uint16

func (pf PacketFlags) String() string {
	return fmt.Sprintf("%04x", uint16(pf))
}

func (pf PacketFlags) IsValid() bool {
	return pf&0x8000 == 0
}

func (pf PacketFlags) DontFrag() bool {
	return pf&0x4000 == 0x4000
}

func (pf PacketFlags) NoFrag() bool {
	return pf == 0x4000
}

func (pf PacketFlags) IsSingle() bool {
	return pf == 0
}

func (pf PacketFlags) ZeroOffset() bool {
	return pf&0x1fff == 0
}

func (pf PacketFlags) Offset() uint16 {
	return uint16(pf << 3)
}

// Flags extract flags from raw data
func Flags(data []byte) PacketFlags {
	return PacketFlags(binary.LittleEndian.Uint16(data[10:12]))
}

package head

// Proto 类型定义
const (
	ProtoHello uint8 = iota
	ProtoNotify
	ProtoQuery
	ProtoData
	ProtoTrans
)

const ProtoTop = uint8(protobit + 1)

func (pf FlagsProto) Proto() uint8 {
	return uint8(pf & protobit)
}

type Hello uint8

const (
	HelloPing Hello = iota
	HelloPong
)

// Notify 是 map[peerip]{network, endpoint}
type Notify = map[string][2]string

// Query 是 peerips 组成的数组
type Query = []string

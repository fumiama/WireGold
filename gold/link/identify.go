package link

type Identity struct {
	PubicKey  [32]byte
	EndPoint  string
	KeepAlive int64
}

var peers = make(map[string]*Identity)

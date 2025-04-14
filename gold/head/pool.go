package head

import (
	"time"

	"github.com/fumiama/WireGold/config"
	"github.com/fumiama/WireGold/internal/file"
	"github.com/fumiama/orbyte/pbuf"
	"github.com/sirupsen/logrus"
)

var packetPool = pbuf.NewBufferPool[Packet]()

func init() {
	if config.ShowDebugLog {
		go status()
	}
}

// selectPacket 从池中取出一个 Packet
func selectPacket(buf ...byte) *PacketItem {
	return packetPool.NewBuffer(buf)
}

func status() {
	for range time.NewTicker(time.Minute).C {
		out, in := packetPool.CountItems()
		logrus.Infoln(file.Header(), "packet outside:", out, "inside:", in)
		out, in = pbuf.CountItems()
		logrus.Infoln(file.Header(), "default outside:", out, "inside:", in)
	}
}

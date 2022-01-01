package link

import (
	"encoding/json"
	"net"

	"github.com/fumiama/WireGold/gold/head"
	"github.com/sirupsen/logrus"
)

// 收到通告包的处理函数
func (l *Link) onNotify(packet []byte) {
	// TODO: 完成data解包与endpoint注册
	// 1. Data 解包
	// ---- 使用 head.Notify 解释 packet
	notify := make(head.Notify, 32)
	err := json.Unmarshal(packet, &notify)
	if err != nil {
		logrus.Errorln("[notify] json unmarshal err:", err)
		return
	}
	// 2. endpoint注册
	// ---- 遍历 Notify，注册对方的 endpoint 到
	// ---- connections，注意使用读写锁connmapmu
	for peer, ep := range notify {
		addr, err := net.ResolveUDPAddr("udp", ep)
		if err == nil {
			p, ok := l.me.IsInPeer(peer)
			if ok {
				if p.endpoint.String() != ep {
					p.endpoint = addr
					logrus.Infoln("[notify] set ep of peer", peer, "to", ep)
				}
				continue
			}
		}
		logrus.Debugln("[notify] drop invalid peer:", peer, "ep:", ep)
	}
}

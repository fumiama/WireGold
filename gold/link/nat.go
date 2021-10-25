package link

import (
	"time"

	"github.com/sirupsen/logrus"

	"github.com/fumiama/WireGold/gold/head"
)

// 保持 NAT
func (l *Link) keepAlive() {
	if l.keepalive > 0 && !l.haskeepruning {
		l.haskeepruning = true
		go func() {
			t := time.NewTicker(time.Second * time.Duration(l.keepalive))
			for range t.C {
				_, _ = l.Write(head.NewPacket(head.ProtoHello, 0, 0, nil))
				logrus.Infoln("[link.nat] send keep alive packet")
			}
		}()
		logrus.Infoln("[link.nat] start to keep alive")
	}
}

// 收到询问包的处理函数
func (l *Link) onQuery(packet *head.Packet) {
	// TODO: 完成data解包与notify分发
	// 1. Data 解包
	// ---- 使用 head.Query 解释 packet.Data
	// ---- 根据 Query 确定需要封装的 Notify
	// 2. notify分发
	// ---- 封装 Notify 到 新的 packet.Data
	// ---- 调用 l.Send 发送到对方
}

// 收到通告包的处理函数
func (l *Link) onNotify(packet *head.Packet) {
	// TODO: 完成data解包与endpoint注册
	// 1. Data 解包
	// ---- 使用 head.Notify 解释 packet.Data
	// 2. endpoint注册
	// ---- 遍历 Notify，注册对方的 endpoint 到
	// ---- connections，注意使用读写锁connmapmu
}

package link

import "github.com/fumiama/WireGold/gold/head"

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

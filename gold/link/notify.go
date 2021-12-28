package link

import "github.com/fumiama/WireGold/gold/head"

// 收到通告包的处理函数
func (l *Link) onNotify(packet *head.Packet) {
	// TODO: 完成data解包与endpoint注册
	// 1. Data 解包
	// ---- 使用 head.Notify 解释 packet.Data
	// 2. endpoint注册
	// ---- 遍历 Notify，注册对方的 endpoint 到
	// ---- connections，注意使用读写锁connmapmu
}

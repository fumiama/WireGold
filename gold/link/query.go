package link

import (
	"encoding/json"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/fumiama/WireGold/gold/head"
	"github.com/fumiama/WireGold/helper"
)

// 收到询问包的处理函数
func (l *Link) onQuery(packet []byte) {
	// 完成data解包与notify分发

	// 1. Data 解包
	// ---- 使用 head.Query 解释 packet
	// ---- 根据 Query 确定需要封装的 Notify
	var peers head.Query
	err := json.Unmarshal(packet, &peers)
	if err != nil {
		logrus.Errorln("[qurey] json unmarshal err:", err)
		return
	}

	// 2. notify分发
	// ---- 封装 Notify 到 新的 packet
	// ---- 调用 l.Send 发送到对方
	notify := make(head.Notify, len(peers))
	for _, p := range peers {
		lnk, ok := l.me.IsInPeer(p)
		if ok {
			notify[p] = lnk.endpoint.String()
		}
	}
	if len(notify) > 0 {
		logrus.Infoln("[query] wrap", len(notify), "notify")
		w := helper.SelectWriter()
		json.NewEncoder(w).Encode(&notify)
		l.Write(head.NewPacket(head.ProtoNotify, l.me.srcport, l.peerip, l.me.dstport, w.Bytes()), false)
		helper.PutWriter(w)
	}
}

// sendquery 主动发起查询，询问对方是否可以到达 peers
func (l *Link) sendquery(tick time.Duration, peers ...string) {
	if len(peers) == 0 {
		return
	}
	data, err := json.Marshal(peers)
	if err != nil {
		panic(err)
	}
	t := time.NewTicker(tick)
	for range t.C {
		logrus.Infoln("[query] send query to", l.peerip)
		_, err = l.Write(head.NewPacket(head.ProtoQuery, l.me.srcport, l.peerip, l.me.dstport, data), false)
		if err != nil {
			logrus.Errorln("[query] write err:", err)
		}
	}
}

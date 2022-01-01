package link

import (
	"encoding/json"
	"net"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/fumiama/WireGold/gold/head"
	"github.com/fumiama/WireGold/helper"
)

// 保持 NAT
// dur 决定本机是否定时向 peer 发送 hello 保持 NAT
// 以秒为单位，小于等于 0 不发送
func (l *Link) keepAlive(dur int64) {
	if dur > 0 {
		logrus.Infoln("[link.nat] start to keep alive")
		t := time.NewTicker(time.Second * time.Duration(dur))
		for range t.C {
			n, err := l.Write(head.NewPacket(head.ProtoHello, l.me.srcport, l.peerip, l.me.dstport, nil), false)
			if err == nil {
				logrus.Infoln("[link] send", n, "bytes keep alive packet")
			} else {
				logrus.Errorln("[link] send keep alive packet error:", err)
			}
		}
	}
}

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

package proto

import (
	"encoding/json"

	"github.com/fumiama/orbyte/pbuf"
	"github.com/sirupsen/logrus"

	"github.com/fumiama/WireGold/config"
	"github.com/fumiama/WireGold/gold/head"
	"github.com/fumiama/WireGold/gold/link"
	"github.com/fumiama/WireGold/gold/p2p"

	"github.com/fumiama/WireGold/internal/bin"
	"github.com/fumiama/WireGold/internal/file"
)

func init() {
	link.AddProto(head.ProtoNotify, func(_ *head.Packet, peer *link.Link, data pbuf.Bytes) {
		data.V(func(b []byte) {
			onNotify(peer, b)
		})
	})
	link.AddProto(head.ProtoQuery, func(_ *head.Packet, peer *link.Link, data pbuf.Bytes) {
		data.V(func(b []byte) {
			onQuery(peer, b)
		})
	})
}

// 收到通告包的处理函数
func onNotify(l *link.Link, packet []byte) {
	// TODO: 完成data解包与endpoint注册
	// 1. Data 解包
	// ---- 使用 head.Notify 解释 packet
	notify := make(head.Notify, 32)
	err := json.Unmarshal(packet, &notify)
	if err != nil {
		logrus.Errorln(file.Header(), "notify json unmarshal err:", err)
		return
	}
	// 2. endpoint注册
	// ---- 遍历 Notify，注册对方的 endpoint 到
	// ---- connections，注意使用读写锁connmapmu
	for peer, ep := range notify {
		nw, epstr := ep[0], ep[1]
		if nw != l.Me().EndPoint().Network() {
			logrus.Warnln(file.Header(), "ignore different network notify", nw, "addr", epstr)
			continue
		}
		addr, err := p2p.NewEndPoint(nw, epstr, l.Me().NetworkConfigs()...)
		if err == nil {
			p, ok := l.Me().IsInPeer(peer)
			if ok {
				if bin.IsNilInterface(p.EndPoint()) || !p.EndPoint().Euqal(addr) {
					p.SetEndPoint(addr)
					logrus.Infoln(file.Header(), "notify set ep of peer", peer, "to", ep)
				}
				continue
			}
		}
		if config.ShowDebugLog {
			logrus.Debugln(file.Header(), "notify drop invalid peer:", peer, "ep:", ep)
		}
	}
}

// 收到询问包的处理函数
func onQuery(l *link.Link, packet []byte) {
	// 完成data解包与notify分发

	// 1. Data 解包
	// ---- 使用 head.Query 解释 packet
	// ---- 根据 Query 确定需要封装的 Notify
	var peers head.Query
	err := json.Unmarshal(packet, &peers)
	if err != nil {
		logrus.Errorln(file.Header(), "query json unmarshal err:", err)
		return
	}

	if l == nil || l.Me() == nil {
		logrus.Errorln(file.Header(), "nil link/me")
		return
	}

	// 2. notify分发
	// ---- 封装 Notify 到 新的 packet
	// ---- 调用 l.Send 发送到对方
	notify := make(head.Notify, len(peers))
	for _, p := range peers {
		lnk, ok := l.Me().IsInPeer(p)
		eps := ""
		if l.Me().EndPoint().Network() == "udp" { // udp has real p2p
			if bin.IsNilInterface(lnk.EndPoint()) {
				continue
			}
			eps = lnk.EndPoint().String()
		}
		if eps == "" {
			eps = l.RawEndPoint() // use registered ep only
		}
		if eps == "" {
			continue
		}
		if ok && bin.IsNonNilInterface(lnk.EndPoint()) {
			notify[p] = [2]string{
				lnk.EndPoint().Network(),
				eps,
			}
		}
	}
	if len(notify) > 0 {
		logrus.Infoln(file.Header(), "query wrap", len(notify), "notify")
		w := bin.SelectWriter()
		_ = json.NewEncoder(w).Encode(&notify)
		w.P(func(b *pbuf.Buffer) {
			l.WritePacket(head.ProtoNotify, b.Bytes())
		})
	}
}

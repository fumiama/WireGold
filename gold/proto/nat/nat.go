package nat

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
	// 收到通告包的处理
	link.RegisterDispacher(head.ProtoNotify, func(_ *head.Packet, peer *link.Link, data pbuf.Bytes) {
		data.V(func(b []byte) {
			// 1. Data 解包
			// ---- 使用 head.Notify 解释 packet
			notify := make(head.Notify, 32)
			err := json.Unmarshal(b, &notify)
			if err != nil {
				logrus.Errorln(file.Header(), "notify json unmarshal err:", err)
				return
			}
			// 2. endpoint注册
			// ---- 遍历 Notify，注册对方的 endpoint 到
			// ---- connections，注意使用读写锁connmapmu
			for ps, ep := range notify {
				nw, epstr := ep[0], ep[1]
				if nw != peer.Me().EndPoint().Network() {
					logrus.Warnln(file.Header(), "ignore different network notify", nw, "addr", epstr)
					continue
				}
				addr, err := p2p.NewEndPoint(nw, epstr, peer.Me().NetworkConfigs()...)
				if err == nil {
					p, ok := peer.Me().IsInPeer(ps)
					if ok {
						if bin.IsNilInterface(p.EndPoint()) || !p.EndPoint().Euqal(addr) {
							p.SetEndPoint(addr)
							logrus.Infoln(file.Header(), "notify set ep of peer", ps, "to", ep)
						}
						continue
					}
				}
				if config.ShowDebugLog {
					logrus.Debugln(file.Header(), "notify drop invalid peer:", ps, "ep:", ep)
				}
			}
		})
	})
	// 收到询问包的处理
	link.RegisterDispacher(head.ProtoQuery, func(_ *head.Packet, peer *link.Link, data pbuf.Bytes) {
		data.V(func(b []byte) {
			// 完成data解包与notify分发

			// 1. Data 解包
			// ---- 使用 head.Query 解释 packet
			// ---- 根据 Query 确定需要封装的 Notify
			var peers head.Query
			err := json.Unmarshal(b, &peers)
			if err != nil {
				logrus.Errorln(file.Header(), "query json unmarshal err:", err)
				return
			}

			if peer == nil || peer.Me() == nil {
				logrus.Errorln(file.Header(), "nil link/me")
				return
			}

			// 2. notify分发
			// ---- 封装 Notify 到 新的 packet
			// ---- 发送到对方
			notify := make(head.Notify, len(peers))
			for _, p := range peers {
				lnk, ok := peer.Me().IsInPeer(p)
				eps := ""
				if peer.Me().EndPoint().Network() == "udp" { // udp has real p2p
					if bin.IsNilInterface(lnk.EndPoint()) {
						continue
					}
					eps = lnk.EndPoint().String()
				}
				if eps == "" {
					eps = peer.RawEndPoint() // use registered ep only
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
					peer.WritePacket(head.ProtoNotify, b.Bytes(), peer.Me().TTL())
				})
			}
		})
	})
}

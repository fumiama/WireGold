package link

import (
	"encoding/json"
	"reflect"
	"sync/atomic"
	"time"
	"unsafe"

	"github.com/sirupsen/logrus"

	"github.com/fumiama/WireGold/config"
	"github.com/fumiama/WireGold/gold/head"
	"github.com/fumiama/WireGold/gold/p2p"
	"github.com/fumiama/WireGold/helper"
)

// 保持 NAT
// dur 决定本机是否定时向 peer 发送 hello 保持 NAT
// 以秒为单位，小于等于 0 不发送
func (l *Link) keepAlive(dur int64) {
	if dur > 0 {
		logrus.Infoln("[nat] start to keep alive")
		t := time.NewTicker(time.Second * time.Duration(dur))
		for range t.C {
			if l.me.connections == nil {
				return
			}
			la := (*time.Time)(atomic.LoadPointer((*unsafe.Pointer)(unsafe.Pointer(&l.lastalive))))
			if la != nil && time.Since(*la) > 10*time.Second*time.Duration(dur) { // 可能已经被阻断， 断开重连
				logrus.Warnln("[nat] no response after 10 keep alive tries, re-connecting...")
				err := l.me.Restart()
				if err != nil {
					logrus.Errorln("[nat] re-connect me err:", err)
				} else {
					logrus.Infoln("[nat] re-connect me succeeded")
				}
			}
			n, err := l.WriteAndPut(head.NewPacket(head.ProtoHello, l.me.srcport, l.peerip, l.me.dstport, nil), false)
			if err == nil {
				logrus.Infoln("[nat] send", n, "bytes keep alive packet")
			} else {
				logrus.Warnln("[nat] send keep alive packet error:", err)
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
		logrus.Errorln("[nat] notify json unmarshal err:", err)
		return
	}
	// 2. endpoint注册
	// ---- 遍历 Notify，注册对方的 endpoint 到
	// ---- connections，注意使用读写锁connmapmu
	for peer, ep := range notify {
		nw, epstr := ep[0], ep[1]
		if nw != l.me.ep.Network() {
			logrus.Warnln("[nat] ignore different network notify", nw, "addr", epstr)
			continue
		}
		addr, err := p2p.NewEndPoint(nw, epstr, l.me.networkconfigs...)
		if err == nil {
			p, ok := l.me.IsInPeer(peer)
			if ok {
				if reflect.ValueOf(p.endpoint).IsZero() || !p.endpoint.Euqal(addr) {
					p.endpoint = addr
					logrus.Infoln("[nat] notify set ep of peer", peer, "to", ep)
				}
				continue
			}
		}
		if config.ShowDebugLog {
			logrus.Debugln("[nat] notify drop invalid peer:", peer, "ep:", ep)
		}
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
		logrus.Errorln("[nat] query json unmarshal err:", err)
		return
	}

	if l == nil || l.me == nil {
		logrus.Errorln("[nat] nil link/me")
		return
	}

	// 2. notify分发
	// ---- 封装 Notify 到 新的 packet
	// ---- 调用 l.Send 发送到对方
	notify := make(head.Notify, len(peers))
	for _, p := range peers {
		lnk, ok := l.me.IsInPeer(p)
		eps := ""
		if l.me.ep.Network() == "udp" { // udp has real p2p
			if reflect.ValueOf(lnk.endpoint).IsZero() {
				continue
			}
			eps = lnk.endpoint.String()
		}
		if eps == "" {
			eps = l.rawep // use registered ep only
		}
		if eps == "" {
			continue
		}
		if ok && !reflect.ValueOf(lnk.endpoint).IsZero() {
			notify[p] = [2]string{
				lnk.endpoint.Network(),
				eps,
			}
		}
	}
	if len(notify) > 0 {
		logrus.Infoln("[nat] query wrap", len(notify), "notify")
		w := helper.SelectWriter()
		_ = json.NewEncoder(w).Encode(&notify)
		_, err = l.WriteAndPut(head.NewPacket(head.ProtoNotify, l.me.srcport, l.peerip, l.me.dstport, w.Bytes()), false)
		if err != nil {
			logrus.Errorln("[nat] notify peer", l, "err:", err)
			return
		}
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
		logrus.Infoln("[nat] query send query to", l.peerip)
		_, err = l.WriteAndPut(head.NewPacket(head.ProtoQuery, l.me.srcport, l.peerip, l.me.dstport, data), false)
		if err != nil {
			logrus.Errorln("[nat] query write err:", err)
		}
	}
}

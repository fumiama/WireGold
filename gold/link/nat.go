package link

import (
	"encoding/json"
	"sync/atomic"
	"time"
	"unsafe"

	"github.com/sirupsen/logrus"

	"github.com/fumiama/WireGold/gold/head"
	"github.com/fumiama/WireGold/internal/file"
)

// 保持 NAT
// dur 决定本机是否定时向 peer 发送 hello 保持 NAT
// 以秒为单位，小于等于 0 不发送
func (l *Link) keepAlive(dur int64) {
	if dur > 0 {
		logrus.Infoln(file.Header(), "start to keep alive")
		t := time.NewTicker(time.Second * time.Duration(dur))
		for range t.C {
			if l.me.connections == nil {
				return
			}
			la := (*time.Time)(atomic.LoadPointer((*unsafe.Pointer)(unsafe.Pointer(&l.lastalive))))
			if la != nil && time.Since(*la) > 10*time.Second*time.Duration(dur) { // 可能已经被阻断， 断开重连
				logrus.Warnln(file.Header(), "no response after 10 keep alive tries, re-connecting...")
				err := l.me.Restart()
				if err != nil {
					logrus.Errorln(file.Header(), "re-connect me err:", err)
				} else {
					logrus.Infoln(file.Header(), "re-connect me succeeded")
				}
			}
			l.WritePacket(head.ProtoHello, []byte{byte(head.HelloPing)})
			logrus.Infoln(file.Header(), "send keep alive to", l.peerip)
		}
	}
}

// sendquery 主动发起查询，询问对方是否可以到达 peers
func (l *Link) sendQuery(tick time.Duration, peers ...string) {
	if len(peers) == 0 {
		return
	}
	data, err := json.Marshal(peers)
	if err != nil {
		panic(err)
	}
	t := time.NewTicker(tick)
	for range t.C {
		l.WritePacket(head.ProtoQuery, data)
		logrus.Infoln(file.Header(), "send query to", l.peerip)
	}
}

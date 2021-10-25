package link

import (
	"net"
	"sync"
)

var (
	routetable   = make(map[string][]*Link)
	routetablemu sync.RWMutex
)

// Accept 判断是否应当接受 ip 发来的包
func (l *Link) Accept(ip net.IP) bool {
	for _, cidr := range l.allowedips {
		if cidr.Contains(ip) {
			return true
		}
	}
	return false
}

// IsToMe 判断是否是发给自己的包
func (l *Link) IsToMe(ip net.IP) bool {
	return ip.Equal(me)
}

// NextHop 得到前往 ip 的下一跳的 link
func (l *Link) NextHop(ip net.IP) *Link {
	// TODO: 遍历 routetable，得到正确的下一跳
	// 注意使用 routetablemu 读写锁避免竞争
	return l
}

package link

import (
	"encoding/binary"
	"net"
	"sync"

	"github.com/sirupsen/logrus"
)

type Router struct {
	// map[cidr]*Link
	table map[string]*Link
	mu    sync.RWMutex
	list  []*net.IPNet
}

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
	return ip.Equal(l.me.me) || ip.Equal(net.IPv4bcast) || isSubnetBcast(ip, &l.me.subnet)
}

// SetDefault 设置默认网关
func (r *Router) SetDefault(l *Link) {
	defnet := &net.IPNet{IP: net.IPv4(0, 0, 0, 0), Mask: net.IPv4Mask(0, 0, 0, 0)}
	r.mu.Lock()
	r.list[len(r.list)-1] = defnet
	r.table[defnet.String()] = l
	r.mu.Unlock()
}

// NextHop 得到前往 ip 的下一跳的 link
func (r *Router) NextHop(cidr string) *Link {
	logrus.Infoln("[router] search for cidr", cidr)
	// TODO: 遍历 r.table，得到正确的下一跳
	// 注意使用 r.mu 读写锁避免竞争
	return r.table[cidr]
}

// SetItem 添加一条表项
func (r *Router) SetItem(ip *net.IPNet, l *Link) {
	r.mu.Lock()
	// 从第一条表项开始匹配
	for i := 0; i < len(r.list); i++ {
		if r.list[i].Contains(ip.IP) {
			// 是同一个网络
			if ip.Mask.String() == r.list[i].Mask.String() {
				logrus.Infoln("[router] change link of item", r.list[i], "from", r.table[r.list[i].String()], "to", l)
				r.table[r.list[i].String()] = l
				break
			}
			// 是新网络
			r.list = append(r.list, nil)
			copy(r.list[i+1:], r.list[i:len(r.list)-1])
			r.list[i] = ip
			r.table[ip.String()] = l
			logrus.Infoln("[router] add item: net =", ip, "link =", l)
			break
		}
	}
	r.mu.Unlock()
}

func isSubnetBcast(ip net.IP, subnet *net.IPNet) bool {
	if !subnet.Contains(ip) {
		return false
	}
	maskr := make(net.IPMask, 4)
	binary.LittleEndian.PutUint32(maskr[:], ^binary.LittleEndian.Uint32(subnet.Mask))
	return ip.Mask(maskr).Equal(net.IP(maskr))
}

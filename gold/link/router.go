package link

import "net"

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
	return l
}

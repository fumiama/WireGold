package link

import "net"

func (l *Link) Accept(ip net.IP) bool {
	for _, cidr := range l.allowedips {
		if cidr.Contains(ip) {
			return true
		}
	}
	return false
}

func NextHop(ip net.IP) *Link {
	return nil
}

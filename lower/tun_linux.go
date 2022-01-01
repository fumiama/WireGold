//go:build linux
// +build linux

package lower

func (n *NIC) Up() {
	execute("/sbin/ip", "link", "set", "dev", n.ifce.Name(), "mtu", n.mtu)
	execute("/sbin/ip", "addr", "add", n.ip, "dev", n.ifce.Name())
	execute("/sbin/ip", "link", "set", "dev", n.ifce.Name(), "up")
	execute("/sbin/ip", "route", "add", n.subnet, "dev", n.ifce.Name())
	for _, c := range n.cidrs {
		execute("/sbin/ip", "route", "add", c, "dev", n.ifce.Name())
	}
}

func (n *NIC) Down() {
	execute("/sbin/ip", "link", "set", "dev", n.ifce.Name(), "down")
	execute("/sbin/ip", "route", "del", n.subnet, "dev", n.ifce.Name())
	for _, c := range n.cidrs {
		execute("/sbin/ip", "route", "del", c, "dev", n.ifce.Name())
	}
}

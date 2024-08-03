//go:build linux
// +build linux

package lower

func (n *NICIO) Up() {
	execute("/sbin/ip", "link", "set", "dev", n.ifce.Name(), "mtu", n.mtu)
	execute("/sbin/ip", "addr", "add", n.rawipnet, "dev", n.ifce.Name())
	execute("/sbin/ip", "link", "set", "dev", n.ifce.Name(), "up")
	for _, c := range n.cidrs {
		execute("/sbin/ip", "route", "add", c, "dev", n.ifce.Name())
	}
}

func (n *NICIO) Down() {
	for _, c := range n.cidrs {
		execute("/sbin/ip", "route", "del", c, "dev", n.ifce.Name())
	}
	execute("/sbin/ip", "link", "set", "dev", n.ifce.Name(), "down")
}

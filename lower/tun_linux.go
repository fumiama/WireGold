//go:build linux
// +build linux

package lower

func (n *NIC) prepare() {
	execute("/sbin/ip", "link", "set", "dev", n.ifce.Name(), "mtu", "1500")
	execute("/sbin/ip", "addr", "add", n.ip, "dev", n.ifce.Name())
	execute("/sbin/ip", "link", "set", "dev", n.ifce.Name(), "up")
	execute("/sbin/ip", "route", "add", n.subnet, "dev", n.ifce.Name())
}

func (n *NIC) Up() {
	execute("/sbin/ip", "link", "set", "dev", n.ifce.Name(), "up")
}

func (n *NIC) Down() {
	execute("/sbin/ip", "link", "set", "dev", n.ifce.Name(), "down")
}

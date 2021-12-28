//go:build linux
// +build linux

package lower

func (n *NIC) prepare() {
	execute("/sbin/ip", "link", "set", "dev", ifcename, "mtu", "1500")
	execute("/sbin/ip", "addr", "add", ip, "dev", ifcename)
	execute("/sbin/ip", "route", "add", subnet, "dev", ifcename)
}

func (n *NIC) Up() {
	execute("/sbin/ip", "link", "set", "dev", ifcename, "up")
}

func (n *NIC) Down() {
	execute("/sbin/ip", "link", "set", "dev", ifcename, "down")
}

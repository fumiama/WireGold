//go:build darwin
// +build darwin

package lower

func (n *NIC) Up() {
	execute("ifconfig", n.ifce.Name(), "mtu", n.mtu)
	execute("ifconfig", n.ifce.Name(), "inet", n.ip, n.ip, "up")
	execute("route", "add", n.subnet, "-interface", n.ifce.Name())
	for _, c := range n.cidrs {
		execute("route", "add", c, "-interface", n.ifce.Name())
	}
}

func (n *NIC) Down() {
	execute("ifconfig", n.ifce.Name(), "down")
	execute("route", "delete", n.subnet, "-interface", n.ifce.Name())
	for _, c := range n.cidrs {
		execute("route", "delete", c, "-interface", n.ifce.Name())
	}
}

//go:build darwin
// +build darwin

package lower

func (n *NIC) prepare() {
	execute("ifconfig", n.ifce.Name(), "inet", n.ip, n.ip, "up")
	execute("route", "add", n.subnet, "-interface", n.ifce.Name())
}

func (n *NIC) Up() {
	execute("ifconfig", n.ifce.Name(), "inet", n.ip, n.ip, "up")
}

func (n *NIC) Down() {
	execute("ifconfig", n.ifce.Name(), "down")
}

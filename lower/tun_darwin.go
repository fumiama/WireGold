//go:build darwin
// +build darwin

package lower

import "net"

func (n *NIC) Up() {
	execute("ifconfig", n.ifce.Name(), "mtu", n.mtu) // max: 9159
	execute(
		"ifconfig", n.ifce.Name(),
		"inet", n.ip.String(), n.ip.String(),
		"netmask", (net.IP)(n.subnet.Mask).String(),
		"up",
	)
	execute("route", "add", n.subnet.String(), "-interface", n.ifce.Name())
	for _, c := range n.cidrs {
		execute("route", "add", c, "-interface", n.ifce.Name())
	}
}

func (n *NIC) Down() {
	execute("route", "delete", n.subnet.String(), "-interface", n.ifce.Name())
	for _, c := range n.cidrs {
		execute("route", "delete", c, "-interface", n.ifce.Name())
	}
	execute("ifconfig", n.ifce.Name(), "down")
}

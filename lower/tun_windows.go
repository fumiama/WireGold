//go:build windows
// +build windows

package lower

import "net"

func (n *NIC) Up() {
	// execute("netsh", "interface", "set", "interface", n.ifce.Name(), "enabled")
	_, ipn, err := net.ParseCIDR(n.subnet)
	if err != nil {
		panic(err)
	}
	execute("cmd", "/c", "netsh interface ip set address name=\""+n.ifce.Name()+"\" source=static addr=\""+n.ip+"\" mask=\""+(net.IP)(ipn.Mask).String()+"\" gateway=none mtu="+n.mtu)
	for _, c := range n.cidrs {
		ip, cidr, err := net.ParseCIDR(c)
		if err != nil {
			panic(err)
		}
		execute("cmd", "/c", "route ADD "+ip.String()+" MASK "+(net.IP)(cidr.Mask).String()+" "+n.ip)
	}
}

func (n *NIC) Down() {
	// execute("netsh", "interface", "set", "interface", n.ifce.Name(), "disabled")
	for _, c := range n.cidrs {
		ip, _, err := net.ParseCIDR(c)
		if err != nil {
			panic(err)
		}
		execute("cmd", "/c", "route DELETE "+ip.String())
	}
}

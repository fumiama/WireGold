//go:build windows
// +build windows

package lower

import "net"

func (n *NIC) Up() {
	execute("cmd", "/c", "netsh interface ip set address name=\""+n.ifce.Name()+"\" source=static addr=\""+n.ip.String()+"\" mask=\""+(net.IP)(n.subnet.Mask).String()+"\" gateway=none")
	execute("cmd", "/c", "netsh interface ipv4 set subinterface \""+n.ifce.Name()+"\" mtu="+n.mtu)
	for _, c := range n.cidrs {
		ip, cidr, err := net.ParseCIDR(c)
		if err != nil {
			panic(err)
		}
		execute("cmd", "/c", "route ADD "+ip.String()+" MASK "+(net.IP)(cidr.Mask).String()+" "+n.ip.String())
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

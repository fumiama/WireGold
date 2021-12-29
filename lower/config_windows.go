//go:build windows
// +build windows

package lower

import "github.com/songgao/water"

var tuncfg = water.Config{
	DeviceType: water.TUN,
	PlatformSpecificParams: water.PlatformSpecificParams{
		ComponentID:   "root\\tap0901",
		InterfaceName: "OpenVPN TAP-Windows6",
		Network:       "192.168.233.0/24",
	},
}

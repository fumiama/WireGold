//go:build !windows
// +build !windows

package lower

import "github.com/songgao/water"

var tuncfg = water.Config{DeviceType: water.TUN}

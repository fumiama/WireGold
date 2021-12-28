//go:build !darwin && !linux && !windows
// +build !darwin,!linux,!windows

package lower

func (n *NIC) prepare() {
	panic("not support this os now")
}

func (n *NIC) Up() {
	panic("not support this os now")
}

func (n *NIC) Down() {
	panic("not support this os now")
}

//go:build !darwin && !linux && !windows
// +build !darwin,!linux,!windows

package lower

func (n *NIC) Up() {
	panic("not support lower on this os now")
}

func (n *NIC) Down() {
	panic("not support lower on this os now")
}

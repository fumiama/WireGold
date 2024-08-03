//go:build !darwin && !linux && !windows
// +build !darwin,!linux,!windows

package lower

func (n *NICIO) Up() {
	panic("not support lower on this os now")
}

func (n *NICIO) Down() {
	panic("not support lower on this os now")
}

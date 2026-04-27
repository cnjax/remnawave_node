//go:build !linux

package connkill

import "errors"

// DropByIPs is not supported on non-Linux platforms.
func DropByIPs(ips []string) error {
	return errors.New("SOCK_DESTROY is only supported on Linux")
}

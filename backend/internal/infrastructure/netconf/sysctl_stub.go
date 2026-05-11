//go:build !linux

package netconf

func EnableForwarding() error {
	return nil
}

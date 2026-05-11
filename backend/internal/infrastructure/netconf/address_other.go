//go:build !linux

package netconf

import "fmt"

func AssignTUN(tunName, cidr string) error {
	return fmt.Errorf("AssignTUN not supported on this platform")
}

//go:build !linux

package netconf

func SetupReturnRoute(fwmark int, routingTable int) error { return nil }

func TeardownReturnRoute(fwmark int, routingTable int) error { return nil }

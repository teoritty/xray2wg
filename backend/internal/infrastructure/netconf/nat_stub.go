//go:build !linux

package netconf

func SetupNATMasquerade(string, string) error { return nil }

func TeardownNATMasquerade(string, string) error { return nil }

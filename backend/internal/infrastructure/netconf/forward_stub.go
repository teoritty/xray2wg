//go:build !linux

package netconf

func SetupForwardRules(tunName string) error {
	return nil
}

func TeardownForwardRules(tunName string) error {
	return nil
}

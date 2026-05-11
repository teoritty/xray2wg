//go:build !linux

package netconf

func SetupTProxy(tunnelID int, tunName string, xrayPort int, fwmark int, localGatewayIP string, vlessFlow string) error {
	return nil
}

func TeardownTProxy(tunnelID int, tunName string, fwmark int) error {
	return nil
}

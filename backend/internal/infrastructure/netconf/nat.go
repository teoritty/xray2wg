package netconf

import (
	"net"
	"strings"
)

// ClientSubnetCIDR returns the network CIDR for SNAT (e.g. 10.100.1.0/24 from 10.100.1.1/24).
func ClientSubnetCIDR(wgAddrHostCIDR string) (string, error) {
	_, n, err := net.ParseCIDR(strings.TrimSpace(wgAddrHostCIDR))
	if err != nil {
		return "", err
	}
	return n.String(), nil
}

// WgGatewayHost returns the interface host IP from a WG server CIDR like "10.100.1.1/24".
func WgGatewayHost(wgAddrHostCIDR string) (string, error) {
	ip, _, err := net.ParseCIDR(strings.TrimSpace(wgAddrHostCIDR))
	if err != nil {
		return "", err
	}
	if ip4 := ip.To4(); ip4 != nil {
		return ip4.String(), nil
	}
	return ip.String(), nil
}

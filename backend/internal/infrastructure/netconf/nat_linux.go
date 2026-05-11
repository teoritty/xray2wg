//go:build linux

package netconf

import (
	"github.com/coreos/go-iptables/iptables"
	"github.com/rs/zerolog/log"
)

const tableNat = "nat"

// SetupNATMasquerade SNATs traffic from the WG client subnet when it leaves via any interface
// other than the tunnel (ICMP, hairpins, etc.). TPROXY→Xray TCP/UDP is not sourced from 10.100.x on the VLESS dial.
func SetupNATMasquerade(tunName, clientSubnetCIDR string) error {
	if clientSubnetCIDR == "" {
		return nil
	}
	log.Info().Str("tun", tunName).Str("subnet", clientSubnetCIDR).Msg("tunnel_trace nat: SetupNATMasquerade begin")
	tb, err := iptables.New(iptables.IPFamily(iptables.ProtocolIPv4))
	if err != nil {
		log.Error().Err(err).Msg("tunnel_trace nat: iptables.New failed")
		return err
	}
	_ = TeardownNATMasquerade(tunName, clientSubnetCIDR)
	rule := []string{"-s", clientSubnetCIDR, "!", "-o", tunName, "-j", "MASQUERADE"}
	if err := tb.Append(tableNat, "POSTROUTING", rule...); err != nil {
		log.Error().Err(err).Msg("tunnel_trace nat: POSTROUTING MASQUERADE append failed")
		return err
	}
	log.Info().Msg("tunnel_trace nat: MASQUERADE installed for WG subnet egress (not via tun)")
	return nil
}

func TeardownNATMasquerade(tunName, clientSubnetCIDR string) error {
	if clientSubnetCIDR == "" {
		return nil
	}
	tb, err := iptables.New(iptables.IPFamily(iptables.ProtocolIPv4))
	if err != nil {
		return err
	}
	rule := []string{"-s", clientSubnetCIDR, "!", "-o", tunName, "-j", "MASQUERADE"}
	_ = tb.Delete(tableNat, "POSTROUTING", rule...)
	return nil
}

//go:build linux

package netconf

import (
	"github.com/coreos/go-iptables/iptables"
	"github.com/rs/zerolog/log"
)

const tableNat = "nat"

// SetupNATMasquerade SNATs traffic from the WG client subnet when it leaves via any interface
// other than the tunnel. Two rules are installed:
//   - A general rule for TCP/UDP (protocol-unspecified) with the subnet guard.
//   - An explicit ICMP rule without the interface guard: TPROXY-active kernels sometimes
//     mis-track ICMP flows that bypass the TPROXY mark path, causing the general rule to miss
//     them. The dedicated ICMP rule is the fix proposed in github.com/teoritty/xray2wg/issues/2.
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
	log.Info().Strs("rule", rule).Msg("tunnel_trace nat: MASQUERADE installed for WG subnet egress (not via tun)")

	// Explicit ICMP rule: no interface guard so the masquerade fires even when the kernel's
	// conntrack does not associate the ICMP flow with the tunnel mark.
	icmpRule := []string{"-s", clientSubnetCIDR, "-p", "icmp", "-j", "MASQUERADE"}
	if err := tb.Append(tableNat, "POSTROUTING", icmpRule...); err != nil {
		log.Error().Err(err).Msg("tunnel_trace nat: POSTROUTING ICMP MASQUERADE append failed")
		return err
	}
	log.Info().Strs("rule", icmpRule).Msg("tunnel_trace nat: ICMP MASQUERADE installed for WG subnet")

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
	_ = tb.Delete(tableNat, "POSTROUTING", "-s", clientSubnetCIDR, "!", "-o", tunName, "-j", "MASQUERADE")
	_ = tb.Delete(tableNat, "POSTROUTING", "-s", clientSubnetCIDR, "-p", "icmp", "-j", "MASQUERADE")
	return nil
}

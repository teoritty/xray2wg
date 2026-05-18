//go:build linux

package netconf

import (
	"github.com/coreos/go-iptables/iptables"
	"github.com/rs/zerolog/log"
)

const tableNat = "nat"

// SetupNATMasquerade SNATs traffic from the WG client subnet when it leaves via any interface
// other than the tunnel itself. One protocol-agnostic rule covers TCP/UDP, ICMP, GRE, ESP and
// anything else — for TCP/UDP the rule never matches in practice because TPROXY consumes those
// packets before POSTROUTING; for non-TCP/UDP (ICMP, PMTUD, traceroute) it is the SNAT path
// that makes ping work out of the WG namespace (issue #2).
//
// The rule is inserted at position 1 so it sits above Docker's MASQUERADE rules. With Append,
// Docker's earlier rules can match first on hosts that publish containers and silently swallow
// the SNAT decision.
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
	if err := tb.Insert(tableNat, "POSTROUTING", 1, rule...); err != nil {
		log.Error().Err(err).Msg("tunnel_trace nat: POSTROUTING MASQUERADE insert failed")
		return err
	}
	log.Info().Strs("rule", rule).Msg("tunnel_trace nat: MASQUERADE installed at POSTROUTING pos=1 for WG subnet egress (not via tun)")

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
	// Older builds installed a dedicated ICMP MASQUERADE rule; delete it best-effort so a
	// rolling upgrade does not leave a stale rule behind.
	_ = tb.Delete(tableNat, "POSTROUTING", "-s", clientSubnetCIDR, "-p", "icmp", "-j", "MASQUERADE")
	return nil
}

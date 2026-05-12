//go:build linux

package netconf

import (
	"fmt"
	"strconv"
	"strings"

	"xray2wg/backend/internal/tracenv"

	"github.com/coreos/go-iptables/iptables"
	"github.com/rs/zerolog/log"
)

const tableMangle = "mangle"

// SetupTProxy steers tun ingress into Xray TPROXY. localGatewayIP must be the WG server address on that tun
// (e.g. 10.100.2.1): traffic to it on port 53 is excluded so Xray's dns-in can bind there; without this,
// TPROXY sends udp:10.100.x.1:53 to transparent-in and it wrongly goes out via VLESS.
//
// vlessFlow: when it is XTLS Vision without vision-udp443, UDP/443 (QUIC) is dropped before TPROXY so the
// client retries on TCP/443 instead of hitting "XTLS rejected UDP/443 traffic" on the outbound.
func SetupTProxy(tunnelID int, tunName string, xrayPort int, fwmark int, localGatewayIP string, vlessFlow string) error {
	log.Info().
		Int("tunnel_id", tunnelID).
		Str("tun", tunName).
		Int("xray_port", xrayPort).
		Int("fwmark", fwmark).
		Str("local_gateway", strings.TrimSpace(localGatewayIP)).
		Str("vless_flow", strings.TrimSpace(vlessFlow)).
		Bool("drop_quic_udp443", dropQUICVision(vlessFlow)).
		Msg("tunnel_trace tproxy: begin SetupTProxy")

	tb, err := iptables.New(iptables.IPFamily(iptables.ProtocolIPv4))
	if err != nil {
		log.Error().Err(err).Msg("tunnel_trace tproxy: iptables.New failed")
		return err
	}
	chain := fmt.Sprintf("XRAY2WG_%d", tunnelID)

	exists, _ := tb.ChainExists(tableMangle, chain)
	if exists {
		log.Info().Str("chain", chain).Msg("tunnel_trace tproxy: clear existing chain")
		_ = tb.ClearChain(tableMangle, chain)
	} else if err := tb.NewChain(tableMangle, chain); err != nil {
		log.Error().Str("chain", chain).Err(err).Msg("tunnel_trace tproxy: NewChain failed")
		return err
	}

	_ = tb.Delete(tableMangle, "PREROUTING", "-i", tunName, "-j", chain)

	mark := strconv.Itoa(fwmark)
	port := strconv.Itoa(xrayPort)
	if g := strings.TrimSpace(localGatewayIP); g != "" {
		skipUDP := []string{"-d", g, "-p", "udp", "-m", "udp", "--dport", "53", "-j", "RETURN"}
		skipTCP := []string{"-d", g, "-p", "tcp", "-m", "tcp", "--dport", "53", "-j", "RETURN"}
		log.Info().Str("chain", chain).Strs("rule_skip_dns", skipUDP).Msg("tunnel_trace tproxy: append RETURN (gateway DNS not via TPROXY)")
		if err := tb.Append(tableMangle, chain, skipUDP...); err != nil {
			log.Error().Err(err).Msg("tunnel_trace tproxy: append RETURN udp/53 failed")
			return err
		}
		if err := tb.Append(tableMangle, chain, skipTCP...); err != nil {
			log.Error().Err(err).Msg("tunnel_trace tproxy: append RETURN tcp/53 failed")
			return err
		}
	}
	// ICMP cannot be proxied via TPROXY/xray-core; let it pass through so ping and Path MTU Discovery work.
	// The existing MASQUERADE rule in nat_linux.go SNATs it out via the host's default interface.
	icmp := []string{"-p", "icmp", "-j", "RETURN"}
	log.Info().Str("chain", chain).Strs("rule_icmp", icmp).Msg("tunnel_trace tproxy: append ICMP RETURN (bypass TPROXY)")
	if err := tb.Append(tableMangle, chain, icmp...); err != nil {
		log.Error().Err(err).Msg("tunnel_trace tproxy: append ICMP RETURN failed")
		return err
	}
	// Shrink TCP MSS before TPROXY so WG+VLESS path does not black-hole large TLS segments.
	mss := []string{"-p", "tcp", "-m", "tcp", "--tcp-flags", "SYN,RST", "SYN", "-j", "TCPMSS", "--set-mss", "1360"}
	tcp := []string{"-p", "tcp", "-j", "TPROXY", "--on-port", port, "--tproxy-mark", mark}
	udp := []string{"-p", "udp", "-j", "TPROXY", "--on-port", port, "--tproxy-mark", mark}
	log.Info().Str("chain", chain).Strs("rule_mss", mss).Msg("tunnel_trace tproxy: append TCPMSS (before TPROXY)")
	if err := tb.Append(tableMangle, chain, mss...); err != nil {
		log.Error().Err(err).Msg("tunnel_trace tproxy: append TCPMSS failed")
		return err
	}
	log.Info().Str("chain", chain).Strs("rule_tcp", tcp).Msg("tunnel_trace tproxy: append TCP TPROXY")
	if err := tb.Append(tableMangle, chain, tcp...); err != nil {
		log.Error().Err(err).Msg("tunnel_trace tproxy: append TCP failed")
		return err
	}
	if dropQUICVision(vlessFlow) {
		quic := []string{"-p", "udp", "-m", "udp", "--dport", "443", "-j", "DROP"}
		log.Info().Str("chain", chain).Strs("rule_drop_quic", quic).Msg("tunnel_trace tproxy: append DROP udp/443 (Vision without vision-udp443)")
		if err := tb.Append(tableMangle, chain, quic...); err != nil {
			log.Error().Err(err).Msg("tunnel_trace tproxy: append DROP udp/443 failed")
			return err
		}
	}
	log.Info().Strs("rule_udp", udp).Msg("tunnel_trace tproxy: append UDP TPROXY")
	if err := tb.Append(tableMangle, chain, udp...); err != nil {
		log.Error().Err(err).Msg("tunnel_trace tproxy: append UDP failed")
		return err
	}
	log.Info().Str("tun", tunName).Str("chain", chain).Msg("tunnel_trace tproxy: jump PREROUTING -> chain")
	if err := tb.Append(tableMangle, "PREROUTING", "-i", tunName, "-j", chain); err != nil {
		log.Error().Err(err).Msg("tunnel_trace tproxy: PREROUTING jump failed")
		return err
	}

	if tracenv.TunnelTrace() {
		if lines, err := tb.List(tableMangle, chain); err == nil {
			log.Info().Str("chain", chain).Str("dump", strings.Join(lines, " | ")).Msg("tunnel_trace tproxy: mangle chain dump")
		} else {
			log.Warn().Err(err).Msg("tunnel_trace tproxy: List chain failed")
		}
		if lines, err := tb.List(tableMangle, "PREROUTING"); err == nil {
			var pr []string
			for _, ln := range lines {
				if strings.Contains(ln, chain) || strings.Contains(ln, tunName) {
					pr = append(pr, ln)
				}
			}
			log.Info().Str("dump", strings.Join(pr, " | ")).Msg("tunnel_trace tproxy: PREROUTING matching lines")
		}
	}

	log.Info().Int("tunnel_id", tunnelID).Str("tun", tunName).Msg("tunnel_trace tproxy: SetupTProxy complete")
	return nil
}

// dropQUICVision is true for xtls-rprx-vision flows that cannot carry QUIC; false when URI uses vision-udp443.
func dropQUICVision(vlessFlow string) bool {
	f := strings.ToLower(strings.TrimSpace(vlessFlow))
	if f == "" {
		return false
	}
	if strings.Contains(f, "vision-udp443") {
		return false
	}
	return strings.Contains(f, "vision")
}

func TeardownTProxy(tunnelID int, tunName string, fwmark int) error {
	log.Info().Int("tunnel_id", tunnelID).Str("tun", tunName).Msg("tunnel_trace tproxy: TeardownTProxy")
	tb, err := iptables.New(iptables.IPFamily(iptables.ProtocolIPv4))
	if err != nil {
		return err
	}
	chain := fmt.Sprintf("XRAY2WG_%d", tunnelID)
	_ = tb.Delete(tableMangle, "PREROUTING", "-i", tunName, "-j", chain)
	if ok, _ := tb.ChainExists(tableMangle, chain); ok {
		_ = tb.ClearChain(tableMangle, chain)
		_ = tb.DeleteChain(tableMangle, chain)
	}
	return nil
}

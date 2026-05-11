//go:build linux

package netconf

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/rs/zerolog/log"
)

// PrepareTunForTransparentProxy relaxes reverse-path and local-address checks on the WG TUN.
// Strict rp_filter often drops replies where the transparent proxy restores the original
// remote IP as source (asymmetric path vs strict RPF on tun-wg*).
func PrepareTunForTransparentProxy(tunName string) {
	base := filepath.Join("/proc/sys/net/ipv4/conf", tunName)
	settings := []struct{ name, val string }{
		{"rp_filter", "0"},     // 0=off, 1=strict (default on many kernels breaks tproxy reply)
		{"accept_local", "1"},  // accept non-interface-primary addressing used by tproxy return path
	}
	for _, s := range settings {
		p := filepath.Join(base, s.name)
		if err := os.WriteFile(p, []byte(s.val+"\n"), 0o644); err != nil {
			log.Warn().Str("tun", tunName).Str("sysctl", s.name).Err(err).
				Msg("tunnel_trace tun sysctl (read-only /proc or missing iface is ok to ignore)")
		} else {
			log.Info().Str("tun", tunName).Str("sysctl", s.name).Str("value", s.val).Msg("tunnel_trace tun sysctl applied")
		}
	}
}

func readProcTrim(path string) string {
	b, err := os.ReadFile(path)
	if err != nil {
		return "?"
	}
	return strings.TrimSpace(string(b))
}

// LogIPv4ForwardingSnapshot logs kernel IPv4 forwarding / rp_filter (read-only). Use to verify Docker sysctls
// when /proc/sys is not writable from the process (tun_rp_filter should be 0 for tproxy replies).
func LogIPv4ForwardingSnapshot(tunName string) {
	log.Info().
		Str("ip_forward", readProcTrim("/proc/sys/net/ipv4/ip_forward")).
		Str("all_rp_filter", readProcTrim("/proc/sys/net/ipv4/conf/all/rp_filter")).
		Str("default_rp_filter", readProcTrim("/proc/sys/net/ipv4/conf/default/rp_filter")).
		Str("tun_rp_filter", readProcTrim(filepath.Join("/proc/sys/net/ipv4/conf", tunName, "rp_filter"))).
		Msg("tunnel_trace sysctl snapshot (reads; use Docker sysctls if tun_rp_filter≠0 or ip_forward≠1)")
}

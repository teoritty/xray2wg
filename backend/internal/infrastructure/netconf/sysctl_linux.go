//go:build linux

package netconf

import (
	"fmt"
	"os"

	"github.com/rs/zerolog/log"
)

// EnableForwarding tries to enable IPv4/IPv6 forwarding in the current network namespace.
// In read-only /proc (hardened Kubernetes, read-only rootfs) this may fail entirely; use
// Docker/Kubernetes sysctls on the pod then.
func EnableForwarding() error {
	e4 := os.WriteFile("/proc/sys/net/ipv4/ip_forward", []byte("1\n"), 0o644)
	e6 := os.WriteFile("/proc/sys/net/ipv6/conf/all/forwarding", []byte("1\n"), 0o644)
	if e4 != nil && e6 != nil {
		log.Warn().AnErr("ipv4", e4).AnErr("ipv6", e6).Msg("tunnel_trace sysctl: both ipv4 and ipv6 forwarding writes failed")
		return fmt.Errorf("cannot write sysctl (read-only /proc?): ipv4: %w; ipv6: %v", e4, e6)
	}
	if e4 != nil {
		log.Warn().Err(e4).Msg("tunnel_trace sysctl: ipv4 ip_forward write failed (ipv6 may be ok)")
	}
	if e6 != nil {
		log.Warn().Err(e6).Msg("tunnel_trace sysctl: ipv6 all.forwarding write failed (ipv4 may be ok)")
	}
	log.Info().Bool("ipv4_ok", e4 == nil).Bool("ipv6_ok", e6 == nil).Msg("tunnel_trace sysctl: forwarding write attempt finished")
	return nil
}

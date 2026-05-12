// Package tracenv reads optional diagnostics environment variables (no init side effects).
package tracenv

import (
	"os"
	"strings"
)

// TunnelTrace enables verbose per-phase tunnel / netfilter / routing logs when true.
func TunnelTrace() bool {
	v := strings.TrimSpace(strings.ToLower(os.Getenv("XRAY2WG_TUNNEL_TRACE")))
	return v == "1" || v == "true" || v == "yes"
}

// XrayLogLevel returns xray-core JSON log.logLevel (warning | info | debug | ...).
func XrayLogLevel() string {
	v := strings.TrimSpace(os.Getenv("XRAY_LOG_LEVEL"))
	if v == "" {
		return "warning"
	}
	return v
}

// XrayAccessLog returns the value for xray-core JSON log.access.
// Empty string means stdout (xray-core default); "none" disables access logs entirely.
// Controlled by XRAY_ACCESS_LOG env var.
func XrayAccessLog() string {
	return strings.TrimSpace(os.Getenv("XRAY_ACCESS_LOG"))
}

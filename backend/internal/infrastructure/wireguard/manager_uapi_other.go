//go:build !linux

package wireguardinfra

import (
	"fmt"
	"net"

	"golang.zx2c4.com/wireguard/device"
)

// startUAPI is a no-op stub for non-linux builds. The production target is
// Linux (Docker container) where the live-stats UAPI socket is required;
// this stub exists so the package still compiles for cross-platform dev
// builds and unit tests on Windows / macOS.
func startUAPI(_ *device.Device, _ string) (net.Listener, chan struct{}, error) {
	return nil, nil, fmt.Errorf("UAPI socket not supported on this platform; live wireguard stats unavailable")
}

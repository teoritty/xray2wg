package netconf

import (
	"net"
	"os"
	"strings"
)

// UnconfiguredPeerEndpointHost is substituted when no explicit host is set and
// no suitable LAN/public IPv4 can be detected (avoids invalid "Endpoint = :port").
const UnconfiguredPeerEndpointHost = "__CONFIGURE_SERVER_HOST__"

func runningInsideDocker() bool {
	_, err := os.Stat("/.dockerenv")
	return err == nil
}

func skipInterfaceNameForPeerEndpoint(name string) bool {
	n := strings.ToLower(strings.TrimSpace(name))
	switch {
	case n == "", n == "lo":
		return true
	case strings.HasPrefix(n, "docker"):
		return true
	case strings.HasPrefix(n, "br-"):
		return true
	case strings.HasPrefix(n, "veth"):
		return true
	case strings.HasPrefix(n, "virbr"):
		return true
	case strings.HasPrefix(n, "vbox"):
		return true
	case strings.HasPrefix(n, "vmnet"):
		return true
	case strings.HasPrefix(n, "tun-wg"):
		return true
	default:
		return false
	}
}

func ipInDockerDefaultBridge(ip net.IP) bool {
	ip4 := ip.To4()
	if ip4 == nil {
		return false
	}
	// Default docker0 bridge range; container eth0 addresses are usually here
	// and are not reachable from the LAN as the WireGuard endpoint.
	return ip4[0] == 172 && ip4[1] == 17
}

// scoreIPv4 ranks addresses for peer Endpoint selection (higher is better).
func scoreIPv4(ip net.IP) int {
	ip4 := ip.To4()
	if ip4 == nil || !ip4.IsGlobalUnicast() || ip4.IsLoopback() || ip4.IsLinkLocalUnicast() {
		return -1
	}
	a, b, c, d := int(ip4[0]), int(ip4[1]), int(ip4[2]), int(ip4[3])
	if ip4.IsPrivate() {
		switch {
		case a == 192 && b == 168:
			return 4_000_000 + c*1000 + d
		case a == 10:
			return 3_000_000 + (b<<16 | c<<8 | d)
		case a == 172 && b >= 16 && b <= 31:
			return 2_000_000 + (b<<16 | c<<8 | d)
		default:
			return 500_000 + (a<<24 | b<<16 | c<<8 | d)
		}
	}
	// Global unicast, non-private (e.g. VPS public IPv4)
	return 1_000_000 + (a<<24 | b<<16 | c<<8 | d)
}

// ifaceAddrSnapshot is a testable view of one interface's IPv4 addresses.
type ifaceAddrSnapshot struct {
	Name  string
	Flags net.Flags
	IPs   []net.IP
}

func snapshotIPv4AddrsFromInterfaces(ifaces []net.Interface) ([]ifaceAddrSnapshot, error) {
	var out []ifaceAddrSnapshot
	for _, iface := range ifaces {
		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}
		var ips []net.IP
		for _, a := range addrs {
			var ip net.IP
			switch v := a.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			default:
				continue
			}
			ip4 := ip.To4()
			if ip4 == nil || ip4.IsLoopback() {
				continue
			}
			if ip4.IsLinkLocalUnicast() {
				continue
			}
			if !(ip4.IsPrivate() || ip4.IsGlobalUnicast() && !ip4.IsPrivate()) {
				continue
			}
			ips = append(ips, ip4)
		}
		if len(ips) == 0 {
			continue
		}
		out = append(out, ifaceAddrSnapshot{Name: iface.Name, Flags: iface.Flags, IPs: ips})
	}
	return out, nil
}

func pickPeerEndpointHostFromSnapshots(inDocker bool, snaps []ifaceAddrSnapshot) string {
	var bestIP net.IP
	bestScore := -1
	for _, snap := range snaps {
		if snap.Flags&net.FlagUp == 0 || snap.Flags&net.FlagLoopback != 0 {
			continue
		}
		if skipInterfaceNameForPeerEndpoint(snap.Name) {
			continue
		}
		for _, ip := range snap.IPs {
			ip4 := ip.To4()
			if ip4 == nil {
				continue
			}
			if inDocker && ipInDockerDefaultBridge(ip4) {
				continue
			}
			s := scoreIPv4(ip4)
			if s > bestScore {
				bestScore = s
				bestIP = ip4
			}
		}
	}
	if bestIP == nil {
		return ""
	}
	return bestIP.String()
}

// ResolvePeerEndpointHost picks the host part for WireGuard client Endpoint lines.
// Priority: non-empty serverHost (settings), then non-empty publicHostEnv (e.g. PUBLIC_HOST),
// then a scored IPv4 from local interfaces (excluding typical Docker bridge/tunnel names),
// then UnconfiguredPeerEndpointHost so the config never contains ":port" with an empty host.
func ResolvePeerEndpointHost(serverHost, publicHostEnv string) string {
	return resolvePeerEndpointHostFromSnapshots(
		strings.TrimSpace(serverHost),
		strings.TrimSpace(publicHostEnv),
		runningInsideDocker(),
		mustSnapshotIPv4Addrs(),
	)
}

func resolvePeerEndpointHostFromSnapshots(serverHost, publicHostEnv string, inDocker bool, snaps []ifaceAddrSnapshot) string {
	if serverHost != "" {
		return serverHost
	}
	if publicHostEnv != "" {
		return publicHostEnv
	}
	if h := pickPeerEndpointHostFromSnapshots(inDocker, snaps); h != "" {
		return h
	}
	return UnconfiguredPeerEndpointHost
}

func mustSnapshotIPv4Addrs() []ifaceAddrSnapshot {
	ifs, err := net.Interfaces()
	if err != nil {
		return nil
	}
	snaps, err := snapshotIPv4AddrsFromInterfaces(ifs)
	if err != nil {
		return nil
	}
	return snaps
}

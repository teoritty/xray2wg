package service

import (
	"fmt"
	"strings"
	"time"

	"xray2wg/backend/internal/domain"
)

func BuildMikrotikCommands(iface *domain.WgInterface, peer *domain.WgPeer, serverHost string, clientPrivB64, pskB64 string) string {
	host := serverHost
	if strings.TrimSpace(host) == "" {
		host = "<YOUR_SERVER_IP>"
	}
	peerName := sanitizeName(peer.Name)
	ts := time.Now().UTC().Format(time.RFC3339)
	ka := peer.PersistentKeepalive
	if ka <= 0 {
		ka = 25
	}

	return fmt.Sprintf(`# === xray2wg: %s → %s ===
# Generated: %s

# 1. Create WireGuard interface
/interface wireguard
add name="xray2wg-%s" \
    private-key="%s" \
    listen-port=0 \
    mtu=%d \
    comment="xray2wg tunnel"

# 2. Add server as peer
/interface wireguard peers
add interface="xray2wg-%s" \
    public-key="%s" \
    preshared-key="%s" \
    endpoint-address=%s \
    endpoint-port=%d \
    allowed-address=0.0.0.0/0 \
    persistent-keepalive=%ds \
    comment="xray2wg server"

# 3. Assign IP address
/ip address
add address=%s \
    interface="xray2wg-%s" \
    comment="xray2wg client IP"

# 4. Allow WireGuard traffic through firewall (input chain)
/ip firewall filter
add chain=input \
    protocol=udp \
    dst-port=%d \
    action=accept \
    place-before=0 \
    comment="Allow xray2wg WireGuard"

# NOTE: Routing rules NOT configured — add manually as needed.
# To route specific traffic through this tunnel:
#   /ip route add dst-address=X.X.X.X/X gateway=xray2wg-%s
`, peer.Name, iface.Name, ts,
		peerName, clientPrivB64, iface.MTU,
		peerName, iface.PublicKey, pskB64, host, iface.ListenPort, ka,
		peer.ClientAddress, peerName,
		iface.ListenPort,
		peerName)
}

func sanitizeName(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return "client"
	}
	s = strings.ReplaceAll(s, `"`, "")
	s = strings.ReplaceAll(s, "\n", "")
	return s
}

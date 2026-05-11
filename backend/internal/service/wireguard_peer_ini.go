package service

import (
	"fmt"
	"strings"
)

// WgPeerConfigEndpointHostPlaceholder is substituted in the generated .conf and
// replaced with the resolved host in the HTTP handler (see api/register.go).
const WgPeerConfigEndpointHostPlaceholder = "<replace-with-server-host>"

// wireGuardPeerClientIni returns WireGuard client .conf text for import/QR.
// Empty dns or psk omits those keys so parsers (including mobile importers) accept the file.
func wireGuardPeerClientIni(
	privKey, clientAddr, dns string,
	mtu int,
	serverPub, psk, allowedIPs string,
	persistentKeepalive, listenPort int,
) string {
	var b strings.Builder
	privKey = strings.TrimSpace(privKey)
	clientAddr = strings.TrimSpace(clientAddr)
	dns = strings.TrimSpace(dns)
	serverPub = strings.TrimSpace(serverPub)
	psk = strings.TrimSpace(psk)
	allowedIPs = strings.TrimSpace(allowedIPs)
	if allowedIPs == "" {
		allowedIPs = "0.0.0.0/0"
	}

	fmt.Fprintf(&b, "[Interface]\n")
	fmt.Fprintf(&b, "PrivateKey = %s\n", privKey)
	fmt.Fprintf(&b, "Address = %s\n", clientAddr)
	if dns != "" {
		fmt.Fprintf(&b, "DNS = %s\n", dns)
	}
	fmt.Fprintf(&b, "MTU = %d\n\n", mtu)

	fmt.Fprintf(&b, "[Peer]\n")
	fmt.Fprintf(&b, "PublicKey = %s\n", serverPub)
	if psk != "" {
		fmt.Fprintf(&b, "PresharedKey = %s\n", psk)
	}
	fmt.Fprintf(&b, "AllowedIPs = %s\n", allowedIPs)
	fmt.Fprintf(&b, "PersistentKeepalive = %d\n", persistentKeepalive)
	fmt.Fprintf(&b, "Endpoint = %s:%d\n", WgPeerConfigEndpointHostPlaceholder, listenPort)
	return b.String()
}

package service

import (
	"strings"
	"testing"

	"xray2wg/backend/internal/domain"
)

func TestBuildMikrotikCommandsPlaceholderHost(t *testing.T) {
	iface := &domain.WgInterface{
		Name:       "WAN",
		PublicKey:  "dGVzdC1zZXJ2ZXItcHVibGljLWtleQ==",
		ListenPort: 51820,
		MTU:        1420,
	}
	peer := &domain.WgPeer{
		Name:                "Living",
		ClientAddress:       "10.100.1.2/24",
		AllowedIPs:          "0.0.0.0/0",
		PersistentKeepalive: 25,
	}
	out := BuildMikrotikCommands(iface, peer, "", "client-priv-b64", "psk-b64-placeholder")
	if !strings.Contains(out, "<YOUR_SERVER_IP>") {
		t.Fatal("missing placeholder")
	}
	if !strings.Contains(out, "endpoint-port=51820") {
		t.Fatal("listen port missing")
	}
	if !strings.Contains(out, "xray2wg-Living") {
		t.Fatal(out)
	}
}

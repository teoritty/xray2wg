package vless

import (
	"encoding/json"
	"testing"
)

func TestParseURI(t *testing.T) {
	raw := `vless://550e8400-e29b-41d4-a716-446655440000@vpn.example.com:443?encryption=none&security=reality&sni=vpn.example.com&fp=chrome&pbk=AbCdEf123456&type=tcp&flow=xtls-rprx-vision&sid=a1b2c3d4#MyNode`

	n, err := ParseURI(raw)
	if err != nil {
		t.Fatal(err)
	}
	if n.UUID != "550e8400-e29b-41d4-a716-446655440000" {
		t.Fatalf("uuid: %q", n.UUID)
	}
	if n.Address != "vpn.example.com" || n.Port != 443 {
		t.Fatalf("host/port: %s:%d", n.Address, n.Port)
	}
	if n.Security != "reality" || n.Flow != "xtls-rprx-vision" || n.Network != "tcp" {
		t.Fatalf("security/flow/network: %q / %q / %q", n.Security, n.Flow, n.Network)
	}
	var sec map[string]any
	if err := json.Unmarshal(n.SecurityConfig, &sec); err != nil {
		t.Fatalf("SecurityConfig: %v", err)
	}
	if sec["publicKey"] != "AbCdEf123456" {
		t.Fatalf("publicKey: %v", sec["publicKey"])
	}
	if n.DisplayName != "MyNode" {
		t.Fatalf("fragment name: %q", n.DisplayName)
	}
}

func TestParseURIDefaultFlowEmpty(t *testing.T) {
	raw := `vless://550e8400-e29b-41d4-a716-446655440000@vpn.example.com:443?security=reality&sni=vpn.example.com&fp=chrome&pbk=AbCdEf123456&type=tcp&sid=a1b2c3d4#Node`
	n, err := ParseURI(raw)
	if err != nil {
		t.Fatal(err)
	}
	if n.Flow != "" {
		t.Fatalf("default flow must be empty for transparent gateway; got %q", n.Flow)
	}
}

func TestParseURIDefaultSecurityNone(t *testing.T) {
	// When ?security= is omitted the modern xray-core canonical default is "none".
	n, err := ParseURI(`vless://abc@x.example.com:443?type=tcp#n`)
	if err != nil {
		t.Fatal(err)
	}
	if n.Security != "none" {
		t.Fatalf("default security: want none, got %q", n.Security)
	}
}

func TestParseURIRealityRequiresPublicKey(t *testing.T) {
	// Without pbk REALITY cannot complete the handshake; the parser must reject it at
	// create-time rather than fail silently at tunnel-start.
	_, err := ParseURI(`vless://abc@x.example.com:443?type=tcp&security=reality&sni=x.example.com#n`)
	if err == nil {
		t.Fatal("expected error for reality URI without pbk")
	}
}

package vless

import "testing"

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
	if n.Security != "reality" || n.Flow != "xtls-rprx-vision" {
		t.Fatalf("security/flow")
	}
	if n.PublicKey != "AbCdEf123456" {
		t.Fatalf("pbk")
	}
	if n.DisplayName != "MyNode" {
		t.Fatalf("fragment name")
	}
}

func TestParseURIDefaultFlowEmpty(t *testing.T) {
	raw := `vless://550e8400-e29b-41d4-a716-446655440000@vpn.example.com:443?encryption=none&security=reality&sni=vpn.example.com&fp=chrome&pbk=AbCdEf123456&type=tcp&sid=a1b2c3d4#Node`
	n, err := ParseURI(raw)
	if err != nil {
		t.Fatal(err)
	}
	if n.Flow != "" {
		t.Fatalf("default flow must be empty for transparent gateway; got %q", n.Flow)
	}
}

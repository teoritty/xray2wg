package vless

import (
	"encoding/json"
	"testing"
)

func TestBuild_roundTripWithParseURI(t *testing.T) {
	in := BuildInput{
		DisplayName:    "node-xhttp",
		UUID:           "550e8400-e29b-41d4-a716-446655440000",
		Address:        "vpn.example.com",
		Port:           443,
		Flow:           "xtls-rprx-vision",
		Encryption:     "none",
		PacketEncoding: "xudp",
		Network:        "xhttp",
		TransportConfig: json.RawMessage(`{
			"path": "/api",
			"host": "cdn.example.com",
			"mode": "stream-up"
		}`),
		Security: "reality",
		SecurityConfig: json.RawMessage(`{
			"serverName": "vpn.example.com",
			"fingerprint": "chrome",
			"publicKey": "abc123",
			"shortId": "ab"
		}`),
	}
	node, err := Build(in)
	if err != nil {
		t.Fatal(err)
	}
	if node.RawURI == "" {
		t.Fatal("Build did not produce a RawURI")
	}
	// The structured input plus the rebuilt RawURI must agree: parsing the RawURI yields a
	// node whose canonical fields match the originals.
	parsed, err := ParseURI(node.RawURI)
	if err != nil {
		t.Fatalf("re-parse: %v\nuri=%s", err, node.RawURI)
	}
	if parsed.Network != "xhttp" || parsed.Security != "reality" {
		t.Fatalf("re-parsed: %q / %q", parsed.Network, parsed.Security)
	}
	if parsed.Flow != "xtls-rprx-vision" || parsed.PacketEncoding != "xudp" {
		t.Fatalf("re-parsed flow=%q pe=%q", parsed.Flow, parsed.PacketEncoding)
	}
	var sec map[string]any
	if err := json.Unmarshal(parsed.SecurityConfig, &sec); err != nil {
		t.Fatal(err)
	}
	if sec["publicKey"] != "abc123" || sec["serverName"] != "vpn.example.com" {
		t.Fatalf("reality fields lost in round-trip: %v", sec)
	}
}

func TestBuild_validatesPort(t *testing.T) {
	for _, p := range []int{0, -1, 70000} {
		_, err := Build(BuildInput{UUID: "u", Address: "a", Port: p, Network: "tcp", Security: "none"})
		if err == nil {
			t.Fatalf("port %d: expected error", p)
		}
	}
}

func TestBuild_requiresUUIDAndAddress(t *testing.T) {
	if _, err := Build(BuildInput{Address: "a", Port: 443}); err == nil {
		t.Fatal("missing uuid: expected error")
	}
	if _, err := Build(BuildInput{UUID: "u", Port: 443}); err == nil {
		t.Fatal("missing address: expected error")
	}
}

func TestBuild_rejectsRealityWithoutPbk(t *testing.T) {
	_, err := Build(BuildInput{
		UUID: "u", Address: "a", Port: 443,
		Network:        "tcp",
		Security:       "reality",
		SecurityConfig: json.RawMessage(`{"serverName": "x"}`),
	})
	if err == nil {
		t.Fatal("expected error: reality without publicKey")
	}
}

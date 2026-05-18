package xrayinfra

import (
	"encoding/json"
	"testing"

	"xray2wg/backend/internal/domain"
	"xray2wg/backend/internal/vless"
)

// nodeFromURI parses a vless:// URI into a node for test convenience. The full parser is
// exercised so the test surface matches what real subscription imports produce.
func nodeFromURI(t *testing.T, raw string) *domain.VlessNode {
	t.Helper()
	n, err := vless.ParseURI(raw)
	if err != nil {
		t.Fatalf("ParseURI(%q): %v", raw, err)
	}
	return n
}

func TestBuildXrayConfigVisionFlowPreserved(t *testing.T) {
	n := nodeFromURI(t, "vless://550e8400-e29b-41d4-a716-446655440000@vpn.example.com:443?"+
		"type=tcp&security=reality&sni=vpn.example.com&fp=chrome&pbk=pbk&sid=a1&flow=xtls-rprx-vision#MyNode")
	raw, err := BuildXrayConfig(13001, 0x1001, "", []*domain.VlessNode{n}, domain.BalancingRoundRobin)
	if err != nil {
		t.Fatal(err)
	}
	var doc struct {
		Outbounds []struct {
			Protocol string         `json:"protocol"`
			Settings map[string]any `json:"settings"`
		} `json:"outbounds"`
	}
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatal(err)
	}
	var userFlow string
	for _, ob := range doc.Outbounds {
		if ob.Protocol != "vless" {
			continue
		}
		vnext := ob.Settings["vnext"].([]any)
		users := vnext[0].(map[string]any)["users"].([]any)
		userFlow, _ = users[0].(map[string]any)["flow"].(string)
		break
	}
	if userFlow != "xtls-rprx-vision" {
		t.Fatalf("expected vision flow, got %q", userFlow)
	}
}

func TestBuildXrayConfigDNSInboundWhenGatewaySet(t *testing.T) {
	n := nodeFromURI(t, "vless://550e8400-e29b-41d4-a716-446655440000@vpn.example.com:443?"+
		"type=tcp&security=reality&sni=vpn.example.com&fp=chrome&pbk=pbk&sid=a1#node")
	raw, err := BuildXrayConfig(13001, 0x1001, "10.100.1.1", []*domain.VlessNode{n}, domain.BalancingRoundRobin)
	if err != nil {
		t.Fatal(err)
	}
	var doc struct {
		Inbounds []struct {
			Tag      string `json:"tag"`
			Listen   string `json:"listen"`
			Port     int    `json:"port"`
			Protocol string `json:"protocol"`
		} `json:"inbounds"`
	}
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatal(err)
	}
	if len(doc.Inbounds) != 2 {
		t.Fatalf("inbounds: want 2, got %d", len(doc.Inbounds))
	}
	var dnsIn bool
	for _, in := range doc.Inbounds {
		if in.Tag == "dns-in" && in.Listen == "10.100.1.1" && in.Port == 53 && in.Protocol == "dokodemo-door" {
			dnsIn = true
		}
	}
	if !dnsIn {
		t.Fatalf("dns-in inbound missing: %+v", doc.Inbounds)
	}
}

func TestBuildXrayConfigMuxEnabledWhenNoFlow(t *testing.T) {
	n := nodeFromURI(t, "vless://550e8400-e29b-41d4-a716-446655440000@vpn.example.com:443?"+
		"type=tcp&security=reality&sni=vpn.example.com&pbk=pbk&sid=a1#node")
	raw, err := BuildXrayConfig(13001, 0x1001, "", []*domain.VlessNode{n}, domain.BalancingRoundRobin)
	if err != nil {
		t.Fatal(err)
	}
	var doc struct {
		Outbounds []map[string]any `json:"outbounds"`
	}
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatal(err)
	}
	for _, ob := range doc.Outbounds {
		if ob["protocol"] != "vless" {
			continue
		}
		mux, ok := ob["mux"].(map[string]any)
		if !ok {
			t.Fatal("mux field missing on VLESS outbound with no flow")
		}
		if mux["enabled"] != true {
			t.Fatalf("mux.enabled: want true, got %v", mux["enabled"])
		}
		return
	}
	t.Fatal("no vless outbound found")
}

func TestBuildXrayConfigMuxDisabledWhenFlowSet(t *testing.T) {
	n := nodeFromURI(t, "vless://550e8400-e29b-41d4-a716-446655440000@vpn.example.com:443?"+
		"type=tcp&security=reality&sni=vpn.example.com&fp=chrome&pbk=pbk&sid=a1&flow=xtls-rprx-vision#node")
	raw, err := BuildXrayConfig(13001, 0x1001, "", []*domain.VlessNode{n}, domain.BalancingRoundRobin)
	if err != nil {
		t.Fatal(err)
	}
	var doc struct {
		Outbounds []map[string]any `json:"outbounds"`
	}
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatal(err)
	}
	for _, ob := range doc.Outbounds {
		if ob["protocol"] != "vless" {
			continue
		}
		if _, ok := ob["mux"]; ok {
			t.Fatal("mux must not be set on VLESS outbound with xtls-rprx-vision flow")
		}
		return
	}
	t.Fatal("no vless outbound found")
}

func TestBuildXrayConfigMuxMultiNodeMixedFlow(t *testing.T) {
	nodes := []*domain.VlessNode{
		nodeFromURI(t, "vless://aaa@a.example.com:443?type=tcp&security=reality&sni=a.example.com&pbk=p&sid=s#a"),
		nodeFromURI(t, "vless://bbb@b.example.com:443?type=tcp&security=reality&sni=b.example.com&pbk=p&sid=s&flow=xtls-rprx-vision#b"),
	}
	raw, err := BuildXrayConfig(13001, 0x1001, "", nodes, domain.BalancingRoundRobin)
	if err != nil {
		t.Fatal(err)
	}
	var doc struct {
		Outbounds []map[string]any `json:"outbounds"`
	}
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatal(err)
	}
	hasMux, noMux := 0, 0
	for _, ob := range doc.Outbounds {
		if ob["protocol"] != "vless" {
			continue
		}
		if _, ok := ob["mux"]; ok {
			hasMux++
		} else {
			noMux++
		}
	}
	if hasMux != 1 || noMux != 1 {
		t.Fatalf("expected 1 outbound with mux and 1 without, got hasMux=%d noMux=%d", hasMux, noMux)
	}
}

func TestBuildXrayConfigObservatoryEnabledForMultiNodeRoundRobin(t *testing.T) {
	nodes := []*domain.VlessNode{
		nodeFromURI(t, "vless://aaa@a.example.com:443?type=tcp&security=reality&sni=a.example.com&pbk=p&sid=s#a"),
		nodeFromURI(t, "vless://bbb@b.example.com:443?type=tcp&security=reality&sni=b.example.com&pbk=p&sid=s#b"),
	}
	for _, strategy := range []domain.BalancingStrategy{domain.BalancingRoundRobin, domain.BalancingLeastPing} {
		raw, err := BuildXrayConfig(13001, 0x1001, "", nodes, strategy)
		if err != nil {
			t.Fatalf("%s: %v", strategy, err)
		}
		var doc struct {
			Observatory map[string]any `json:"observatory"`
			Routing     struct {
				Balancers []struct {
					Tag         string         `json:"tag"`
					Selector    []string       `json:"selector"`
					FallbackTag string         `json:"fallbackTag"`
					Strategy    map[string]any `json:"strategy"`
				} `json:"balancers"`
			} `json:"routing"`
		}
		if err := json.Unmarshal(raw, &doc); err != nil {
			t.Fatalf("%s: %v", strategy, err)
		}
		if doc.Observatory == nil {
			t.Fatalf("%s: observatory must be present", strategy)
		}
		wantType := "roundRobin"
		if strategy == domain.BalancingLeastPing {
			wantType = "leastPing"
		}
		if got := doc.Routing.Balancers[0].Strategy["type"]; got != wantType {
			t.Fatalf("%s: balancer strategy type: want %s, got %v", strategy, wantType, got)
		}
		// fallbackTag is what wires the observatory into the strategy (xray-core
		// app/router/balancing.go:32-40). Without it RoundRobinStrategy never filters
		// dead outbounds and traffic flips between alive and dead nodes at ~50%.
		if got := doc.Routing.Balancers[0].FallbackTag; got != "vless-out-1" {
			t.Fatalf("%s: balancer fallbackTag: want vless-out-1, got %q", strategy, got)
		}
	}
}

func TestBuildXrayConfigMultiNode_directOutboundIsLast(t *testing.T) {
	nodes := []*domain.VlessNode{
		nodeFromURI(t, "vless://aaa@a.example.com:443?type=tcp&security=reality&sni=a.example.com&pbk=p&sid=s#a"),
		nodeFromURI(t, "vless://bbb@b.example.com:443?type=tcp&security=reality&sni=b.example.com&pbk=p&sid=s#b"),
	}
	raw, err := BuildXrayConfig(13001, 0x1001, "", nodes, domain.BalancingRoundRobin)
	if err != nil {
		t.Fatal(err)
	}
	var doc struct {
		Outbounds []map[string]any `json:"outbounds"`
	}
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatal(err)
	}
	if len(doc.Outbounds) < 2 {
		t.Fatalf("expected at least 2 outbounds, got %d", len(doc.Outbounds))
	}
	// xray-core uses the first outbound as the implicit default handler. Putting "direct"
	// first would silently leak traffic when any routing edge case occurs.
	if got := doc.Outbounds[0]["tag"]; got != "vless-out-1" {
		t.Fatalf("first outbound must be vless-out-1 to avoid default-handler leak; got %v", got)
	}
	last := doc.Outbounds[len(doc.Outbounds)-1]
	if last["tag"] != "direct" || last["protocol"] != "freedom" {
		t.Fatalf("last outbound must be the freedom 'direct' fallback; got %v", last)
	}
}

func TestBuildXrayConfigMultiNode_observatoryUsesHTTPSProbe(t *testing.T) {
	nodes := []*domain.VlessNode{
		nodeFromURI(t, "vless://aaa@a.example.com:443?type=tcp&security=reality&sni=a.example.com&pbk=p&sid=s#a"),
		nodeFromURI(t, "vless://bbb@b.example.com:443?type=tcp&security=reality&sni=b.example.com&pbk=p&sid=s#b"),
	}
	raw, err := BuildXrayConfig(13001, 0x1001, "", nodes, domain.BalancingRoundRobin)
	if err != nil {
		t.Fatal(err)
	}
	var doc struct {
		Observatory map[string]any `json:"observatory"`
	}
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatal(err)
	}
	got, _ := doc.Observatory["probeUrl"].(string)
	if got == "" || got[:6] != "https:" {
		t.Fatalf("probeUrl must use HTTPS for a representative liveness signal; got %q", got)
	}
}

func TestBuildXrayConfigSingleNodeHasNoObservatory(t *testing.T) {
	n := nodeFromURI(t, "vless://aaa@a.example.com:443?type=tcp&security=reality&sni=a.example.com&pbk=p&sid=s#a")
	raw, err := BuildXrayConfig(13001, 0x1001, "", []*domain.VlessNode{n}, domain.BalancingRoundRobin)
	if err != nil {
		t.Fatal(err)
	}
	var doc map[string]any
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatal(err)
	}
	if _, ok := doc["observatory"]; ok {
		t.Fatal("observatory must not be present for single-node configs")
	}
}

func TestBuildXrayConfigTransparentInboundDoesNotSetSocketMark(t *testing.T) {
	n := nodeFromURI(t, "vless://aaa@a.example.com:443?type=tcp&security=reality&sni=a.example.com&pbk=p&sid=s#a")
	raw, err := BuildXrayConfig(13001, 0x1001, "10.100.1.1", []*domain.VlessNode{n}, domain.BalancingRoundRobin)
	if err != nil {
		t.Fatal(err)
	}
	var doc struct {
		Inbounds []struct {
			Tag            string         `json:"tag"`
			StreamSettings map[string]any `json:"streamSettings"`
		} `json:"inbounds"`
	}
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatal(err)
	}
	for _, in := range doc.Inbounds {
		if in.Tag != "transparent-in" {
			continue
		}
		sockopt, _ := in.StreamSettings["sockopt"].(map[string]any)
		if _, ok := sockopt["mark"]; ok {
			t.Fatalf("transparent inbound sockopt must not set mark: %+v", sockopt)
		}
		return
	}
	t.Fatal("transparent-in inbound missing")
}

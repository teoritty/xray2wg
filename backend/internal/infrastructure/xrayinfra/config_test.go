package xrayinfra

import (
	"encoding/json"
	"testing"

	"xray2wg/backend/internal/domain"
)

func TestBuildXrayConfigVisionFlowPreserved(t *testing.T) {
	n := &domain.VlessNode{
		UUID:        "550e8400-e29b-41d4-a716-446655440000",
		Address:     "vpn.example.com",
		Port:        443,
		Flow:        "xtls-rprx-vision",
		Network:     "tcp",
		Security:    "reality",
		SNI:         "vpn.example.com",
		PublicKey:   "pbk",
		ShortID:     "a1",
		Fingerprint: "chrome",
	}
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
		vnext, _ := ob.Settings["vnext"].([]any)
		if len(vnext) == 0 {
			t.Fatal("no vnext")
		}
		vm0, _ := vnext[0].(map[string]any)
		users, _ := vm0["users"].([]any)
		if len(users) == 0 {
			t.Fatal("no users")
		}
		u0, _ := users[0].(map[string]any)
		f, _ := u0["flow"].(string)
		userFlow = f
		break
	}
	if userFlow != "xtls-rprx-vision" {
		t.Fatalf("expected vision flow in outbound user, got %q", userFlow)
	}
}

func TestBuildXrayConfigDNSInboundWhenGatewaySet(t *testing.T) {
	n := &domain.VlessNode{
		UUID:      "550e8400-e29b-41d4-a716-446655440000",
		Address:   "vpn.example.com",
		Port:      443,
		Network:   "tcp",
		Security:  "reality",
		SNI:       "vpn.example.com",
		PublicKey: "pbk",
		ShortID:   "a1",
	}
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
	n := &domain.VlessNode{
		UUID:      "550e8400-e29b-41d4-a716-446655440000",
		Address:   "vpn.example.com",
		Port:      443,
		Network:   "tcp",
		Security:  "reality",
		SNI:       "vpn.example.com",
		PublicKey: "pbk",
		ShortID:   "a1",
	}
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
	n := &domain.VlessNode{
		UUID:        "550e8400-e29b-41d4-a716-446655440000",
		Address:     "vpn.example.com",
		Port:        443,
		Flow:        "xtls-rprx-vision",
		Network:     "tcp",
		Security:    "reality",
		SNI:         "vpn.example.com",
		PublicKey:   "pbk",
		ShortID:     "a1",
		Fingerprint: "chrome",
	}
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
		{UUID: "aaa", Address: "a.example.com", Port: 443, Network: "tcp", Security: "reality", SNI: "a.example.com", PublicKey: "p", ShortID: "s"},
		{UUID: "bbb", Address: "b.example.com", Port: 443, Flow: "xtls-rprx-vision", Network: "tcp", Security: "reality", SNI: "b.example.com", PublicKey: "p", ShortID: "s"},
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
		{UUID: "aaa", Address: "a.example.com", Port: 443, Network: "tcp", Security: "reality", SNI: "a.example.com", PublicKey: "p", ShortID: "s"},
		{UUID: "bbb", Address: "b.example.com", Port: 443, Network: "tcp", Security: "reality", SNI: "b.example.com", PublicKey: "p", ShortID: "s"},
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
					Tag      string         `json:"tag"`
					Selector []string       `json:"selector"`
					Strategy map[string]any `json:"strategy"`
				} `json:"balancers"`
			} `json:"routing"`
		}
		if err := json.Unmarshal(raw, &doc); err != nil {
			t.Fatalf("%s: %v", strategy, err)
		}
		if doc.Observatory == nil {
			t.Fatalf("%s: observatory must be present for multi-node configs (regression for #4)", strategy)
		}
		if got := doc.Observatory["probeUrl"]; got != "http://www.google.com/generate_204" {
			t.Fatalf("%s: probeUrl: want generate_204, got %v", strategy, got)
		}
		if len(doc.Routing.Balancers) != 1 {
			t.Fatalf("%s: want 1 balancer, got %d", strategy, len(doc.Routing.Balancers))
		}
		wantType := "roundRobin"
		if strategy == domain.BalancingLeastPing {
			wantType = "leastPing"
		}
		if got := doc.Routing.Balancers[0].Strategy["type"]; got != wantType {
			t.Fatalf("%s: balancer strategy type: want %s, got %v", strategy, wantType, got)
		}
	}
}

func TestBuildXrayConfigSingleNodeHasNoObservatory(t *testing.T) {
	n := &domain.VlessNode{UUID: "aaa", Address: "a.example.com", Port: 443, Network: "tcp", Security: "reality", SNI: "a.example.com", PublicKey: "p", ShortID: "s"}
	raw, err := BuildXrayConfig(13001, 0x1001, "", []*domain.VlessNode{n}, domain.BalancingRoundRobin)
	if err != nil {
		t.Fatal(err)
	}
	var doc map[string]any
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatal(err)
	}
	if _, ok := doc["observatory"]; ok {
		t.Fatal("observatory must not be present for single-node configs (avoids unnecessary probes)")
	}
}

func TestBuildXrayConfigTransparentInboundDoesNotSetSocketMark(t *testing.T) {
	n := &domain.VlessNode{
		UUID:      "550e8400-e29b-41d4-a716-446655440000",
		Address:   "vpn.example.com",
		Port:      443,
		Network:   "tcp",
		Security:  "reality",
		SNI:       "vpn.example.com",
		PublicKey: "pbk",
		ShortID:   "a1",
	}
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

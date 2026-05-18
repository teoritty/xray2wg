package xrayinfra

import (
	"encoding/json"
	"fmt"
	"strings"

	"xray2wg/backend/internal/domain"
	"xray2wg/backend/internal/tracenv"
	"xray2wg/backend/internal/vless/security"
	"xray2wg/backend/internal/vless/transport"
)

// BuildXrayConfig produces embedded Xray JSON: tproxy dokodemo + VLESS outbound(s).
// When localGatewayIP is set (WG server IP on the tunnel, e.g. 10.100.1.1), an extra dokodemo inbound
// listens on localGatewayIP:53 and forwards to 1.1.1.1:53 through the same outbound — DNS from clients
// using DNS=10.100.x.1 avoids raw UDP to public resolvers through TPROXY.
// With multiple nodes a balancer is configured; strategy selects round-robin or least-ping mode.
func BuildXrayConfig(xrayListenPort int, fwmark int, localGatewayIP string, nodes []*domain.VlessNode, strategy domain.BalancingStrategy) ([]byte, error) {
	if len(nodes) == 0 {
		return nil, fmt.Errorf("BuildXrayConfig: no nodes provided")
	}

	inbounds := buildInbounds(xrayListenPort, fwmark, localGatewayIP)
	routeInboundTags := []string{"transparent-in"}
	if strings.TrimSpace(localGatewayIP) != "" {
		routeInboundTags = append(routeInboundTags, "dns-in")
	}

	var routing map[string]any
	var outbounds []any
	var observatoryDoc map[string]any

	if len(nodes) == 1 {
		// Single-node: original config path, no balancer overhead.
		outbounds, routing = buildSingleNodeConfig(nodes[0], routeInboundTags)
	} else {
		// Multi-node: balancer config.
		outbounds, routing, observatoryDoc = buildMultiNodeConfig(nodes, strategy, routeInboundTags)
	}

	logDoc := map[string]any{"logLevel": tracenv.XrayLogLevel()}
	if v := tracenv.XrayAccessLog(); v != "" {
		logDoc["access"] = v
	}
	doc := map[string]any{
		"log":       logDoc,
		"inbounds":  inbounds,
		"outbounds": outbounds,
		"routing":   routing,
		"dns":       map[string]any{"servers": []any{"8.8.8.8", "1.1.1.1"}},
	}
	if observatoryDoc != nil {
		doc["observatory"] = observatoryDoc
	}
	return json.Marshal(doc)
}

func buildInbounds(xrayListenPort, fwmark int, localGatewayIP string) []any {
	inbounds := []any{
		map[string]any{
			"tag":      "transparent-in",
			"listen":   "0.0.0.0",
			"port":     xrayListenPort,
			"protocol": "dokodemo-door",
			"settings": map[string]any{
				"network": "tcp,udp",
				// Required for TPROXY: read original dst from the transparent socket.
				"followRedirect": true,
				"address":        "0.0.0.0",
				"port":           0,
			},
			"streamSettings": streamSettingsInboundTProxy(fwmark),
			// Sniffing breaks raw-IP TCP flows through tproxy; keep off.
			"sniffing": map[string]any{"enabled": false},
		},
	}
	if g := strings.TrimSpace(localGatewayIP); g != "" {
		inbounds = append(inbounds, map[string]any{
			"tag":      "dns-in",
			"listen":   g,
			"port":     53,
			"protocol": "dokodemo-door",
			"settings": map[string]any{
				"network":        "tcp,udp",
				"followRedirect": false,
				"address":        "1.1.1.1",
				"port":           53,
			},
			"sniffing": map[string]any{"enabled": false},
		})
	}
	return inbounds
}

func buildSingleNodeConfig(node *domain.VlessNode, routeInboundTags []string) (outbounds []any, routing map[string]any) {
	userObj := map[string]any{"id": node.UUID, "encryption": "none"}
	if f := strings.TrimSpace(node.Flow); f != "" {
		userObj["flow"] = f
	}
	vlessOut := map[string]any{
		"protocol": "vless",
		"tag":      "proxy",
		"settings": map[string]any{
			"vnext": []any{
				map[string]any{
					"address": node.Address,
					"port":    node.Port,
					"users":   []any{userObj},
				},
			},
		},
		"streamSettings": streamSettingsOutbound(node),
	}
	if strings.TrimSpace(node.Flow) == "" {
		vlessOut["mux"] = map[string]any{"enabled": true, "concurrency": 8}
	}
	outbounds = []any{
		map[string]any{"protocol": "freedom", "tag": "direct"},
		vlessOut,
	}
	routing = map[string]any{
		"domainStrategy": "AsIs",
		"rules": []any{
			map[string]any{
				"type":        "field",
				"inboundTag":  routeInboundTags,
				"outboundTag": "proxy",
			},
		},
	}
	return
}

func buildMultiNodeConfig(nodes []*domain.VlessNode, strategy domain.BalancingStrategy, routeInboundTags []string) (outbounds []any, routing map[string]any, observatory map[string]any) {
	outbounds = []any{map[string]any{"protocol": "freedom", "tag": "direct"}}

	for i, node := range nodes {
		userObj := map[string]any{"id": node.UUID, "encryption": "none"}
		if f := strings.TrimSpace(node.Flow); f != "" {
			userObj["flow"] = f
		}
		tag := fmt.Sprintf("vless-out-%d", i+1)
		outEntry := map[string]any{
			"protocol": "vless",
			"tag":      tag,
			"settings": map[string]any{
				"vnext": []any{
					map[string]any{
						"address": node.Address,
						"port":    node.Port,
						"users":   []any{userObj},
					},
				},
			},
			"streamSettings": streamSettingsOutbound(node),
		}
		if strings.TrimSpace(node.Flow) == "" {
			outEntry["mux"] = map[string]any{"enabled": true, "concurrency": 8}
		}
		outbounds = append(outbounds, outEntry)
	}

	strategyType := "roundRobin"
	if strategy == domain.BalancingLeastPing {
		strategyType = "leastPing"
	}

	routing = map[string]any{
		"domainStrategy": "AsIs",
		"balancers": []any{
			map[string]any{
				"tag":      "main-balancer",
				"selector": []string{"vless-out-"},
				"strategy": map[string]any{"type": strategyType},
			},
		},
		"rules": []any{
			map[string]any{
				"type":        "field",
				"inboundTag":  routeInboundTags,
				"balancerTag": "main-balancer",
			},
		},
	}

	// Observatory is required for any multi-node balancer: xray-core's roundRobin strategy rotates
	// only among outbounds the observatory reports as alive. Without it the balancer silently
	// degrades to a single outbound (see issue #4). leastPing additionally relies on the measured
	// delay; roundRobin only needs the liveness signal.
	observatory = map[string]any{
		"subjectSelector":   []string{"vless-out-"},
		"probeUrl":          "http://www.google.com/generate_204",
		"probeInterval":     "10s",
		"enableConcurrency": true,
	}
	return
}

func streamSettingsInboundTProxy(_ int) map[string]any {
	return map[string]any{
		"sockopt": map[string]any{
			"tproxy": "tproxy",
		},
	}
}

// streamSettingsOutbound dispatches through the transport and security registries so that
// every emitted JSON block is sourced from a single per-transport/per-security
// implementation. The legacy hard-coded defaults (type=tcp, security=reality) are preserved
// inside the resolution step; the modern canonical defaults are introduced in a later
// commit alongside the VlessNode redesign.
func streamSettingsOutbound(node *domain.VlessNode) map[string]any {
	netName := node.Network
	if netName == "" {
		netName = "tcp"
	}
	secName := node.Security
	if secName == "" {
		secName = "reality"
	}

	tr, err := transport.Default.Resolve(netName)
	if err != nil {
		// Unknown transport: fall back to plain TCP so xray still receives a valid config
		// even if a node was created with a transport that has not been registered yet.
		tr, _ = transport.Default.Resolve("tcp")
	}
	tSpec := tr.SpecFromLegacyNode(node)
	tSettings, _ := tr.EmitSettings(tSpec)

	sec, err := security.Default.Resolve(secName)
	if err != nil {
		sec, _ = security.Default.Resolve("none")
	}
	sSpec := sec.SpecFromLegacyNode(node)
	sSettings, _ := sec.EmitSettings(sSpec)

	out := map[string]any{
		"network":  tr.Name(),
		"security": sec.Name(),
	}
	if len(tSettings) > 0 {
		out[tr.Name()+"Settings"] = tSettings
	}
	if len(sSettings) > 0 {
		out[sec.Name()+"Settings"] = sSettings
	}
	return out
}

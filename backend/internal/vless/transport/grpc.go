package transport

import (
	"strings"

	"xray2wg/backend/internal/domain"
)

// GRPCSpec captures gRPC transport parameters. xray-core accepts more fields (multiMode,
// authority, idleTimeout) but the legacy code only emitted serviceName; this shim keeps
// behavior identical until the canonicalization pass adds the rest.
type GRPCSpec struct {
	ServiceName string
}

type grpcTransport struct{}

func (grpcTransport) Name() string      { return "grpc" }
func (grpcTransport) Aliases() []string { return []string{"gun"} }

// ParseURI reads ?serviceName= when present; otherwise falls back to ?alpn= which is what
// the legacy code wrongly used as the service-name source for grpc nodes (and is what some
// existing subscription URIs in the wild encode it as). Preserved for bit-identity.
func (grpcTransport) ParseURI(ctx ParseCtx) (Spec, error) {
	svc := strings.TrimSpace(ctx.Query.Get("serviceName"))
	if svc == "" {
		svc = strings.TrimSpace(ctx.Query.Get("alpn"))
	}
	return GRPCSpec{ServiceName: svc}, nil
}

func (grpcTransport) Validate(spec Spec) error { return nil }

// EmitSettings emits grpcSettings.serviceName. If ServiceName is empty, the legacy code
// defaults to "GunService"; preserved here.
func (grpcTransport) EmitSettings(spec Spec) (map[string]any, error) {
	s := spec.(GRPCSpec)
	svc := s.ServiceName
	if svc == "" {
		svc = "GunService"
	}
	return map[string]any{"serviceName": svc}, nil
}

// ApplyToLegacyNode writes ServiceName into node.ALPN — the legacy storage location.
func (grpcTransport) ApplyToLegacyNode(spec Spec, n *domain.VlessNode) {
	s := spec.(GRPCSpec)
	if s.ServiceName != "" {
		n.ALPN = s.ServiceName
	}
}

func (grpcTransport) SpecFromLegacyNode(n *domain.VlessNode) Spec {
	return GRPCSpec{ServiceName: n.ALPN}
}

func init() { Default.Register(grpcTransport{}) }

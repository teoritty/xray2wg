package transport

import (
	"encoding/json"
	"net/url"
	"strings"
)

// GRPCSpec captures gRPC transport parameters. xray-core accepts more fields (multiMode,
// authority, idleTimeout); MultiMode is the most useful of those and is exposed.
type GRPCSpec struct {
	ServiceName string `json:"serviceName,omitempty"`
	MultiMode   bool   `json:"multiMode,omitempty"`
	Authority   string `json:"authority,omitempty"`
}

type grpcTransport struct{}

func (grpcTransport) Name() string      { return "grpc" }
func (grpcTransport) Aliases() []string { return []string{"gun"} }

// ParseURI reads ?serviceName= when present; otherwise falls back to ?alpn= which is what
// some existing subscription URIs in the wild encode it as. ?mode=multi enables multiMode.
func (grpcTransport) ParseURI(ctx ParseCtx) (Spec, error) {
	svc := strings.TrimSpace(ctx.Query.Get("serviceName"))
	if svc == "" {
		svc = strings.TrimSpace(ctx.Query.Get("alpn"))
	}
	return GRPCSpec{
		ServiceName: svc,
		MultiMode:   strings.EqualFold(strings.TrimSpace(ctx.Query.Get("mode")), "multi"),
		Authority:   strings.TrimSpace(ctx.Query.Get("authority")),
	}, nil
}

func (grpcTransport) Validate(spec Spec) error { return nil }

// EmitSettings emits grpcSettings. ServiceName defaults to "GunService" preserving the
// historical fallback used by gun-style URIs.
func (grpcTransport) EmitSettings(spec Spec) (map[string]any, error) {
	s := spec.(GRPCSpec)
	svc := s.ServiceName
	if svc == "" {
		svc = "GunService"
	}
	out := map[string]any{"serviceName": svc}
	if s.MultiMode {
		out["multiMode"] = true
	}
	if s.Authority != "" {
		out["authority"] = s.Authority
	}
	return out, nil
}

func (grpcTransport) EncodeSpec(spec Spec) (json.RawMessage, error) {
	return json.Marshal(spec.(GRPCSpec))
}

func (grpcTransport) DecodeSpec(data json.RawMessage) (Spec, error) {
	var s GRPCSpec
	if len(data) > 0 {
		if err := json.Unmarshal(data, &s); err != nil {
			return nil, err
		}
	}
	return s, nil
}

func (grpcTransport) ShareLink(spec Spec) (url.Values, error) {
	s := spec.(GRPCSpec)
	v := url.Values{}
	if s.ServiceName != "" {
		v.Set("serviceName", s.ServiceName)
	}
	if s.MultiMode {
		v.Set("mode", "multi")
	}
	if s.Authority != "" {
		v.Set("authority", s.Authority)
	}
	return v, nil
}

func init() { Default.Register(grpcTransport{}) }

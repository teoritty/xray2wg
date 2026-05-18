package transport

import (
	"strings"

	"xray2wg/backend/internal/domain"
)

// WSSpec captures WebSocket transport parameters: HTTP request path and Host header.
type WSSpec struct {
	Path string
	Host string
}

type wsTransport struct{}

func (wsTransport) Name() string      { return "ws" }
func (wsTransport) Aliases() []string { return []string{"websocket"} }

// ParseURI preserves the historical mapping: vless://...?path=&host=&sni=... — the URI
// "path" feeds the WS request path; "host" falls back into sni if sni is empty so the WS
// Host header matches the TLS SNI. This matches existing user URIs from common subscription
// providers.
func (wsTransport) ParseURI(ctx ParseCtx) (Spec, error) {
	return WSSpec{
		Path: strings.TrimSpace(ctx.Query.Get("path")),
		Host: strings.TrimSpace(ctx.Query.Get("host")),
	}, nil
}

func (wsTransport) Validate(spec Spec) error { return nil }

// EmitSettings produces the wsSettings JSON block xray-core expects. Default path "/" is
// used when the URI does not specify one or specifies a relative path that does not start
// with "/" (defensive: prevents xray rejecting the config).
func (wsTransport) EmitSettings(spec Spec) (map[string]any, error) {
	s := spec.(WSSpec)
	path := "/"
	if s.Path != "" && strings.HasPrefix(s.Path, "/") {
		path = s.Path
	}
	return map[string]any{
		"path":    path,
		"headers": map[string]any{"Host": s.Host},
	}, nil
}

// ApplyToLegacyNode and SpecFromLegacyNode preserve the historical encoding where the WS
// path is stored in node.SpiderX and the WS Host in node.SNI (with sniff-from-host
// fallback). These shim methods are deleted once VlessNode stops carrying flat transport
// columns.
func (wsTransport) ApplyToLegacyNode(spec Spec, n *domain.VlessNode) {
	s := spec.(WSSpec)
	if s.Path != "" {
		n.SpiderX = s.Path
	}
	if s.Host != "" && n.SNI == "" {
		n.SNI = s.Host
	}
}

func (wsTransport) SpecFromLegacyNode(n *domain.VlessNode) Spec {
	return WSSpec{Path: n.SpiderX, Host: n.SNI}
}

func init() { Default.Register(wsTransport{}) }

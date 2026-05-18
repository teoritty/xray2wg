package security

import (
	"strings"

	"xray2wg/backend/internal/domain"
)

// TLSSpec captures standard TLS parameters parseable from a vless:// share-link.
type TLSSpec struct {
	ServerName    string
	ALPN          []string
	Fingerprint   string
	AllowInsecure bool
}

type tlsSecurity struct{}

func (tlsSecurity) Name() string      { return "tls" }
func (tlsSecurity) Aliases() []string { return nil }

func (tlsSecurity) ParseURI(ctx ParseCtx) (Spec, error) {
	q := ctx.Query
	return TLSSpec{
		ServerName:  strings.TrimSpace(q.Get("sni")),
		ALPN:        splitALPN(q.Get("alpn")),
		Fingerprint: strings.TrimSpace(q.Get("fp")),
	}, nil
}

func splitALPN(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

func (tlsSecurity) Validate(spec Spec) error { return nil }

// EmitSettings reproduces the historical tlsSettings layout: hard-coded allowInsecure=false
// (commit 2 will surface AllowInsecure from the Spec) and ALPN as []any so json.Marshal
// emits a JSON array even when empty.
func (tlsSecurity) EmitSettings(spec Spec) (map[string]any, error) {
	s := spec.(TLSSpec)
	var alpnVals []any
	for _, a := range s.ALPN {
		alpnVals = append(alpnVals, a)
	}
	return map[string]any{
		"allowInsecure": s.AllowInsecure,
		"serverName":    s.ServerName,
		"alpn":          alpnVals,
	}, nil
}

func (tlsSecurity) ApplyToLegacyNode(spec Spec, n *domain.VlessNode) {
	s := spec.(TLSSpec)
	if s.ServerName != "" {
		n.SNI = s.ServerName
	}
	if s.Fingerprint != "" {
		n.Fingerprint = s.Fingerprint
	}
	if len(s.ALPN) > 0 {
		n.ALPN = strings.Join(s.ALPN, ",")
	}
}

func (tlsSecurity) SpecFromLegacyNode(n *domain.VlessNode) Spec {
	return TLSSpec{
		ServerName:  n.SNI,
		ALPN:        splitALPN(n.ALPN),
		Fingerprint: n.Fingerprint,
	}
}

func init() { Default.Register(tlsSecurity{}) }

package security

import (
	"strings"

	"xray2wg/backend/internal/domain"
)

// RealitySpec carries the REALITY-specific fields exchanged over a vless:// share-link.
type RealitySpec struct {
	ServerName  string
	Fingerprint string
	PublicKey   string
	ShortID     string
	SpiderX     string
}

type realitySecurity struct{}

func (realitySecurity) Name() string      { return "reality" }
func (realitySecurity) Aliases() []string { return nil }

func (realitySecurity) ParseURI(ctx ParseCtx) (Spec, error) {
	q := ctx.Query
	return RealitySpec{
		ServerName:  strings.TrimSpace(q.Get("sni")),
		Fingerprint: strings.TrimSpace(q.Get("fp")),
		PublicKey:   strings.TrimSpace(q.Get("pbk")),
		ShortID:     strings.TrimSpace(q.Get("sid")),
		SpiderX:     strings.TrimSpace(q.Get("spx")),
	}, nil
}

func (realitySecurity) Validate(spec Spec) error { return nil }

// EmitSettings reproduces the historical realitySettings layout with the same defaults: fp
// falls back to "chrome", spiderX falls back to "/".
func (realitySecurity) EmitSettings(spec Spec) (map[string]any, error) {
	s := spec.(RealitySpec)
	fp := s.Fingerprint
	if fp == "" {
		fp = "chrome"
	}
	spider := s.SpiderX
	if spider == "" {
		spider = "/"
	}
	return map[string]any{
		"fingerprint": fp,
		"serverName":  s.ServerName,
		"publicKey":   s.PublicKey,
		"shortId":     s.ShortID,
		"spiderX":     spider,
	}, nil
}

func (realitySecurity) ApplyToLegacyNode(spec Spec, n *domain.VlessNode) {
	s := spec.(RealitySpec)
	if s.ServerName != "" {
		n.SNI = s.ServerName
	}
	if s.Fingerprint != "" {
		n.Fingerprint = s.Fingerprint
	}
	if s.PublicKey != "" {
		n.PublicKey = s.PublicKey
	}
	if s.ShortID != "" {
		n.ShortID = s.ShortID
	}
	if s.SpiderX != "" {
		n.SpiderX = s.SpiderX
	}
}

func (realitySecurity) SpecFromLegacyNode(n *domain.VlessNode) Spec {
	return RealitySpec{
		ServerName:  n.SNI,
		Fingerprint: n.Fingerprint,
		PublicKey:   n.PublicKey,
		ShortID:     n.ShortID,
		SpiderX:     n.SpiderX,
	}
}

func init() { Default.Register(realitySecurity{}) }

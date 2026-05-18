package security

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
)

// RealitySpec carries the REALITY-specific fields exchanged over a vless:// share-link.
type RealitySpec struct {
	ServerName  string `json:"serverName,omitempty"`
	Fingerprint string `json:"fingerprint,omitempty"`
	PublicKey   string `json:"publicKey,omitempty"`
	ShortID     string `json:"shortId,omitempty"`
	SpiderX     string `json:"spiderX,omitempty"`
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

// Validate refuses a REALITY spec without a public key: the protocol cannot work without it
// and a silent failure at tunnel-start time is worse than an explicit error at create time.
func (realitySecurity) Validate(spec Spec) error {
	s := spec.(RealitySpec)
	if s.PublicKey == "" {
		return fmt.Errorf("reality: missing required pbk (publicKey)")
	}
	return nil
}

// EmitSettings reproduces the realitySettings layout with the same defaults: fp falls back
// to "chrome", spiderX to "/".
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

func (realitySecurity) EncodeSpec(spec Spec) (json.RawMessage, error) {
	return json.Marshal(spec.(RealitySpec))
}

func (realitySecurity) DecodeSpec(data json.RawMessage) (Spec, error) {
	var s RealitySpec
	if len(data) > 0 {
		if err := json.Unmarshal(data, &s); err != nil {
			return nil, err
		}
	}
	return s, nil
}

func (realitySecurity) ShareLink(spec Spec) (url.Values, error) {
	s := spec.(RealitySpec)
	v := url.Values{}
	if s.ServerName != "" {
		v.Set("sni", s.ServerName)
	}
	if s.Fingerprint != "" {
		v.Set("fp", s.Fingerprint)
	}
	if s.PublicKey != "" {
		v.Set("pbk", s.PublicKey)
	}
	if s.ShortID != "" {
		v.Set("sid", s.ShortID)
	}
	if s.SpiderX != "" {
		v.Set("spx", s.SpiderX)
	}
	return v, nil
}

func init() { Default.Register(realitySecurity{}) }

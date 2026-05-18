package security

import (
	"encoding/json"
	"net/url"
	"strings"
)

// TLSSpec captures standard TLS parameters parseable from a vless:// share-link.
type TLSSpec struct {
	ServerName    string   `json:"serverName,omitempty"`
	ALPN          []string `json:"alpn,omitempty"`
	Fingerprint   string   `json:"fingerprint,omitempty"`
	AllowInsecure bool     `json:"allowInsecure,omitempty"`
}

type tlsSecurity struct{}

func (tlsSecurity) Name() string      { return "tls" }
func (tlsSecurity) Aliases() []string { return nil }

func (tlsSecurity) ParseURI(ctx ParseCtx) (Spec, error) {
	q := ctx.Query
	allow := false
	switch strings.ToLower(strings.TrimSpace(q.Get("allowInsecure"))) {
	case "1", "true", "yes":
		allow = true
	}
	return TLSSpec{
		ServerName:    strings.TrimSpace(q.Get("sni")),
		ALPN:          splitALPN(q.Get("alpn")),
		Fingerprint:   strings.TrimSpace(q.Get("fp")),
		AllowInsecure: allow,
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

// EmitSettings reproduces the tlsSettings layout xray-core expects. ALPN is serialized as
// []any so json.Marshal emits a JSON array; an empty slice serializes to null which xray
// accepts.
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

func (tlsSecurity) EncodeSpec(spec Spec) (json.RawMessage, error) {
	return json.Marshal(spec.(TLSSpec))
}

func (tlsSecurity) DecodeSpec(data json.RawMessage) (Spec, error) {
	var s TLSSpec
	if len(data) > 0 {
		if err := json.Unmarshal(data, &s); err != nil {
			return nil, err
		}
	}
	return s, nil
}

func (tlsSecurity) ShareLink(spec Spec) (url.Values, error) {
	s := spec.(TLSSpec)
	v := url.Values{}
	if s.ServerName != "" {
		v.Set("sni", s.ServerName)
	}
	if len(s.ALPN) > 0 {
		v.Set("alpn", strings.Join(s.ALPN, ","))
	}
	if s.Fingerprint != "" {
		v.Set("fp", s.Fingerprint)
	}
	if s.AllowInsecure {
		v.Set("allowInsecure", "1")
	}
	return v, nil
}

func init() { Default.Register(tlsSecurity{}) }

package transport

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
)

// KCPSpec captures the share-link surface of mKCP. The lower-level tuning knobs (MTU/TTI/
// buffer sizes) are not part of the standard vless:// URI and use sensible xray defaults
// in EmitSettings; users who need to tune them can override via structured node edit
// (added in a later commit).
type KCPSpec struct {
	HeaderType string `json:"headerType,omitempty"`
	Seed       string `json:"seed,omitempty"`
}

// allowedKCPHeaderTypes is the canonical list from xray-core docs; rejected values cause
// the upstream to silently misbehave so we reject them at create time.
var allowedKCPHeaderTypes = map[string]struct{}{
	"none":          {},
	"srtp":          {},
	"utp":           {},
	"wechat-video":  {},
	"dtls":          {},
	"wireguard":     {},
	"dns":           {},
}

type kcpTransport struct{}

func (kcpTransport) Name() string      { return "kcp" }
func (kcpTransport) Aliases() []string { return []string{"mkcp"} }

func (kcpTransport) ParseURI(ctx ParseCtx) (Spec, error) {
	return KCPSpec{
		HeaderType: strings.TrimSpace(ctx.Query.Get("headerType")),
		Seed:       strings.TrimSpace(ctx.Query.Get("seed")),
	}, nil
}

func (kcpTransport) Validate(spec Spec) error {
	s := spec.(KCPSpec)
	if s.HeaderType == "" {
		return nil
	}
	if _, ok := allowedKCPHeaderTypes[s.HeaderType]; !ok {
		allowed := make([]string, 0, len(allowedKCPHeaderTypes))
		for k := range allowedKCPHeaderTypes {
			allowed = append(allowed, k)
		}
		return fmt.Errorf("kcp: invalid headerType %q (allowed: %s)", s.HeaderType, strings.Join(allowed, ", "))
	}
	return nil
}

// EmitSettings produces a kcpSettings block with xray-recommended defaults for the timing
// and buffer knobs. The share-link standard does not carry these so applying the upstream
// defaults is the right thing.
func (kcpTransport) EmitSettings(spec Spec) (map[string]any, error) {
	s := spec.(KCPSpec)
	header := s.HeaderType
	if header == "" {
		header = "none"
	}
	out := map[string]any{
		"mtu":              1350,
		"tti":              20,
		"uplinkCapacity":   5,
		"downlinkCapacity": 20,
		"congestion":       false,
		"readBufferSize":   1,
		"writeBufferSize":  1,
		"header":           map[string]any{"type": header},
	}
	if s.Seed != "" {
		out["seed"] = s.Seed
	}
	return out, nil
}

func (kcpTransport) EncodeSpec(spec Spec) (json.RawMessage, error) {
	return json.Marshal(spec.(KCPSpec))
}

func (kcpTransport) DecodeSpec(data json.RawMessage) (Spec, error) {
	var s KCPSpec
	if len(data) > 0 {
		if err := json.Unmarshal(data, &s); err != nil {
			return nil, err
		}
	}
	return s, nil
}

func (kcpTransport) ShareLink(spec Spec) (url.Values, error) {
	s := spec.(KCPSpec)
	v := url.Values{}
	if s.HeaderType != "" {
		v.Set("headerType", s.HeaderType)
	}
	if s.Seed != "" {
		v.Set("seed", s.Seed)
	}
	return v, nil
}

func init() { Default.Register(kcpTransport{}) }

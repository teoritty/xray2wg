package transport

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
)

// XHTTPSpec captures parameters for the modern xhttp transport. Mode controls whether
// upload and download use the same HTTP connection ("stream-one") or split across two
// HTTP transactions ("packet-up", "stream-up"); "auto" lets xray pick. Extra is a
// passthrough JSON object that the share-link standard reserves for transport-specific
// tuning (downloadSettings, scMaxEachPostBytes, …) without expanding this struct.
type XHTTPSpec struct {
	Path  string          `json:"path,omitempty"`
	Host  string          `json:"host,omitempty"`
	Mode  string          `json:"mode,omitempty"`
	Extra json.RawMessage `json:"extra,omitempty"`
}

type xhttpTransport struct{}

func (xhttpTransport) Name() string      { return "xhttp" }
func (xhttpTransport) Aliases() []string { return []string{"splithttp"} }

func (xhttpTransport) ParseURI(ctx ParseCtx) (Spec, error) {
	q := ctx.Query
	mode := strings.TrimSpace(q.Get("mode"))
	if mode == "" {
		mode = "auto"
	}
	var extra json.RawMessage
	if raw := strings.TrimSpace(q.Get("extra")); raw != "" {
		// extra is allowed to be either a JSON object or a base64-encoded JSON object per
		// the in-the-wild URI conventions; only accept it if it parses as valid JSON to
		// avoid storing garbage.
		if json.Valid([]byte(raw)) {
			extra = json.RawMessage(raw)
		}
	}
	return XHTTPSpec{
		Path:  strings.TrimSpace(q.Get("path")),
		Host:  strings.TrimSpace(q.Get("host")),
		Mode:  mode,
		Extra: extra,
	}, nil
}

func (xhttpTransport) Validate(spec Spec) error {
	s := spec.(XHTTPSpec)
	switch s.Mode {
	case "", "auto", "packet-up", "stream-up", "stream-one":
		return nil
	default:
		return fmt.Errorf("xhttp: invalid mode %q (allowed: auto, packet-up, stream-up, stream-one)", s.Mode)
	}
}

func (xhttpTransport) EmitSettings(spec Spec) (map[string]any, error) {
	s := spec.(XHTTPSpec)
	path := s.Path
	if path == "" {
		path = "/"
	}
	out := map[string]any{
		"path": path,
		"mode": defaultString(s.Mode, "auto"),
	}
	if s.Host != "" {
		out["host"] = s.Host
	}
	if len(s.Extra) > 0 {
		var extraMap map[string]any
		if err := json.Unmarshal(s.Extra, &extraMap); err == nil {
			out["extra"] = extraMap
		}
	}
	return out, nil
}

func (xhttpTransport) EncodeSpec(spec Spec) (json.RawMessage, error) {
	return json.Marshal(spec.(XHTTPSpec))
}

func (xhttpTransport) DecodeSpec(data json.RawMessage) (Spec, error) {
	var s XHTTPSpec
	if len(data) > 0 {
		if err := json.Unmarshal(data, &s); err != nil {
			return nil, err
		}
	}
	return s, nil
}

func (xhttpTransport) ShareLink(spec Spec) (url.Values, error) {
	s := spec.(XHTTPSpec)
	v := url.Values{}
	if s.Path != "" {
		v.Set("path", s.Path)
	}
	if s.Host != "" {
		v.Set("host", s.Host)
	}
	if s.Mode != "" && s.Mode != "auto" {
		v.Set("mode", s.Mode)
	}
	if len(s.Extra) > 0 {
		v.Set("extra", string(s.Extra))
	}
	return v, nil
}

func defaultString(v, def string) string {
	if v == "" {
		return def
	}
	return v
}

func init() { Default.Register(xhttpTransport{}) }

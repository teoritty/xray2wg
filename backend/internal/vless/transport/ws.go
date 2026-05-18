package transport

import (
	"encoding/json"
	"net/url"
	"strings"
)

// WSSpec captures WebSocket transport parameters: HTTP request path and Host header.
type WSSpec struct {
	Path string `json:"path,omitempty"`
	Host string `json:"host,omitempty"`
}

type wsTransport struct{}

func (wsTransport) Name() string      { return "ws" }
func (wsTransport) Aliases() []string { return []string{"websocket"} }

// ParseURI preserves the historical mapping: vless://...?path=&host=&sni=... — the URI
// "path" feeds the WS request path; "host" is the WS Host header.
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

func (wsTransport) EncodeSpec(spec Spec) (json.RawMessage, error) {
	return json.Marshal(spec.(WSSpec))
}

func (wsTransport) DecodeSpec(data json.RawMessage) (Spec, error) {
	var s WSSpec
	if len(data) > 0 {
		if err := json.Unmarshal(data, &s); err != nil {
			return nil, err
		}
	}
	return s, nil
}

func (wsTransport) ShareLink(spec Spec) (url.Values, error) {
	s := spec.(WSSpec)
	v := url.Values{}
	if s.Path != "" {
		v.Set("path", s.Path)
	}
	if s.Host != "" {
		v.Set("host", s.Host)
	}
	return v, nil
}

func init() { Default.Register(wsTransport{}) }

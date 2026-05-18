package transport

import (
	"encoding/json"
	"net/url"
	"strings"
)

// HTTPUpgradeSpec captures the parameters for the httpupgrade transport, a CDN-friendly
// HTTP/1.1 Upgrade-based stream (closer to WebSocket than to xhttp).
type HTTPUpgradeSpec struct {
	Path    string            `json:"path,omitempty"`
	Host    string            `json:"host,omitempty"`
	Headers map[string]string `json:"headers,omitempty"`
}

type httpUpgradeTransport struct{}

func (httpUpgradeTransport) Name() string      { return "httpupgrade" }
func (httpUpgradeTransport) Aliases() []string { return nil }

func (httpUpgradeTransport) ParseURI(ctx ParseCtx) (Spec, error) {
	q := ctx.Query
	return HTTPUpgradeSpec{
		Path: strings.TrimSpace(q.Get("path")),
		Host: strings.TrimSpace(q.Get("host")),
	}, nil
}

func (httpUpgradeTransport) Validate(spec Spec) error { return nil }

func (httpUpgradeTransport) EmitSettings(spec Spec) (map[string]any, error) {
	s := spec.(HTTPUpgradeSpec)
	path := s.Path
	if path == "" {
		path = "/"
	}
	out := map[string]any{"path": path}
	if s.Host != "" {
		out["host"] = s.Host
	}
	if len(s.Headers) > 0 {
		headers := make(map[string]any, len(s.Headers))
		for k, v := range s.Headers {
			headers[k] = v
		}
		out["headers"] = headers
	}
	return out, nil
}

func (httpUpgradeTransport) EncodeSpec(spec Spec) (json.RawMessage, error) {
	return json.Marshal(spec.(HTTPUpgradeSpec))
}

func (httpUpgradeTransport) DecodeSpec(data json.RawMessage) (Spec, error) {
	var s HTTPUpgradeSpec
	if len(data) > 0 {
		if err := json.Unmarshal(data, &s); err != nil {
			return nil, err
		}
	}
	return s, nil
}

func (httpUpgradeTransport) ShareLink(spec Spec) (url.Values, error) {
	s := spec.(HTTPUpgradeSpec)
	v := url.Values{}
	if s.Path != "" {
		v.Set("path", s.Path)
	}
	if s.Host != "" {
		v.Set("host", s.Host)
	}
	return v, nil
}

func init() { Default.Register(httpUpgradeTransport{}) }

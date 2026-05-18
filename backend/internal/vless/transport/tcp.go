package transport

import (
	"encoding/json"
	"net/url"
)

// TCPSpec carries no parameters in the current code path. Header obfuscation (header.type)
// could be added later without breaking the registry contract.
type TCPSpec struct{}

type tcpTransport struct{}

func (tcpTransport) Name() string                              { return "tcp" }
func (tcpTransport) Aliases() []string                         { return nil }
func (tcpTransport) ParseURI(ctx ParseCtx) (Spec, error)       { return TCPSpec{}, nil }
func (tcpTransport) EmitSettings(spec Spec) (map[string]any, error) { return nil, nil }
func (tcpTransport) Validate(spec Spec) error                  { return nil }
func (tcpTransport) EncodeSpec(spec Spec) (json.RawMessage, error) {
	// {} is intentional (vs. null/empty) so callers can distinguish "configured, no fields"
	// from "never configured".
	return json.RawMessage("{}"), nil
}
func (tcpTransport) DecodeSpec(data json.RawMessage) (Spec, error) { return TCPSpec{}, nil }
func (tcpTransport) ShareLink(spec Spec) (url.Values, error)       { return url.Values{}, nil }

func init() { Default.Register(tcpTransport{}) }

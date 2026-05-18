package transport

import "xray2wg/backend/internal/domain"

// TCPSpec carries no parameters in the current code path. Header obfuscation (header.type)
// could be added later without breaking the registry contract.
type TCPSpec struct{}

type tcpTransport struct{}

func (tcpTransport) Name() string      { return "tcp" }
func (tcpTransport) Aliases() []string { return nil }

func (tcpTransport) ParseURI(ctx ParseCtx) (Spec, error) {
	return TCPSpec{}, nil
}

func (tcpTransport) EmitSettings(spec Spec) (map[string]any, error) {
	// xray-core accepts a missing tcpSettings block; no JSON is emitted for plain TCP.
	return nil, nil
}

func (tcpTransport) Validate(spec Spec) error { return nil }

func (tcpTransport) ApplyToLegacyNode(spec Spec, n *domain.VlessNode) {}
func (tcpTransport) SpecFromLegacyNode(n *domain.VlessNode) Spec      { return TCPSpec{} }

func init() { Default.Register(tcpTransport{}) }

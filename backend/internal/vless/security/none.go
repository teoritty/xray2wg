package security

import "xray2wg/backend/internal/domain"

// NoneSpec carries no parameters.
type NoneSpec struct{}

type noneSecurity struct{}

func (noneSecurity) Name() string                                 { return "none" }
func (noneSecurity) Aliases() []string                            { return nil }
func (noneSecurity) ParseURI(ctx ParseCtx) (Spec, error)          { return NoneSpec{}, nil }
func (noneSecurity) EmitSettings(spec Spec) (map[string]any, error) {
	// No settings block; the streamSettings.security="none" line by itself is enough.
	return nil, nil
}
func (noneSecurity) Validate(spec Spec) error                       { return nil }
func (noneSecurity) ApplyToLegacyNode(spec Spec, n *domain.VlessNode) {}
func (noneSecurity) SpecFromLegacyNode(n *domain.VlessNode) Spec    { return NoneSpec{} }

func init() { Default.Register(noneSecurity{}) }

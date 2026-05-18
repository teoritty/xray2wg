package security

import (
	"encoding/json"
	"net/url"
)

// NoneSpec carries no parameters.
type NoneSpec struct{}

type noneSecurity struct{}

func (noneSecurity) Name() string                                   { return "none" }
func (noneSecurity) Aliases() []string                              { return nil }
func (noneSecurity) ParseURI(ctx ParseCtx) (Spec, error)            { return NoneSpec{}, nil }
func (noneSecurity) EmitSettings(spec Spec) (map[string]any, error) { return nil, nil }
func (noneSecurity) Validate(spec Spec) error                       { return nil }
func (noneSecurity) EncodeSpec(spec Spec) (json.RawMessage, error)  { return json.RawMessage("{}"), nil }
func (noneSecurity) DecodeSpec(data json.RawMessage) (Spec, error)  { return NoneSpec{}, nil }
func (noneSecurity) ShareLink(spec Spec) (url.Values, error)        { return url.Values{}, nil }

func init() { Default.Register(noneSecurity{}) }

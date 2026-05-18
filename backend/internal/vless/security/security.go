// Package security is the registry of VLESS security implementations (none, tls, reality).
// Mirrors the design of vless/transport: registry built at init(), read-only on the hot path.
package security

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
)

type Spec any

type ParseCtx struct {
	Query url.Values
}

type Security interface {
	Name() string
	Aliases() []string
	ParseURI(ctx ParseCtx) (Spec, error)
	EmitSettings(spec Spec) (map[string]any, error)
	Validate(spec Spec) error
	EncodeSpec(spec Spec) (json.RawMessage, error)
	DecodeSpec(data json.RawMessage) (Spec, error)
	ShareLink(spec Spec) (url.Values, error)
}

type Registry struct {
	byName  map[string]Security
	aliases map[string]string
}

func NewRegistry() *Registry {
	return &Registry{
		byName:  map[string]Security{},
		aliases: map[string]string{},
	}
}

func (r *Registry) Register(s Security) {
	name := strings.ToLower(s.Name())
	if name == "" {
		panic("vless/security: empty Name()")
	}
	if _, exists := r.byName[name]; exists {
		panic("vless/security: duplicate security " + name)
	}
	r.byName[name] = s
	for _, a := range s.Aliases() {
		a = strings.ToLower(a)
		if a == "" || a == name {
			continue
		}
		if _, exists := r.aliases[a]; exists {
			panic("vless/security: duplicate alias " + a)
		}
		r.aliases[a] = name
	}
}

func (r *Registry) Resolve(name string) (Security, error) {
	key := strings.ToLower(strings.TrimSpace(name))
	if s, ok := r.byName[key]; ok {
		return s, nil
	}
	if canon, ok := r.aliases[key]; ok {
		return r.byName[canon], nil
	}
	return nil, fmt.Errorf("vless/security: unknown %q (supported: %s)", name, strings.Join(r.Names(), ", "))
}

func (r *Registry) Names() []string {
	out := make([]string, 0, len(r.byName))
	for n := range r.byName {
		out = append(out, n)
	}
	return out
}

var Default = NewRegistry()

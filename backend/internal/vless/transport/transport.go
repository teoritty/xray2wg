// Package transport is the registry of VLESS stream-transport implementations. Each transport
// (raw/tcp, ws, grpc, xhttp, httpupgrade, kcp, …) is one struct that implements Transport and
// registers itself via init(); after process start the registry is read-only and free of
// locks on the hot path (URI parsing, xray-config emission).
package transport

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
)

// Spec is the transport-specific decoded parameter struct (WSSpec, GRPCSpec, …). Each
// Transport implementation defines its own concrete Spec type and type-asserts on it inside
// its methods.
type Spec any

// ParseCtx carries shared parsing context — query params plus already-decoded address/port —
// so transports do not have to re-parse the URL themselves.
type ParseCtx struct {
	Address string
	Port    int
	Query   url.Values
}

// Transport is the contract every stream transport must satisfy.
type Transport interface {
	// Name returns the canonical lowercase identifier (e.g. "ws", "grpc", "xhttp"). It is
	// also used as the streamSettings key prefix: "<name>Settings".
	Name() string
	// Aliases returns alternate URI ?type= values that map to this transport (e.g. ws also
	// matches "websocket"). All returned strings must be lowercase.
	Aliases() []string
	// ParseURI builds a Spec from the URI query.
	ParseURI(ctx ParseCtx) (Spec, error)
	// EmitSettings produces the JSON object that lands under streamSettings.<name>Settings.
	// Return an empty map if the transport has no settings to emit (e.g. plain TCP).
	EmitSettings(spec Spec) (map[string]any, error)
	// Validate fails fast on semantically broken Spec values (e.g. missing required field).
	Validate(spec Spec) error
	// EncodeSpec marshals a Spec to JSON for persistence in VlessNode.TransportConfig.
	EncodeSpec(spec Spec) (json.RawMessage, error)
	// DecodeSpec parses a previously-encoded Spec JSON back into a concrete Spec. An empty
	// payload returns the zero-value Spec, which is the right thing for transports whose
	// only configuration is implicit defaults (e.g. plain TCP).
	DecodeSpec(data json.RawMessage) (Spec, error)
	// ShareLink produces the subset of vless:// query parameters that re-encode the Spec.
	// Used when rebuilding a canonical RawURI after a structured edit.
	ShareLink(spec Spec) (url.Values, error)
}

// Registry is a name+alias lookup for Transports. After all init()s have run the registry is
// effectively immutable; callers MUST NOT register entries after process start.
type Registry struct {
	byName  map[string]Transport
	aliases map[string]string
}

// NewRegistry returns an empty Registry. Tests use this to build isolated registries; the
// production registry is the package-level Default.
func NewRegistry() *Registry {
	return &Registry{
		byName:  map[string]Transport{},
		aliases: map[string]string{},
	}
}

// Register adds t to the registry under its canonical name and every alias. Duplicate names
// or aliases panic — this is a programming error caught at init().
func (r *Registry) Register(t Transport) {
	name := strings.ToLower(t.Name())
	if name == "" {
		panic("vless/transport: empty Name()")
	}
	if _, exists := r.byName[name]; exists {
		panic("vless/transport: duplicate transport " + name)
	}
	r.byName[name] = t
	for _, a := range t.Aliases() {
		a = strings.ToLower(a)
		if a == "" || a == name {
			continue
		}
		if _, exists := r.aliases[a]; exists {
			panic("vless/transport: duplicate alias " + a)
		}
		r.aliases[a] = name
	}
}

// Resolve returns the transport registered under the given name or any of its aliases. The
// lookup is case-insensitive.
func (r *Registry) Resolve(name string) (Transport, error) {
	key := strings.ToLower(strings.TrimSpace(name))
	if t, ok := r.byName[key]; ok {
		return t, nil
	}
	if canon, ok := r.aliases[key]; ok {
		return r.byName[canon], nil
	}
	return nil, fmt.Errorf("vless/transport: unknown %q (supported: %s)", name, strings.Join(r.Names(), ", "))
}

// Names returns canonical transport names in stable lexicographic-ish insertion-order-stable
// fashion for use in error messages.
func (r *Registry) Names() []string {
	out := make([]string, 0, len(r.byName))
	for n := range r.byName {
		out = append(out, n)
	}
	return out
}

// Default is the process-wide registry. Each transport file registers itself in its init().
var Default = NewRegistry()

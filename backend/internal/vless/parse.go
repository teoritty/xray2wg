package vless

import (
	"net/url"
	"strconv"
	"strings"

	"xray2wg/backend/internal/domain"
	"xray2wg/backend/internal/vless/security"
	"xray2wg/backend/internal/vless/transport"
)

// ParseURI parses a vless:// outbound URI into a VlessNode.
//
// Transport- and security-specific parsing is delegated to the registries in
// vless/transport and vless/security so that adding a new transport or security mode does
// not require touching this dispatcher.
func ParseURI(raw string) (*domain.VlessNode, error) {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return nil, err
	}
	if strings.ToLower(u.Scheme) != "vless" {
		return nil, domain.ErrValidation
	}

	user := u.User
	if user == nil || user.Username() == "" {
		return nil, domain.ErrValidation
	}
	port := 443
	if p := u.Port(); p != "" {
		port, err = strconv.Atoi(p)
		if err != nil {
			return nil, err
		}
	}
	addr := u.Hostname()
	if addr == "" {
		return nil, domain.ErrValidation
	}
	q := u.Query()

	// Defaults preserve the pre-registry behavior: type=tcp, security=reality. These are
	// changed in a later commit when the canonicalization pass switches to the modern
	// xray-core defaults; here we are bit-identical with the legacy implementation.
	netName := q.Get("type")
	if netName == "" {
		netName = "tcp"
	}
	secName := q.Get("security")
	if secName == "" {
		secName = "reality"
	}

	tr, err := transport.Default.Resolve(netName)
	if err != nil {
		return nil, err
	}
	tSpec, err := tr.ParseURI(transport.ParseCtx{Address: addr, Port: port, Query: q})
	if err != nil {
		return nil, err
	}
	if err := tr.Validate(tSpec); err != nil {
		return nil, err
	}

	sec, err := security.Default.Resolve(secName)
	if err != nil {
		return nil, err
	}
	sSpec, err := sec.ParseURI(security.ParseCtx{Query: q})
	if err != nil {
		return nil, err
	}
	if err := sec.Validate(sSpec); err != nil {
		return nil, err
	}

	display := u.Fragment
	if display == "" {
		display = addr
	}

	// Pre-populate the legacy flat fields directly from the URI for storage compatibility
	// (some fields — e.g. alpn — are not consumed by every transport/security combination
	// but are still serialized into the DB by callers of the pre-registry code). The
	// per-transport / per-security ApplyToLegacyNode hooks may then override these values.
	node := &domain.VlessNode{
		UUID:        user.Username(),
		Address:     addr,
		Port:        port,
		Flow:        strings.TrimSpace(q.Get("flow")),
		Network:     tr.Name(),
		Security:    sec.Name(),
		SNI:         strings.TrimSpace(q.Get("sni")),
		Fingerprint: strings.TrimSpace(q.Get("fp")),
		PublicKey:   strings.TrimSpace(q.Get("pbk")),
		ShortID:     strings.TrimSpace(q.Get("sid")),
		SpiderX:     strings.TrimSpace(q.Get("spx")),
		ALPN:        strings.TrimSpace(q.Get("alpn")),
		DisplayName: display,
		RawURI:      raw,
	}
	sec.ApplyToLegacyNode(sSpec, node)
	tr.ApplyToLegacyNode(tSpec, node)
	return node, nil
}

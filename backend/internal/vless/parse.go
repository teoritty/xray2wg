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
// not require touching this dispatcher. The transport and security parameters are stored
// as opaque JSON in node.TransportConfig / node.SecurityConfig.
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

	// xray-core 2026 defaults: when ?type= or ?security= is missing, fall back to plain TCP
	// and no security. This differs from the pre-registry parser (which defaulted security
	// to "reality") because the modern xray docs treat "none" as the unambiguous default;
	// a REALITY URI always specifies security=reality explicitly.
	netName := q.Get("type")
	if netName == "" {
		netName = "tcp"
	}
	secName := q.Get("security")
	if secName == "" {
		secName = "none"
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
	tJSON, err := tr.EncodeSpec(tSpec)
	if err != nil {
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
	sJSON, err := sec.EncodeSpec(sSpec)
	if err != nil {
		return nil, err
	}

	display := u.Fragment
	if display == "" {
		display = addr
	}

	enc := strings.TrimSpace(q.Get("encryption"))
	if enc == "" {
		enc = "none"
	}
	pe := strings.TrimSpace(q.Get("packetEncoding"))

	return &domain.VlessNode{
		UUID:            user.Username(),
		Address:         addr,
		Port:            port,
		Flow:            strings.TrimSpace(q.Get("flow")),
		Encryption:      enc,
		PacketEncoding:  pe,
		Network:         tr.Name(),
		TransportConfig: tJSON,
		Security:        sec.Name(),
		SecurityConfig:  sJSON,
		DisplayName:     display,
		RawURI:          raw,
	}, nil
}

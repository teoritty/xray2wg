package vless

import (
	"net/url"
	"strconv"
	"strings"

	"xray2wg/backend/internal/domain"
)

// ParseURI parses a vless:// outbound URI into a VlessNode (subscription / manual node source).
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
	flow := strings.TrimSpace(q.Get("flow"))
	network := q.Get("type")
	if network == "" {
		network = "tcp"
	}
	security := q.Get("security")
	if security == "" {
		security = "reality"
	}
	display := u.Fragment
	if display == "" {
		display = addr
	}
	alpn := q.Get("alpn")
	spx := q.Get("spx")
	sni := q.Get("sni")
	if strings.EqualFold(network, "ws") || strings.EqualFold(network, "websocket") {
		if pt := q.Get("path"); pt != "" {
			spx = pt
		}
		if h := q.Get("host"); h != "" && sni == "" {
			sni = h
		}
	}
	return &domain.VlessNode{
		UUID:        user.Username(),
		Address:     addr,
		Port:        port,
		Flow:        flow,
		Network:     network,
		Security:    security,
		SNI:         sni,
		Fingerprint: q.Get("fp"),
		PublicKey:   q.Get("pbk"),
		ShortID:     q.Get("sid"),
		SpiderX:     spx,
		ALPN:        alpn,
		DisplayName: display,
		RawURI:      raw,
	}, nil
}

package vless

import (
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/url"
	"strconv"
	"strings"

	"xray2wg/backend/internal/domain"
	"xray2wg/backend/internal/vless/security"
	"xray2wg/backend/internal/vless/transport"
)

// BuildInput is the structured counterpart of a vless:// URI: every field a node carries,
// addressable individually for use by UI forms or external automation.
type BuildInput struct {
	DisplayName    string
	UUID           string
	Address        string
	Port           int
	Flow           string
	Encryption     string
	PacketEncoding string

	Network         string          // canonical transport name (or alias resolvable by the registry)
	TransportConfig json.RawMessage // marshaled TransportSpec
	Security        string          // "none" / "tls" / "reality"
	SecurityConfig  json.RawMessage // marshaled SecuritySpec
}

// Build constructs a fully-populated VlessNode from a structured input, validating the
// transport- and security-specific configs and re-encoding them to canonical JSON. RawURI
// is rebuilt via the Transport.ShareLink / Security.ShareLink hooks so a node round-trips
// between structured form and share-link without drift.
func Build(in BuildInput) (*domain.VlessNode, error) {
	if strings.TrimSpace(in.UUID) == "" {
		return nil, errors.New("vless.Build: uuid is required")
	}
	if strings.TrimSpace(in.Address) == "" {
		return nil, errors.New("vless.Build: address is required")
	}
	if in.Port <= 0 || in.Port > 65535 {
		return nil, fmt.Errorf("vless.Build: invalid port %d", in.Port)
	}

	netName := strings.TrimSpace(in.Network)
	if netName == "" {
		netName = "tcp"
	}
	tr, err := transport.Default.Resolve(netName)
	if err != nil {
		return nil, err
	}
	tSpec, err := tr.DecodeSpec(in.TransportConfig)
	if err != nil {
		return nil, fmt.Errorf("vless.Build: decode transport config: %w", err)
	}
	if err := tr.Validate(tSpec); err != nil {
		return nil, err
	}
	tJSON, err := tr.EncodeSpec(tSpec)
	if err != nil {
		return nil, err
	}

	secName := strings.TrimSpace(in.Security)
	if secName == "" {
		secName = "none"
	}
	sec, err := security.Default.Resolve(secName)
	if err != nil {
		return nil, err
	}
	sSpec, err := sec.DecodeSpec(in.SecurityConfig)
	if err != nil {
		return nil, fmt.Errorf("vless.Build: decode security config: %w", err)
	}
	if err := sec.Validate(sSpec); err != nil {
		return nil, err
	}
	sJSON, err := sec.EncodeSpec(sSpec)
	if err != nil {
		return nil, err
	}

	enc := strings.TrimSpace(in.Encryption)
	if enc == "" {
		enc = "none"
	}

	rawURI, err := buildRawURI(in, tr.Name(), sec.Name(), tr, tSpec, sec, sSpec, enc)
	if err != nil {
		return nil, err
	}

	display := strings.TrimSpace(in.DisplayName)
	if display == "" {
		display = in.Address
	}

	return &domain.VlessNode{
		UUID:            in.UUID,
		Address:         in.Address,
		Port:            in.Port,
		Flow:            strings.TrimSpace(in.Flow),
		Encryption:      enc,
		PacketEncoding:  strings.TrimSpace(in.PacketEncoding),
		Network:         tr.Name(),
		TransportConfig: tJSON,
		Security:        sec.Name(),
		SecurityConfig:  sJSON,
		DisplayName:     display,
		RawURI:          rawURI,
	}, nil
}

func buildRawURI(in BuildInput, netName, secName string,
	tr transport.Transport, tSpec transport.Spec,
	sec security.Security, sSpec security.Spec,
	encryption string,
) (string, error) {
	q := url.Values{}
	q.Set("type", netName)
	q.Set("security", secName)
	if encryption != "" && encryption != "none" {
		q.Set("encryption", encryption)
	}
	if pe := strings.TrimSpace(in.PacketEncoding); pe != "" {
		q.Set("packetEncoding", pe)
	}
	if f := strings.TrimSpace(in.Flow); f != "" {
		q.Set("flow", f)
	}
	tShare, err := tr.ShareLink(tSpec)
	if err != nil {
		return "", err
	}
	for k, vs := range tShare {
		for _, v := range vs {
			q.Set(k, v)
		}
	}
	sShare, err := sec.ShareLink(sSpec)
	if err != nil {
		return "", err
	}
	for k, vs := range sShare {
		for _, v := range vs {
			q.Set(k, v)
		}
	}
	u := url.URL{
		Scheme:   "vless",
		User:     url.User(in.UUID),
		Host:     net.JoinHostPort(in.Address, strconv.Itoa(in.Port)),
		RawQuery: q.Encode(),
		Fragment: in.DisplayName,
	}
	return u.String(), nil
}

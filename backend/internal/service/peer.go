package service

import (
	"context"
	"fmt"
	"net"
	"strings"

	"xray2wg/backend/internal/domain"
	cryptoutil "xray2wg/backend/internal/infrastructure/crypto"
	"xray2wg/backend/internal/wgkey"

	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
)

type PeerService struct {
	repo     domain.PeerRepository
	tunnels  domain.TunnelRepository
	master   []byte
	wg       WgTunnel
	tunnelOn func(int64) bool
}

func NewPeerService(repo domain.PeerRepository, tun domain.TunnelRepository, master []byte, wg WgTunnel, tunnelOn func(int64) bool) *PeerService {
	return &PeerService{repo: repo, tunnels: tun, master: master, wg: wg, tunnelOn: tunnelOn}
}

func (p *PeerService) PSKCallback(ctx context.Context, ifaceID int64) func(pub string) []byte {
	peers, err := p.repo.ListByInterface(ctx, ifaceID)
	if err != nil {
		return func(pub string) []byte { return nil }
	}
	m := make(map[string][]byte)
	for _, peer := range peers {
		_, _, pskEnc, err := p.repo.GetByID(ctx, ifaceID, peer.ID)
		if err != nil || pskEnc == "" {
			continue
		}
		raw, err := cryptoutil.DecryptGCM(p.master, pskEnc)
		if err != nil {
			continue
		}
		k, err := wgtypes.ParseKey(strings.TrimSpace(string(raw)))
		if err != nil {
			continue
		}
		b := []byte(k[:])
		m[strings.TrimSpace(peer.PublicKey)] = b
	}
	return func(pub string) []byte {
		return m[strings.TrimSpace(pub)]
	}
}

func (p *PeerService) NextClientIP(ctx context.Context, ifaceID int64, cidr string) (string, error) {
	_, ipnet, err := net.ParseCIDR(cidr)
	if err != nil {
		return "", err
	}
	ip := ipnet.IP.To4()
	if ip == nil {
		return "", fmt.Errorf("need ipv4 cidr")
	}
	prefix := ip[0:3]
	peers, err := p.repo.ListByInterface(ctx, ifaceID)
	if err != nil {
		return "", err
	}
	used := map[byte]struct{}{}
	for _, pr := range peers {
		a, _, err := net.ParseCIDR(pr.ClientAddress)
		if err != nil {
			continue
		}
		a4 := a.To4()
		if a4 == nil {
			continue
		}
		if a4[0] == prefix[0] && a4[1] == prefix[1] && a4[2] == prefix[2] {
			used[a4[3]] = struct{}{}
		}
	}
	for b := byte(2); b < 255; b++ {
		if _, ok := used[b]; ok {
			continue
		}
		return fmt.Sprintf("%d.%d.%d.%d/32", prefix[0], prefix[1], prefix[2], b), nil
	}
	return "", domain.ErrConflict
}

func (p *PeerService) Create(ctx context.Context, ifaceID int64, name string, pubKey string, clientIP string) (*domain.WgPeer, error) {
	iface, _, err := p.tunnels.GetByID(ctx, ifaceID)
	if err != nil {
		return nil, err
	}
	var privEnc *string
	if strings.TrimSpace(pubKey) == "" {
		priv, pub, err := wgkey.GenerateKeypair()
		if err != nil {
			return nil, err
		}
		pubKey = pub
		enc, err := cryptoutil.EncryptGCM(p.master, []byte(priv))
		if err != nil {
			return nil, err
		}
		privEnc = &enc
	}
	ip := clientIP
	if ip == "" {
		ip, err = p.NextClientIP(ctx, ifaceID, iface.WgAddress)
		if err != nil {
			return nil, err
		}
	}
	psk, err := wgkey.GeneratePSK()
	if err != nil {
		return nil, err
	}
	pskEnc, err := cryptoutil.EncryptGCM(p.master, []byte(psk))
	if err != nil {
		return nil, err
	}
	peer := &domain.WgPeer{
		Name:          name,
		PublicKey:     pubKey,
		ClientAddress: ip,
		// Gateway TPROXY is IPv4-only; ::/0 would send v6 into WG with no path through Xray.
		AllowedIPs:          "0.0.0.0/0",
		PersistentKeepalive: 25,
	}
	if err := p.repo.Create(ctx, ifaceID, peer, privEnc, &pskEnc); err != nil {
		return nil, err
	}
	if p.tunnelOn != nil && p.tunnelOn(ifaceID) && p.wg != nil {
		peers, _ := p.repo.ListByInterface(ctx, ifaceID)
		ifaceFull, sk, err := p.tunnels.GetByID(ctx, ifaceID)
		if err == nil {
			skBytes, _ := cryptoutil.DecryptGCM(p.master, sk)
			_ = p.wg.ReloadPeers(int(ifaceID), ifaceFull.MTU, ifaceFull.ListenPort, string(skBytes), peers, p.PSKCallback(ctx, ifaceID))
		}
	}
	return peer, nil
}

func (p *PeerService) ClientConfig(ctx context.Context, ifaceID, peerID int64) (string, error) {
	iface, _, err := p.tunnels.GetByID(ctx, ifaceID)
	if err != nil {
		return "", err
	}
	peer, privEnc, _, err := p.repo.GetByID(ctx, ifaceID, peerID)
	if err != nil {
		return "", err
	}
	if peer == nil || privEnc == "" {
		return "", domain.ErrValidation
	}
	privBytes, err := cryptoutil.DecryptGCM(p.master, privEnc)
	if err != nil {
		return "", err
	}
	_, _, pskEnc, err := p.repo.GetByID(ctx, ifaceID, peerID)
	pskStr := ""
	if err == nil && pskEnc != "" {
		if raw, err := cryptoutil.DecryptGCM(p.master, pskEnc); err == nil {
			pskStr = string(raw)
		}
	}
	// WG conf uses base64 in file - privBytes raw 32 decoded from storage - decrypt returns bytes of utf8 ASCII base64 of key
	// we stored plaintext key string encrypted — Decrypt yields original ASCII base64 wg key
	return wireGuardPeerClientIni(
		string(privBytes),
		peer.ClientAddress,
		iface.DNS,
		iface.MTU,
		iface.PublicKey,
		pskStr,
		peer.AllowedIPs,
		peer.PersistentKeepalive,
		iface.ListenPort,
	), nil
}

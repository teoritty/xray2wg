package infra

import (
	"xray2wg/backend/internal/domain"
	wginfra "xray2wg/backend/internal/infrastructure/wireguard"
)

// WgAdapter implements [service.WgTunnel] using [wginfra.Manager].
type WgAdapter struct {
	M *wginfra.Manager
}

func (a *WgAdapter) Create(tunnelID int, mtu int, listenPort int, privKey string, peers []*domain.WgPeer, pskForPub func(pub string) []byte, tunName string) error {
	return a.M.Create(tunnelID, mtu, listenPort, privKey, peers, pskForPub, tunName)
}

func (a *WgAdapter) Destroy(tunnelID int) {
	a.M.Destroy(tunnelID)
}

func (a *WgAdapter) ReloadPeers(tunnelID int, mtu int, listenPort int, privKey string, peers []*domain.WgPeer, pskForPub func(pub string) []byte) error {
	return a.M.ReloadPeers(tunnelID, mtu, listenPort, privKey, peers, pskForPub)
}

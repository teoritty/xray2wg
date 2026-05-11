package service

import "xray2wg/backend/internal/domain"

// XrayRunner is the injectable Xray sidecar contract (alias for historical naming in hardening plans).
type XrayRunner = XrayProcess

// WgRunner is the injectable WireGuard contract (alias for historical naming in hardening plans).
type WgRunner = WgTunnel

// XrayProcess runs an embedded Xray inbound for one tunnel (TPROXY path).
type XrayProcess interface {
	Start(tunnelID int, xrayPort int, fwmark int, localGatewayIP string, nodes []*domain.VlessNode, strategy domain.BalancingStrategy) error
	Stop(tunnelID int) error
	NodeHealth(tunnelID int) []domain.NodeHealthEntry
}

// WgTunnel manages a userspace WireGuard device for one tunnel.
type WgTunnel interface {
	Create(tunnelID int, mtu int, listenPort int, privKey string, peers []*domain.WgPeer, pskForPub func(pub string) []byte, tunName string) error
	Destroy(tunnelID int)
	ReloadPeers(tunnelID int, mtu int, listenPort int, privKey string, peers []*domain.WgPeer, pskForPub func(pub string) []byte) error
}

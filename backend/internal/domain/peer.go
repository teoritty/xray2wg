package domain

import (
	"context"
	"time"
)

type WgPeer struct {
	ID                 int64
	InterfaceID        int64
	Name               string
	PublicKey          string
	ClientAddress      string
	AllowedIPs         string
	PersistentKeepalive int
	LastHandshake      *time.Time
	RxBytes            int64
	TxBytes            int64
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

// PeerWithTunnel augments a peer with its WireGuard interface display name for global listings.
type PeerWithTunnel struct {
	WgPeer
	TunnelName string
}

type PeerRepository interface {
	Create(ctx context.Context, ifaceID int64, p *WgPeer, privKeyEnc, pskEnc *string) error
	GetByID(ctx context.Context, ifaceID int64, peerID int64) (*WgPeer, string, string, error) // peer, privKeyEnc, pskEnc
	ListByInterface(ctx context.Context, ifaceID int64) ([]*WgPeer, error)
	ListAllWithTunnel(ctx context.Context) ([]*PeerWithTunnel, error)
	Update(ctx context.Context, p *WgPeer, privKeyEnc, pskEnc *string) error
	Delete(ctx context.Context, ifaceID int64, peerID int64) error
	UpdateTraffic(ctx context.Context, peerID int64, lastHS *time.Time, rx, tx int64) error
	GetByPubKey(ctx context.Context, ifaceID int64, pubkey string) (*WgPeer, error)
}

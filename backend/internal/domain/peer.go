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
	// UpdateTraffic accumulates wireguard cumulative byte counters into the peer's running
	// totals. rxRaw/txRaw are the current cumulative values reported by wireguard-go; the
	// repository stores the previous raw values per peer and adds only the delta, so a
	// userspace device restart (which resets the wg counters to zero) no longer truncates
	// the dashboard totals. Returns the new accumulated rx/tx after the update.
	UpdateTraffic(ctx context.Context, peerID int64, lastHS *time.Time, rxRaw, txRaw int64) (accumRx, accumTx int64, err error)
	GetByPubKey(ctx context.Context, ifaceID int64, pubkey string) (*WgPeer, error)
}

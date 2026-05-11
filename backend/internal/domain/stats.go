package domain

import (
	"context"
	"time"
)

type StatSnapshot struct {
	ID           int64
	InterfaceID  *int64
	PeerID       *int64
	RxBytes      int64
	TxBytes      int64
	RxRate       int64
	TxRate       int64
	ActivePeers  *int32
	SampledAt    time.Time
}

type StatsRepository interface {
	Insert(ctx context.Context, s *StatSnapshot) error
	DeleteOlderThan(ctx context.Context, cutoff time.Time) error
	QueryInterfaceWindow(ctx context.Context, ifaceID int64, from, to time.Time) ([]StatSnapshot, error)
	QueryPeerWindow(ctx context.Context, peerID int64, from, to time.Time) ([]StatSnapshot, error)
	QueryLatestInterfaceRates(ctx context.Context) (map[int64][2]int64, error) // iface id -> rx,tx rate bytes/s
	DBCounts(ctx context.Context) (tunnelsRunning int64, totalPeers int64, totalRX int64, totalTX int64, err error)
}

package domain

import (
	"context"
	"time"
)

type WgStatus string

const (
	WgStatusRunning WgStatus = "running"
	WgStatusStopped WgStatus = "stopped"
	WgStatusError   WgStatus = "error"
)

type BalancingStrategy string

const (
	BalancingRoundRobin BalancingStrategy = "round_robin"
	BalancingLeastPing  BalancingStrategy = "least_ping"
)

// NodeHealthEntry reports the health of one VLESS outbound from the xray observatory.
type NodeHealthEntry struct {
	Tag     string // e.g. "vless-out-1"
	Alive   bool
	DelayMs int64 // -1 = unknown
}

type WgInterface struct {
	ID                int64
	Name              string
	TunName           string
	PublicKey         string
	ListenPort        int
	WgAddress         string
	DNS               string
	MTU               int
	SubscriptionID    *int64
	ActiveNodeID      *int64
	XrayPort          int
	FWMark            int
	Status            WgStatus
	ErrorMessage      string
	UptimeStarted     *time.Time
	BalancingStrategy BalancingStrategy
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

type TunnelRepository interface {
	Create(ctx context.Context, iface *WgInterface, privKeyEnc string) error
	GetByID(ctx context.Context, id int64) (*WgInterface, string, error)
	List(ctx context.Context) ([]*WgInterface, error)
	Update(ctx context.Context, iface *WgInterface) error
	// ClearActiveNodeID sets active_node_id to NULL (used when subscription changes without a new node).
	ClearActiveNodeID(ctx context.Context, id int64) error
	Delete(ctx context.Context, id int64) error
	UpdateStatus(ctx context.Context, id int64, status WgStatus, errMsg string) error
	UpdateRuntimeFields(ctx context.Context, id int64, tunName string, xrayPort, fwmark int) error
	ListRunningIDs(ctx context.Context) ([]int64, error)
	CountPeers(ctx context.Context, interfaceID int64) (int64, error)

	// SetNodes atomically replaces the ordered list of VLESS nodes for a tunnel.
	SetNodes(ctx context.Context, tunnelID int64, nodeIDs []int64) error
	// ListNodes returns nodes assigned to a tunnel ordered by position.
	ListNodes(ctx context.Context, tunnelID int64) ([]*VlessNode, error)
}

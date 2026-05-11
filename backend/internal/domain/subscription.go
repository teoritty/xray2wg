package domain

import (
	"context"
	"time"
)

type SubscriptionStatus string

const (
	SubStatusInactive SubscriptionStatus = "inactive"
	SubStatusActive   SubscriptionStatus = "active"
	SubStatusError    SubscriptionStatus = "error"
)

type Subscription struct {
	ID               int64
	Name             string
	URL              string
	RefreshInterval  int64
	LastFetchedAt    *time.Time
	NodeCount        int64
	Status           SubscriptionStatus
	ErrorMessage     string
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

// ActiveNodeBinding captures wg_interfaces.active_node → vless_nodes.raw_uri before a subscription refresh deletes nodes.
type ActiveNodeBinding struct {
	WgInterfaceID int64
	RawURI        string
}

type VlessNode struct {
	ID             int64
	SubscriptionID int64
	DisplayName    string
	UUID           string
	Address        string
	Port           int
	Flow           string
	Network        string
	Security       string
	SNI            string
	Fingerprint    string
	PublicKey      string
	ShortID        string
	SpiderX        string
	ALPN           string
	RawURI         string
	CreatedAt      time.Time
}

type SubscriptionRepository interface {
	Create(ctx context.Context, s *Subscription) error
	GetByID(ctx context.Context, id int64) (*Subscription, error)
	List(ctx context.Context) ([]*Subscription, error)
	Update(ctx context.Context, s *Subscription) error
	Delete(ctx context.Context, id int64) error
	DeleteNodes(ctx context.Context, subscriptionID int64) error
	SnapshotActiveNodesForSubscription(ctx context.Context, subscriptionID int64) ([]ActiveNodeBinding, error)
	RemapActiveNodesAfterRefresh(ctx context.Context, subscriptionID int64, bindings []ActiveNodeBinding, newNodes []*VlessNode) error
	InsertNodes(ctx context.Context, nodes []*VlessNode) error
	ListNodes(ctx context.Context, subscriptionID int64) ([]*VlessNode, error)
	ListAllNodes(ctx context.Context) ([]*VlessNode, error)
	GetNode(ctx context.Context, id int64) (*VlessNode, error)
	FindTunnelIDsUsingNode(ctx context.Context, nodeID int64) ([]int64, error)
	UpdateNode(ctx context.Context, n *VlessNode) error
	DeleteNode(ctx context.Context, id int64) error
}

package sqldb

import (
	"time"

	"gorm.io/gorm"
)

type Setting struct {
	Key       string `gorm:"primaryKey"`
	Value     string `gorm:"not null"`
	UpdatedAt time.Time
}

type SubscriptionRow struct {
	ID               int64  `gorm:"primaryKey;autoIncrement"`
	Name             string `gorm:"not null"`
	URL              string `gorm:"not null"`
	RefreshInterval  int64  `gorm:"not null;default:3600"`
	LastFetchedAt    *time.Time
	NodeCount        int64 `gorm:"default:0"`
	Status           string `gorm:"not null;default:'inactive'"`
	ErrorMessage     string
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

func (SubscriptionRow) TableName() string { return "subscriptions" }

type VlessNodeRow struct {
	ID             int64  `gorm:"primaryKey;autoIncrement"`
	SubscriptionID int64  `gorm:"not null;index:idx_vless_nodes_sub"`
	DisplayName    string `gorm:"not null"`
	UUID           string `gorm:"not null"`
	Address        string `gorm:"not null"`
	Port           int    `gorm:"not null"`
	Flow           string `gorm:"default:''"`
	Network        string `gorm:"default:'tcp'"`
	Security       string `gorm:"default:'reality'"`
	SNI            string
	Fingerprint    string
	PublicKey      string
	ShortID        string
	SpiderX        string
	Alpn           string
	RawURI         string `gorm:"not null"`
	CreatedAt      time.Time
}

func (VlessNodeRow) TableName() string { return "vless_nodes" }

type WgInterfaceRow struct {
	ID               int64  `gorm:"primaryKey;autoIncrement"`
	Name             string `gorm:"not null;uniqueIndex"`
	TunName          string
	PrivateKeyEnc    string `gorm:"not null"`
	PublicKey        string `gorm:"not null"`
	ListenPort       int    `gorm:"not null;uniqueIndex"`
	WgAddress        string `gorm:"not null"`
	DNS              string `gorm:"default:'1.1.1.1,8.8.8.8'"`
	MTU              int    `gorm:"default:1420"`
	SubscriptionID   *int64
	ActiveNodeID     *int64
	XrayPort         int
	FWMark           int
	Status           string `gorm:"not null;default:'stopped'"`
	ErrorMessage     string
	UptimeStartedAt  *time.Time
	BalancingStrategy string `gorm:"not null;default:'round_robin'"`
	CreatedAt        time.Time
	UpdatedAt        time.Time

	Subscription *SubscriptionRow `gorm:"foreignKey:SubscriptionID"`
	ActiveNode   *VlessNodeRow    `gorm:"foreignKey:ActiveNodeID"`
	Peers        []WgPeerRow      `gorm:"foreignKey:InterfaceID"`
	TunnelNodes  []TunnelNodeRow  `gorm:"foreignKey:InterfaceID"`
}

func (WgInterfaceRow) TableName() string { return "wg_interfaces" }

type TunnelNodeRow struct {
	ID          int64     `gorm:"primaryKey;autoIncrement"`
	InterfaceID int64     `gorm:"not null;index:idx_tunnel_nodes_iface"`
	NodeID      int64     `gorm:"not null"`
	Position    int       `gorm:"not null;default:0"`
	CreatedAt   time.Time

	Node *VlessNodeRow `gorm:"foreignKey:NodeID"`
}

func (TunnelNodeRow) TableName() string { return "tunnel_nodes" }

type WgPeerRow struct {
	ID                  int64  `gorm:"primaryKey;autoIncrement"`
	InterfaceID         int64  `gorm:"not null;index:idx_peers_interface"`
	Name                string `gorm:"not null"`
	PrivateKeyEnc       string
	PublicKey           string `gorm:"not null"`
	PresharedKeyEnc     string
	ClientAddress       string `gorm:"not null"`
	AllowedIPs          string `gorm:"default:'0.0.0.0/0'"`
	PersistentKeepalive int    `gorm:"default:25"`
	LastHandshake       *time.Time
	RxBytes             int64 `gorm:"default:0"`
	TxBytes             int64 `gorm:"default:0"`
	CreatedAt           time.Time
	UpdatedAt           time.Time

	Interface *WgInterfaceRow `gorm:"foreignKey:InterfaceID"`
}

func (WgPeerRow) TableName() string { return "wg_peers" }

type StatsSnapshotRow struct {
	ID           int64  `gorm:"primaryKey;autoIncrement"`
	InterfaceID  *int64
	PeerID       *int64
	RxBytes      int64 `gorm:"not null"`
	TxBytes      int64 `gorm:"not null"`
	RxRate       int64 `gorm:"not null"`
	TxRate       int64 `gorm:"not null"`
	ActivePeers *int32
	SampledAt   time.Time `gorm:"not null;autoCreateTime"`
}

func (StatsSnapshotRow) TableName() string { return "stats_snapshots" }

func AutoMigrate(conn *gorm.DB) error {
	if err := conn.AutoMigrate(&Setting{}, &SubscriptionRow{}, &VlessNodeRow{},
		&WgInterfaceRow{}, &WgPeerRow{}, &StatsSnapshotRow{}, &TunnelNodeRow{}); err != nil {
		return err
	}
	// Backfill: existing single-node tunnels → junction table.
	return conn.Exec(`
		INSERT OR IGNORE INTO tunnel_nodes (interface_id, node_id, position, created_at)
		SELECT id, active_node_id, 0, datetime('now')
		FROM wg_interfaces WHERE active_node_id IS NOT NULL
	`).Error
}

package sqldb

import (
	"strings"
	"time"

	"xray2wg/backend/internal/vless"

	"github.com/rs/zerolog/log"
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
	Encryption     string `gorm:"default:'none'"`
	PacketEncoding string `gorm:"default:''"`
	Network        string `gorm:"default:'tcp'"`
	// TransportConfig and SecurityConfig are the JSON-encoded transport/security parameter
	// structs. They are opaque to the DB and parsed by vless/transport and vless/security
	// at use time. Storing as TEXT keeps the schema schema-less and avoids a column
	// explosion every time a new transport is added.
	TransportConfig string `gorm:"type:text;not null;default:'{}'"`
	Security        string `gorm:"default:'none'"`
	SecurityConfig  string `gorm:"type:text;not null;default:'{}'"`
	RawURI          string `gorm:"not null"`
	CreatedAt       time.Time
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
	// LastSeenRxRaw / LastSeenTxRaw store the most recent cumulative counters reported by
	// wireguard-go for this peer. They let UpdateTraffic compute the delta since the prior
	// poll, so a wg device restart (which zeroes the cumulative counters) does not erase
	// the accumulated totals in rx_bytes / tx_bytes (issue #5).
	LastSeenRxRaw int64 `gorm:"not null;default:0"`
	LastSeenTxRaw int64 `gorm:"not null;default:0"`
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

type AuditLogRow struct {
	ID        int64     `gorm:"primaryKey;autoIncrement"`
	Level     string    `gorm:"not null;index"`
	Source    string    `gorm:"not null;default:'system'"`
	Message   string    `gorm:"not null"`
	CreatedAt time.Time `gorm:"not null;autoCreateTime;index"`
}

func (AuditLogRow) TableName() string { return "audit_logs" }

func AutoMigrate(conn *gorm.DB) error {
	if err := conn.AutoMigrate(&Setting{}, &SubscriptionRow{}, &VlessNodeRow{},
		&WgInterfaceRow{}, &WgPeerRow{}, &StatsSnapshotRow{}, &TunnelNodeRow{}, &AuditLogRow{}); err != nil {
		return err
	}
	// Backfill: existing single-node tunnels → junction table.
	if err := conn.Exec(`
		INSERT OR IGNORE INTO tunnel_nodes (interface_id, node_id, position, created_at)
		SELECT id, active_node_id, 0, datetime('now')
		FROM wg_interfaces WHERE active_node_id IS NOT NULL
	`).Error; err != nil {
		return err
	}
	// Migrate vless_nodes from flat columns to JSON config columns. Idempotent: only rows
	// where transport_config is empty or "{}" are re-parsed from raw_uri.
	if err := backfillVlessNodeConfigs(conn); err != nil {
		return err
	}
	// Drop the now-redundant flat columns. SQLite ≥ 3.35 supports DROP COLUMN; the bundled
	// modernc.org/sqlite driver ships SQLite ≥ 3.45 so this is always available in
	// production builds. The error is logged (not returned) defensively in case an
	// operator runs against an external SQLite they pointed us at.
	for _, col := range []string{"sni", "fingerprint", "public_key", "short_id", "spider_x", "alpn"} {
		if !columnExists(conn, "vless_nodes", col) {
			continue
		}
		if err := conn.Exec("ALTER TABLE vless_nodes DROP COLUMN " + col).Error; err != nil {
			log.Warn().Err(err).Str("column", col).Msg("AutoMigrate: DROP COLUMN failed; the legacy column will linger but is unused")
		}
	}
	return nil
}

func backfillVlessNodeConfigs(conn *gorm.DB) error {
	type row struct {
		ID     int64
		RawURI string
	}
	var rows []row
	if err := conn.Raw(`
		SELECT id, raw_uri FROM vless_nodes
		WHERE COALESCE(transport_config, '') IN ('', '{}')
		   OR COALESCE(security_config, '') IN ('', '{}')
	`).Scan(&rows).Error; err != nil {
		return err
	}
	for _, r := range rows {
		if strings.TrimSpace(r.RawURI) == "" {
			continue
		}
		node, err := vless.ParseURI(r.RawURI)
		if err != nil {
			log.Warn().Int64("node_id", r.ID).Err(err).Msg("AutoMigrate: cannot reparse vless raw_uri; row left at default config")
			continue
		}
		updates := map[string]any{
			"network":          node.Network,
			"security":         node.Security,
			"flow":             node.Flow,
			"encryption":       node.Encryption,
			"packet_encoding":  node.PacketEncoding,
			"transport_config": string(node.TransportConfig),
			"security_config":  string(node.SecurityConfig),
		}
		if err := conn.Exec(`
			UPDATE vless_nodes SET
				network = ?,
				security = ?,
				flow = ?,
				encryption = ?,
				packet_encoding = ?,
				transport_config = ?,
				security_config = ?
			WHERE id = ?
		`,
			updates["network"], updates["security"], updates["flow"], updates["encryption"],
			updates["packet_encoding"], updates["transport_config"], updates["security_config"],
			r.ID,
		).Error; err != nil {
			return err
		}
	}
	return nil
}

func columnExists(conn *gorm.DB, table, column string) bool {
	var count int64
	err := conn.Raw("SELECT COUNT(*) FROM pragma_table_info(?) WHERE name = ?", table, column).Scan(&count).Error
	return err == nil && count > 0
}

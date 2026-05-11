package sqldb

import (
	"context"

	"xray2wg/backend/internal/domain"

	"gorm.io/gorm"
)

// TunnelNodeRepo manages the tunnel_nodes junction table.
type TunnelNodeRepo struct {
	db *gorm.DB
}

func NewTunnelNodeRepo(db *gorm.DB) *TunnelNodeRepo {
	return &TunnelNodeRepo{db: db}
}

// SetNodes atomically replaces all node assignments for a tunnel.
func (r *TunnelNodeRepo) SetNodes(ctx context.Context, tunnelID int64, nodeIDs []int64) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("interface_id = ?", tunnelID).Delete(&TunnelNodeRow{}).Error; err != nil {
			return err
		}
		for i, nid := range nodeIDs {
			row := TunnelNodeRow{InterfaceID: tunnelID, NodeID: nid, Position: i}
			if err := tx.Create(&row).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

// ListNodes returns nodes assigned to a tunnel ordered by position.
func (r *TunnelNodeRepo) ListNodes(ctx context.Context, tunnelID int64) ([]*domain.VlessNode, error) {
	var rows []TunnelNodeRow
	if err := r.db.WithContext(ctx).
		Preload("Node").
		Where("interface_id = ?", tunnelID).
		Order("position asc").
		Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]*domain.VlessNode, 0, len(rows))
	for _, row := range rows {
		if row.Node != nil {
			out = append(out, rowNode(row.Node))
		}
	}
	return out, nil
}

// RemapNodesAfterRefresh updates tunnel_nodes entries for a subscription after its nodes are refreshed.
// For each existing junction entry whose node belonged to the refreshed subscription, it tries to
// find a new node with the same raw_uri. Entries without a match are removed.
func (r *TunnelNodeRepo) RemapNodesAfterRefresh(ctx context.Context, subID int64, newNodes []*domain.VlessNode) error {
	// Build raw_uri → new node id map.
	uriToID := make(map[string]int64, len(newNodes))
	for _, n := range newNodes {
		if n.SubscriptionID == subID {
			uriToID[n.RawURI] = n.ID
		}
	}

	// Find all tunnel_nodes rows whose node belongs to this subscription.
	var oldRows []TunnelNodeRow
	if err := r.db.WithContext(ctx).
		Joins("JOIN vless_nodes ON vless_nodes.id = tunnel_nodes.node_id").
		Where("vless_nodes.subscription_id = ?", subID).
		Find(&oldRows).Error; err != nil {
		return err
	}

	for _, row := range oldRows {
		// We don't have raw_uri here without preloading; do a targeted lookup.
		var nodeRow VlessNodeRow
		if err := r.db.WithContext(ctx).First(&nodeRow, row.NodeID).Error; err != nil {
			// Node already gone — delete junction entry.
			r.db.WithContext(ctx).Delete(&TunnelNodeRow{}, row.ID)
			continue
		}
		newID, ok := uriToID[nodeRow.RawURI]
		if !ok {
			// Node no longer in subscription — remove from tunnel.
			r.db.WithContext(ctx).Delete(&TunnelNodeRow{}, row.ID)
			continue
		}
		if newID != row.NodeID {
			r.db.WithContext(ctx).Model(&TunnelNodeRow{}).Where("id = ?", row.ID).Update("node_id", newID)
		}
	}
	return nil
}

package sqldb

import (
	"context"
	"errors"
	"strings"

	"xray2wg/backend/internal/domain"

	"gorm.io/gorm"
)

type SubscriptionRepo struct {
	db *gorm.DB
}

func NewSubscriptionRepo(db *gorm.DB) *SubscriptionRepo { return &SubscriptionRepo{db: db} }

var _ domain.SubscriptionRepository = (*SubscriptionRepo)(nil)

func rowSub(r *SubscriptionRow) *domain.Subscription {
	return &domain.Subscription{
		ID:               r.ID,
		Name:             r.Name,
		URL:              r.URL,
		RefreshInterval:  r.RefreshInterval,
		LastFetchedAt:    r.LastFetchedAt,
		NodeCount:        r.NodeCount,
		Status:           domain.SubscriptionStatus(r.Status),
		ErrorMessage:     r.ErrorMessage,
		CreatedAt:        r.CreatedAt,
		UpdatedAt:        r.UpdatedAt,
	}
}

func rowNode(n *VlessNodeRow) *domain.VlessNode {
	return &domain.VlessNode{
		ID:             n.ID,
		SubscriptionID: n.SubscriptionID,
		DisplayName:    n.DisplayName,
		UUID:           n.UUID,
		Address:        n.Address,
		Port:           n.Port,
		Flow:           n.Flow,
		Network:        n.Network,
		Security:       n.Security,
		SNI:            n.SNI,
		Fingerprint:    n.Fingerprint,
		PublicKey:      n.PublicKey,
		ShortID:        n.ShortID,
		SpiderX:        n.SpiderX,
		ALPN:           n.Alpn,
		RawURI:         n.RawURI,
		CreatedAt:      n.CreatedAt,
	}
}

func (r *SubscriptionRepo) Create(ctx context.Context, s *domain.Subscription) error {
	row := SubscriptionRow{
		Name:            s.Name,
		URL:             s.URL,
		RefreshInterval: s.RefreshInterval,
		Status:          string(domain.SubStatusInactive),
	}
	if err := r.db.WithContext(ctx).Create(&row).Error; err != nil {
		return err
	}
	s.ID = row.ID
	s.CreatedAt = row.CreatedAt
	s.UpdatedAt = row.UpdatedAt
	return nil
}

func (r *SubscriptionRepo) GetByID(ctx context.Context, id int64) (*domain.Subscription, error) {
	var row SubscriptionRow
	if err := r.db.WithContext(ctx).First(&row, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return rowSub(&row), nil
}

func (r *SubscriptionRepo) List(ctx context.Context) ([]*domain.Subscription, error) {
	var rows []SubscriptionRow
	if err := r.db.WithContext(ctx).Order("id asc").Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]*domain.Subscription, len(rows))
	for i := range rows {
		out[i] = rowSub(&rows[i])
	}
	return out, nil
}

func (r *SubscriptionRepo) Update(ctx context.Context, s *domain.Subscription) error {
	res := r.db.WithContext(ctx).Model(&SubscriptionRow{}).Where("id = ?", s.ID).Updates(map[string]any{
		"name":              s.Name,
		"url":               s.URL,
		"refresh_interval":  s.RefreshInterval,
		"last_fetched_at":   s.LastFetchedAt,
		"node_count":        s.NodeCount,
		"status":            string(s.Status),
		"error_message":     s.ErrorMessage,
	})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *SubscriptionRepo) Delete(ctx context.Context, id int64) error {
	res := r.db.WithContext(ctx).Delete(&SubscriptionRow{}, id)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *SubscriptionRepo) DeleteNodes(ctx context.Context, subscriptionID int64) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec(`
			UPDATE wg_interfaces SET active_node_id = NULL
			WHERE active_node_id IN (SELECT id FROM vless_nodes WHERE subscription_id = ?)
		`, subscriptionID).Error; err != nil {
			return err
		}
		return tx.Where("subscription_id = ?", subscriptionID).Delete(&VlessNodeRow{}).Error
	})
}

func (r *SubscriptionRepo) SnapshotActiveNodesForSubscription(ctx context.Context, subscriptionID int64) ([]domain.ActiveNodeBinding, error) {
	var rows []struct {
		WgID   int64  `gorm:"column:wg_id"`
		RawURI string `gorm:"column:raw_uri"`
	}
	err := r.db.WithContext(ctx).Raw(`
		SELECT w.id AS wg_id, n.raw_uri AS raw_uri
		FROM wg_interfaces w
		INNER JOIN vless_nodes n ON n.id = w.active_node_id
		WHERE w.subscription_id = ? AND w.active_node_id IS NOT NULL
	`, subscriptionID).Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	out := make([]domain.ActiveNodeBinding, 0, len(rows))
	for _, row := range rows {
		out = append(out, domain.ActiveNodeBinding{
			WgInterfaceID: row.WgID,
			RawURI:        strings.TrimSpace(row.RawURI),
		})
	}
	return out, nil
}

func (r *SubscriptionRepo) RemapActiveNodesAfterRefresh(ctx context.Context, subscriptionID int64, bindings []domain.ActiveNodeBinding, newNodes []*domain.VlessNode) error {
	if len(bindings) == 0 || len(newNodes) == 0 {
		return nil
	}
	firstByURI := make(map[string]int64)
	for _, n := range newNodes {
		if n == nil || n.ID == 0 {
			continue
		}
		key := strings.TrimSpace(n.RawURI)
		if key == "" {
			continue
		}
		if _, ok := firstByURI[key]; !ok {
			firstByURI[key] = n.ID
		}
	}
	for _, b := range bindings {
		newID := firstByURI[strings.TrimSpace(b.RawURI)]
		if newID == 0 {
			continue
		}
		if err := r.db.WithContext(ctx).Model(&WgInterfaceRow{}).
			Where("id = ? AND subscription_id = ?", b.WgInterfaceID, subscriptionID).
			Update("active_node_id", newID).Error; err != nil {
			return err
		}
	}
	return nil
}

func (r *SubscriptionRepo) InsertNodes(ctx context.Context, nodes []*domain.VlessNode) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for _, n := range nodes {
			row := VlessNodeRow{
				SubscriptionID: n.SubscriptionID,
				DisplayName:    n.DisplayName,
				UUID:           n.UUID,
				Address:        n.Address,
				Port:           n.Port,
				Flow:           n.Flow,
				Network:        n.Network,
				Security:       n.Security,
				SNI:            n.SNI,
				Fingerprint:    n.Fingerprint,
				PublicKey:      n.PublicKey,
				ShortID:        n.ShortID,
				SpiderX:        n.SpiderX,
				Alpn:           n.ALPN,
				RawURI:         n.RawURI,
			}
			if err := tx.Create(&row).Error; err != nil {
				return err
			}
			n.ID = row.ID
		}
		return nil
	})
}

func (r *SubscriptionRepo) ListNodes(ctx context.Context, subscriptionID int64) ([]*domain.VlessNode, error) {
	var rows []VlessNodeRow
	if err := r.db.WithContext(ctx).Where("subscription_id = ?", subscriptionID).Order("id asc").Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]*domain.VlessNode, len(rows))
	for i := range rows {
		out[i] = rowNode(&rows[i])
	}
	return out, nil
}

func (r *SubscriptionRepo) ListAllNodes(ctx context.Context) ([]*domain.VlessNode, error) {
	var rows []VlessNodeRow
	if err := r.db.WithContext(ctx).Order("id asc").Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]*domain.VlessNode, len(rows))
	for i := range rows {
		out[i] = rowNode(&rows[i])
	}
	return out, nil
}

func (r *SubscriptionRepo) GetNode(ctx context.Context, id int64) (*domain.VlessNode, error) {
	var row VlessNodeRow
	if err := r.db.WithContext(ctx).First(&row, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return rowNode(&row), nil
}

func (r *SubscriptionRepo) FindTunnelIDsUsingNode(ctx context.Context, nodeID int64) ([]int64, error) {
	var ids []int64
	err := r.db.WithContext(ctx).Model(&WgInterfaceRow{}).Where("active_node_id = ?", nodeID).Pluck("id", &ids).Error
	return ids, err
}

func (r *SubscriptionRepo) UpdateNode(ctx context.Context, n *domain.VlessNode) error {
	res := r.db.WithContext(ctx).Model(&VlessNodeRow{}).Where("id = ?", n.ID).Updates(map[string]any{
		"subscription_id": n.SubscriptionID,
		"display_name":    n.DisplayName,
		"uuid":            n.UUID,
		"address":         n.Address,
		"port":            n.Port,
		"flow":            n.Flow,
		"network":         n.Network,
		"security":        n.Security,
		"sni":             n.SNI,
		"fingerprint":     n.Fingerprint,
		"public_key":      n.PublicKey,
		"short_id":        n.ShortID,
		"spider_x":        n.SpiderX,
		"alpn":            n.ALPN,
		"raw_uri":         n.RawURI,
	})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *SubscriptionRepo) DeleteNode(ctx context.Context, id int64) error {
	res := r.db.WithContext(ctx).Delete(&VlessNodeRow{}, id)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return domain.ErrNotFound
	}
	return nil
}

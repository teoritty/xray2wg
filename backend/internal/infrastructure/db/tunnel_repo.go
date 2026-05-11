package sqldb

import (
	"context"
	"errors"

	"xray2wg/backend/internal/domain"

	"gorm.io/gorm"
)

type TunnelRepo struct {
	db    *gorm.DB
	nodes *TunnelNodeRepo
}

func NewTunnelRepo(db *gorm.DB) *TunnelRepo {
	return &TunnelRepo{db: db, nodes: NewTunnelNodeRepo(db)}
}

var _ domain.TunnelRepository = (*TunnelRepo)(nil)

func ifaceRow(r *WgInterfaceRow) *domain.WgInterface {
	i := &domain.WgInterface{
		ID:                r.ID,
		Name:              r.Name,
		TunName:           r.TunName,
		PublicKey:         r.PublicKey,
		ListenPort:        r.ListenPort,
		WgAddress:         r.WgAddress,
		DNS:               r.DNS,
		MTU:               r.MTU,
		XrayPort:          r.XrayPort,
		FWMark:            r.FWMark,
		Status:            domain.WgStatus(r.Status),
		ErrorMessage:      r.ErrorMessage,
		UptimeStarted:     r.UptimeStartedAt,
		BalancingStrategy: domain.BalancingStrategy(r.BalancingStrategy),
		CreatedAt:         r.CreatedAt,
		UpdatedAt:         r.UpdatedAt,
	}
	if r.SubscriptionID != nil {
		i.SubscriptionID = r.SubscriptionID
	}
	if r.ActiveNodeID != nil {
		i.ActiveNodeID = r.ActiveNodeID
	}
	return i
}

func (r *TunnelRepo) Create(ctx context.Context, iface *domain.WgInterface, privKeyEnc string) error {
	row := WgInterfaceRow{
		Name:           iface.Name,
		TunName:        iface.TunName,
		PrivateKeyEnc:  privKeyEnc,
		PublicKey:      iface.PublicKey,
		ListenPort:     iface.ListenPort,
		WgAddress:      iface.WgAddress,
		DNS:            iface.DNS,
		MTU:            iface.MTU,
		SubscriptionID: iface.SubscriptionID,
		ActiveNodeID:   iface.ActiveNodeID,
		XrayPort:       iface.XrayPort,
		FWMark:         iface.FWMark,
		Status:         string(iface.Status),
	}
	if err := r.db.WithContext(ctx).Create(&row).Error; err != nil {
		return err
	}
	iface.ID = row.ID
	iface.CreatedAt = row.CreatedAt
	iface.UpdatedAt = row.UpdatedAt
	return nil
}

func (r *TunnelRepo) GetByID(ctx context.Context, id int64) (*domain.WgInterface, string, error) {
	var row WgInterfaceRow
	if err := r.db.WithContext(ctx).First(&row, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, "", domain.ErrNotFound
		}
		return nil, "", err
	}
	return ifaceRow(&row), row.PrivateKeyEnc, nil
}

func (r *TunnelRepo) List(ctx context.Context) ([]*domain.WgInterface, error) {
	var rows []WgInterfaceRow
	if err := r.db.WithContext(ctx).Order("id asc").Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]*domain.WgInterface, len(rows))
	for i := range rows {
		out[i] = ifaceRow(&rows[i])
	}
	return out, nil
}

func (r *TunnelRepo) Update(ctx context.Context, iface *domain.WgInterface) error {
	// Do not pass active_node_id when nil: GORM Updates(map) would write SQL NULL and
	// wipe bindings on partial updates (e.g. status-only paths or PUT bodies omitting the field).
	m := map[string]any{
		"name":               iface.Name,
		"listen_port":        iface.ListenPort,
		"wg_address":         iface.WgAddress,
		"dns":                iface.DNS,
		"mtu":                iface.MTU,
		"subscription_id":    iface.SubscriptionID,
		"xray_port":          iface.XrayPort,
		"fw_mark":            iface.FWMark,
		"tun_name":           iface.TunName,
		"status":             string(iface.Status),
		"error_message":      iface.ErrorMessage,
		"uptime_started_at":  iface.UptimeStarted,
		"balancing_strategy": string(iface.BalancingStrategy),
	}
	if iface.ActiveNodeID != nil {
		m["active_node_id"] = iface.ActiveNodeID
	}
	res := r.db.WithContext(ctx).Model(&WgInterfaceRow{}).Where("id = ?", iface.ID).Updates(m)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *TunnelRepo) ClearActiveNodeID(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).Model(&WgInterfaceRow{}).Where("id = ?", id).Update("active_node_id", nil).Error
}

func (r *TunnelRepo) Delete(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var peerIDs []int64
		if err := tx.Model(&WgPeerRow{}).Where("interface_id = ?", id).Pluck("id", &peerIDs).Error; err != nil {
			return err
		}
		if len(peerIDs) > 0 {
			if err := tx.Where("peer_id IN ?", peerIDs).Delete(&StatsSnapshotRow{}).Error; err != nil {
				return err
			}
		}
		if err := tx.Where("interface_id = ?", id).Delete(&StatsSnapshotRow{}).Error; err != nil {
			return err
		}
		if err := tx.Where("interface_id = ?", id).Delete(&WgPeerRow{}).Error; err != nil {
			return err
		}
		res := tx.Delete(&WgInterfaceRow{}, id)
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return domain.ErrNotFound
		}
		return nil
	})
}

func (r *TunnelRepo) UpdateStatus(ctx context.Context, id int64, status domain.WgStatus, errMsg string) error {
	return r.db.WithContext(ctx).Model(&WgInterfaceRow{}).Where("id = ?", id).Updates(map[string]any{
		"status":        string(status),
		"error_message": errMsg,
	}).Error
}

func (r *TunnelRepo) UpdateRuntimeFields(ctx context.Context, id int64, tunName string, xrayPort, fwmark int) error {
	return r.db.WithContext(ctx).Model(&WgInterfaceRow{}).Where("id = ?", id).Updates(map[string]any{
		"tun_name":  tunName,
		"xray_port": xrayPort,
		"fw_mark":   fwmark,
	}).Error
}

func (r *TunnelRepo) ListRunningIDs(ctx context.Context) ([]int64, error) {
	var ids []int64
	err := r.db.WithContext(ctx).Model(&WgInterfaceRow{}).Where("status = ?", string(domain.WgStatusRunning)).Pluck("id", &ids).Error
	return ids, err
}

func (r *TunnelRepo) CountPeers(ctx context.Context, interfaceID int64) (int64, error) {
	var n int64
	err := r.db.WithContext(ctx).Model(&WgPeerRow{}).Where("interface_id = ?", interfaceID).Count(&n).Error
	return n, err
}

func (r *TunnelRepo) SetNodes(ctx context.Context, tunnelID int64, nodeIDs []int64) error {
	return r.nodes.SetNodes(ctx, tunnelID, nodeIDs)
}

func (r *TunnelRepo) ListNodes(ctx context.Context, tunnelID int64) ([]*domain.VlessNode, error) {
	return r.nodes.ListNodes(ctx, tunnelID)
}

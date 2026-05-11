package sqldb

import (
	"context"
	"errors"
	"time"

	"xray2wg/backend/internal/domain"

	"gorm.io/gorm"
)

type PeerRepo struct {
	db *gorm.DB
}

func NewPeerRepo(db *gorm.DB) *PeerRepo { return &PeerRepo{db: db} }

var _ domain.PeerRepository = (*PeerRepo)(nil)

func peerRow(r *WgPeerRow) *domain.WgPeer {
	return &domain.WgPeer{
		ID:                  r.ID,
		InterfaceID:         r.InterfaceID,
		Name:                r.Name,
		PublicKey:           r.PublicKey,
		ClientAddress:       r.ClientAddress,
		AllowedIPs:          r.AllowedIPs,
		PersistentKeepalive: r.PersistentKeepalive,
		LastHandshake:       r.LastHandshake,
		RxBytes:             r.RxBytes,
		TxBytes:             r.TxBytes,
		CreatedAt:           r.CreatedAt,
		UpdatedAt:           r.UpdatedAt,
	}
}

func (r *PeerRepo) Create(ctx context.Context, ifaceID int64, p *domain.WgPeer, privKeyEnc, pskEnc *string) error {
	row := WgPeerRow{
		InterfaceID:         ifaceID,
		Name:                p.Name,
		PublicKey:           p.PublicKey,
		ClientAddress:       p.ClientAddress,
		AllowedIPs:          p.AllowedIPs,
		PersistentKeepalive: p.PersistentKeepalive,
	}
	if privKeyEnc != nil {
		row.PrivateKeyEnc = *privKeyEnc
	}
	if pskEnc != nil {
		row.PresharedKeyEnc = *pskEnc
	}
	if err := r.db.WithContext(ctx).Create(&row).Error; err != nil {
		return err
	}
	p.ID = row.ID
	p.InterfaceID = ifaceID
	p.CreatedAt = row.CreatedAt
	p.UpdatedAt = row.UpdatedAt
	return nil
}

func (r *PeerRepo) GetByID(ctx context.Context, ifaceID int64, peerID int64) (*domain.WgPeer, string, string, error) {
	var row WgPeerRow
	if err := r.db.WithContext(ctx).Where("interface_id = ? AND id = ?", ifaceID, peerID).First(&row).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, "", "", domain.ErrNotFound
		}
		return nil, "", "", err
	}
	return peerRow(&row), row.PrivateKeyEnc, row.PresharedKeyEnc, nil
}

func (r *PeerRepo) ListByInterface(ctx context.Context, ifaceID int64) ([]*domain.WgPeer, error) {
	var rows []WgPeerRow
	if err := r.db.WithContext(ctx).Where("interface_id = ?", ifaceID).Order("id asc").Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]*domain.WgPeer, len(rows))
	for i := range rows {
		out[i] = peerRow(&rows[i])
	}
	return out, nil
}

func (r *PeerRepo) ListAllWithTunnel(ctx context.Context) ([]*domain.PeerWithTunnel, error) {
	var rows []WgPeerRow
	if err := r.db.WithContext(ctx).Preload("Interface").Order("interface_id asc, id asc").Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]*domain.PeerWithTunnel, 0, len(rows))
	for i := range rows {
		p := peerRow(&rows[i])
		tn := ""
		if rows[i].Interface != nil {
			tn = rows[i].Interface.Name
		}
		out = append(out, &domain.PeerWithTunnel{WgPeer: *p, TunnelName: tn})
	}
	return out, nil
}

func (r *PeerRepo) Update(ctx context.Context, p *domain.WgPeer, privKeyEnc, pskEnc *string) error {
	m := map[string]any{
		"name":                  p.Name,
		"public_key":            p.PublicKey,
		"client_address":        p.ClientAddress,
		"allowed_ips":           p.AllowedIPs,
		"persistent_keepalive":  p.PersistentKeepalive,
		"last_handshake":        p.LastHandshake,
		"rx_bytes":              p.RxBytes,
		"tx_bytes":              p.TxBytes,
	}
	if privKeyEnc != nil {
		m["private_key_enc"] = *privKeyEnc
	}
	if pskEnc != nil {
		m["preshared_key_enc"] = *pskEnc
	}
	res := r.db.WithContext(ctx).Model(&WgPeerRow{}).Where("id = ? AND interface_id = ?", p.ID, p.InterfaceID).Updates(m)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *PeerRepo) Delete(ctx context.Context, ifaceID int64, peerID int64) error {
	res := r.db.WithContext(ctx).Where("interface_id = ? AND id = ?", ifaceID, peerID).Delete(&WgPeerRow{})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *PeerRepo) UpdateTraffic(ctx context.Context, peerID int64, lastHS *time.Time, rx, tx int64) error {
	return r.db.WithContext(ctx).Model(&WgPeerRow{}).Where("id = ?", peerID).Updates(map[string]any{
		"last_handshake": lastHS,
		"rx_bytes":       rx,
		"tx_bytes":       tx,
	}).Error
}

func (r *PeerRepo) GetByPubKey(ctx context.Context, ifaceID int64, pubkey string) (*domain.WgPeer, error) {
	var row WgPeerRow
	if err := r.db.WithContext(ctx).Where("interface_id = ? AND public_key = ?", ifaceID, pubkey).First(&row).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	p := peerRow(&row)
	return p, nil
}

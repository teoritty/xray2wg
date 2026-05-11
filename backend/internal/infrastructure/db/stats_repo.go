package sqldb

import (
	"context"
	"time"

	"xray2wg/backend/internal/domain"

	"gorm.io/gorm"
)

type StatsRepo struct {
	db *gorm.DB
}

func NewStatsRepo(db *gorm.DB) *StatsRepo { return &StatsRepo{db: db} }

var _ domain.StatsRepository = (*StatsRepo)(nil)

func (r *StatsRepo) Insert(ctx context.Context, s *domain.StatSnapshot) error {
	row := StatsSnapshotRow{
		InterfaceID: s.InterfaceID,
		PeerID:      s.PeerID,
		RxBytes:     s.RxBytes,
		TxBytes:     s.TxBytes,
		RxRate:      s.RxRate,
		TxRate:      s.TxRate,
		ActivePeers: s.ActivePeers,
		SampledAt:   s.SampledAt,
	}
	return r.db.WithContext(ctx).Create(&row).Error
}

func (r *StatsRepo) DeleteOlderThan(ctx context.Context, cutoff time.Time) error {
	return r.db.WithContext(ctx).Where("sampled_at < ?", cutoff).Delete(&StatsSnapshotRow{}).Error
}

func (r *StatsRepo) QueryInterfaceWindow(ctx context.Context, ifaceID int64, from, to time.Time) ([]domain.StatSnapshot, error) {
	var rows []StatsSnapshotRow
	if err := r.db.WithContext(ctx).Where(
		"interface_id = ? AND peer_id IS NULL AND sampled_at BETWEEN ? AND ?",
		ifaceID, from, to,
	).Order("sampled_at asc").Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]domain.StatSnapshot, len(rows))
	for i := range rows {
		row := rows[i]
		ap := row.ActivePeers
		out[i] = domain.StatSnapshot{
			ID:           row.ID,
			InterfaceID:  row.InterfaceID,
			RxBytes:      row.RxBytes,
			TxBytes:      row.TxBytes,
			RxRate:       row.RxRate,
			TxRate:       row.TxRate,
			ActivePeers:  ap,
			SampledAt:    row.SampledAt,
		}
	}
	return out, nil
}

func (r *StatsRepo) QueryPeerWindow(ctx context.Context, peerID int64, from, to time.Time) ([]domain.StatSnapshot, error) {
	var rows []StatsSnapshotRow
	if err := r.db.WithContext(ctx).Where(
		"peer_id = ? AND sampled_at BETWEEN ? AND ?",
		peerID, from, to,
	).Order("sampled_at asc").Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]domain.StatSnapshot, len(rows))
	for i := range rows {
		row := rows[i]
		pid := row.PeerID
		out[i] = domain.StatSnapshot{
			ID:          row.ID,
			InterfaceID: row.InterfaceID,
			PeerID:      pid,
			RxBytes:     row.RxBytes,
			TxBytes:     row.TxBytes,
			RxRate:      row.RxRate,
			TxRate:      row.TxRate,
			SampledAt:   row.SampledAt,
		}
	}
	return out, nil
}

func (r *StatsRepo) QueryLatestInterfaceRates(ctx context.Context) (map[int64][2]int64, error) {
	sub := r.db.Table("stats_snapshots s1").
		Select("s1.interface_id, MAX(s1.sampled_at) AS mt").
		Where("s1.interface_id IS NOT NULL AND s1.peer_id IS NULL").
		Group("s1.interface_id")
	var joins []struct {
		InterfaceID int64 `gorm:"column:interface_id"`
		RxRate      int64 `gorm:"column:rx_rate"`
		TxRate      int64 `gorm:"column:tx_rate"`
	}
	err := r.db.WithContext(ctx).Table("stats_snapshots s").
		Select("s.interface_id, s.rx_rate, s.tx_rate").
		Joins("INNER JOIN (?) AS mx ON mx.interface_id = s.interface_id AND mx.mt = s.sampled_at", sub).
		Find(&joins).Error
	if err != nil {
		return nil, err
	}
	m := make(map[int64][2]int64, len(joins))
	for _, row := range joins {
		m[row.InterfaceID] = [2]int64{row.RxRate, row.TxRate}
	}
	return m, nil
}

func (r *StatsRepo) DBCounts(ctx context.Context) (tunnelsRunning int64, totalPeers int64, totalRX int64, totalTX int64, err error) {
	err = r.db.WithContext(ctx).Model(&WgInterfaceRow{}).Where("status = ?", string(domain.WgStatusRunning)).Count(&tunnelsRunning).Error
	if err != nil {
		return
	}
	err = r.db.WithContext(ctx).Model(&WgPeerRow{}).Count(&totalPeers).Error
	if err != nil {
		return
	}

	var wgSum struct {
		RxSum int64
		TxSum int64
	}
	err = r.db.WithContext(ctx).Model(&WgPeerRow{}).Select(
		"COALESCE(SUM(rx_bytes),0) as rx_sum, COALESCE(SUM(tx_bytes),0) as tx_sum",
	).Scan(&wgSum).Error
	if err != nil {
		return
	}
	totalRX = wgSum.RxSum
	totalTX = wgSum.TxSum
	return
}

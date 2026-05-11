package sqldb

import (
	"context"
	"errors"

	"xray2wg/backend/internal/domain"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/logger"
)

type SettingRepo struct {
	db *gorm.DB
}

func NewSettingRepo(db *gorm.DB) *SettingRepo { return &SettingRepo{db: db} }

var _ domain.SettingRepository = (*SettingRepo)(nil)

func (r *SettingRepo) Get(ctx context.Context, key string) (string, error) {
	var row Setting
	tx := r.db.WithContext(ctx).Session(&gorm.Session{
		Logger: logger.Default.LogMode(logger.Silent),
	}).Where("key = ?", key).Take(&row)
	if errors.Is(tx.Error, gorm.ErrRecordNotFound) {
		return "", nil
	}
	if tx.Error != nil {
		return "", tx.Error
	}
	return row.Value, nil
}

func (r *SettingRepo) Set(ctx context.Context, key, value string) error {
	row := Setting{Key: key, Value: value}
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "key"}},
		DoUpdates: clause.AssignmentColumns([]string{"value", "updated_at"}),
	}).Create(&row).Error
}

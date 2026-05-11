package sqldb

import (
	"time"

	"gorm.io/gorm"
)

type AuditLogRepo struct {
	db *gorm.DB
}

func NewAuditLogRepo(db *gorm.DB) *AuditLogRepo {
	return &AuditLogRepo{db: db}
}

func (r *AuditLogRepo) Save(level, source, msg string) error {
	row := AuditLogRow{
		Level:     level,
		Source:    source,
		Message:   msg,
		CreatedAt: time.Now().UTC(),
	}
	return r.db.Create(&row).Error
}

type AuditLogFilter struct {
	Level  string // "", "warn", "error"
	Search string
	Limit  int
	Offset int
}

type AuditLogPage struct {
	Items []AuditLogRow
	Total int64
}

func (r *AuditLogRepo) List(f AuditLogFilter) (AuditLogPage, error) {
	q := r.db.Model(&AuditLogRow{})

	switch f.Level {
	case "error":
		q = q.Where("level = ?", "error")
	case "warn":
		q = q.Where("level IN ?", []string{"warn", "error"})
	}

	if f.Search != "" {
		q = q.Where("message LIKE ?", "%"+f.Search+"%")
	}

	var total int64
	if err := q.Count(&total).Error; err != nil {
		return AuditLogPage{}, err
	}

	limit := f.Limit
	if limit <= 0 || limit > 200 {
		limit = 50
	}

	var rows []AuditLogRow
	if err := q.Order("created_at DESC").Limit(limit).Offset(f.Offset).Find(&rows).Error; err != nil {
		return AuditLogPage{}, err
	}

	return AuditLogPage{Items: rows, Total: total}, nil
}

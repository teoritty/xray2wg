package sqldb

import (
	"context"
	"os"
	"path/filepath"

	gormsqlite "github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func InitDB(ctx context.Context, dataDir string, logLevel logger.LogLevel) (*gorm.DB, error) {
	_ = ctx
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		return nil, err
	}
	dbPath := filepath.Join(dataDir, "xray2wg.db")
	gormDB, err := gorm.Open(gormsqlite.Open(dbPath+"?_pragma=foreign_keys(1)"), &gorm.Config{
		Logger: logger.Default.LogMode(logLevel),
	})
	if err != nil {
		return nil, err
	}
	if err := AutoMigrate(gormDB); err != nil {
		return nil, err
	}
	return gormDB, nil
}

package database

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"nginx-certs-manager/internal/model"
)

var DB *gorm.DB

func Init(dbPath string) error {
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("create db directory: %w", err)
	}

	var err error
	DB, err = gorm.Open(sqlite.Open(dbPath), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Error),
	})
	if err != nil {
		return fmt.Errorf("open database: %w", err)
	}

	if err := DB.AutoMigrate(
		&model.User{},
		&model.Certificate{},
		&model.Site{},
		&model.OperationLog{},
		&model.Setting{},
	); err != nil {
		return fmt.Errorf("migrate: %w", err)
	}

	return nil
}

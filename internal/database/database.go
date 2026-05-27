package database

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/glebarez/sqlite"
	"golang.org/x/crypto/bcrypt"
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

	seedAdmin()
	return nil
}

func seedAdmin() {
	var count int64
	DB.Model(&model.User{}).Where("username = ?", "admin").Count(&count)
	if count > 0 {
		return
	}

	password := os.Getenv("ADMIN_PASSWORD")
	if password == "" {
		password = "admin"
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		log.Printf("failed to hash admin password: %v", err)
		return
	}

	admin := model.User{
		Username: "admin",
		Password: string(hash),
		Role:     "admin",
	}
	DB.Create(&admin)
	log.Printf("Default admin created (username: admin). Change password after first login!")
}

// Package database 提供数据库初始化和管理功能。
// 使用 GORM 作为 ORM 框架，底层数据库为 SQLite（纯 Go 实现，无需 CGO）。
// 包含数据库连接初始化、自动迁移和默认管理员账户创建等功能。
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

// DB 是全局数据库连接实例，供整个应用程序使用。
// 在 Init 函数中初始化后，其他包可以通过 database.DB 访问数据库。
var DB *gorm.DB

// Init 初始化数据库连接，执行以下操作：
// 1. 创建数据库文件所在目录（如果不存在）
// 2. 打开 SQLite 数据库连接
// 3. 执行自动迁移（创建/更新所有数据表结构）
// 4. 创建默认管理员账户（如果不存在）
// 参数:
//   - dbPath: SQLite 数据库文件的路径
//
// 返回值:
//   - error: 初始化过程中的错误，成功则返回 nil
func Init(dbPath string) error {
	// 获取数据库文件所在目录，如果不存在则创建
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("create db directory: %w", err)
	}

	// 使用纯 Go 实现的 SQLite 驱动打开数据库连接
	// 日志级别设置为 Error，只记录错误信息，减少日志输出
	var err error
	DB, err = gorm.Open(sqlite.Open(dbPath), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Error),
	})
	if err != nil {
		return fmt.Errorf("open database: %w", err)
	}

	// 自动迁移：根据模型定义创建或更新数据表结构
	// 包括 User（用户）、Certificate（证书）、Site（站点）、
	// OperationLog（操作日志）、Setting（系统设置）五张表
	if err := DB.AutoMigrate(
		&model.User{},
		&model.Certificate{},
		&model.Site{},
		&model.OperationLog{},
		&model.Setting{},
	); err != nil {
		return fmt.Errorf("migrate: %w", err)
	}

	// 创建默认管理员账户
	seedAdmin()
	return nil
}

// seedAdmin 创建默认管理员账户（用户名: admin）。
// 如果管理员账户已存在则跳过创建。
// 密码优先从环境变量 ADMIN_PASSWORD 读取，未设置则默认为 "admin"。
// 密码使用 bcrypt 算法进行哈希存储，确保安全性。
func seedAdmin() {
	// 检查是否已存在 admin 用户
	var count int64
	DB.Model(&model.User{}).Where("username = ?", "admin").Count(&count)
	if count > 0 {
		return // 管理员已存在，跳过创建
	}

	// 从环境变量读取管理员密码，未设置则使用默认值
	password := os.Getenv("ADMIN_PASSWORD")
	if password == "" {
		password = "admin"
	}

	// 使用 bcrypt 对密码进行哈希处理（默认 cost=10，安全性较高）
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		log.Printf("failed to hash admin password: %v", err)
		return
	}

	// 创建管理员用户记录
	admin := model.User{
		Username: "admin",
		Password: string(hash),
		Role:     "admin",
	}
	DB.Create(&admin)
	log.Printf("Default admin created (username: admin). Change password after first login!")
}

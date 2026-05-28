// Package config 提供应用程序的配置管理功能。
// 配置通过环境变量加载，每个配置项都有对应的默认值。
package config

import "os"

// Config 结构体定义了应用程序的所有配置项。
// 所有配置都可以通过环境变量进行自定义。
type Config struct {
	Port       string // HTTP 服务器监听端口，环境变量 PORT，默认 "8080"
	DBPath     string // SQLite 数据库文件路径，环境变量 DB_PATH，默认 "./data/certs.db"
	WebDir     string // 前端构建产物目录路径，环境变量 WEB_DIR，默认 "./web/dist"
	NginxDir   string // Nginx 配置文件目录路径，环境变量 NGINX_DIR，默认 "/etc/nginx"
	CertbotDir string // Certbot/Let's Encrypt 证书目录路径，环境变量 CERTBOT_DIR，默认 "/etc/letsencrypt"
}

// Load 从环境变量加载配置，如果环境变量未设置则使用默认值。
// 返回值:
//   - *Config: 包含所有配置项的配置结构体指针
func Load() *Config {
	return &Config{
		Port:       getEnv("PORT", "8080"),
		DBPath:     getEnv("DB_PATH", "./data/certs.db"),
		WebDir:     getEnv("WEB_DIR", "./web/dist"),
		NginxDir:   getEnv("NGINX_DIR", "/etc/nginx"),
		CertbotDir: getEnv("CERTBOT_DIR", "/etc/letsencrypt"),
	}
}

// getEnv 辅助函数：获取环境变量的值，如果环境变量不存在或为空则返回 fallback 默认值。
// 参数:
//   - key: 环境变量名称
//   - fallback: 默认值，当环境变量未设置时使用
//
// 返回值:
//   - string: 环境变量的值或默认值
func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

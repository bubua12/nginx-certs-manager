package config

import "os"

type Config struct {
	Port       string
	DBPath     string
	WebDir     string
	NginxDir   string
	CertbotDir string
}

func Load() *Config {
	return &Config{
		Port:       getEnv("PORT", "8080"),
		DBPath:     getEnv("DB_PATH", "./data/certs.db"),
		WebDir:     getEnv("WEB_DIR", "./web/dist"),
		NginxDir:   getEnv("NGINX_DIR", "/etc/nginx"),
		CertbotDir: getEnv("CERTBOT_DIR", "/etc/letsencrypt"),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

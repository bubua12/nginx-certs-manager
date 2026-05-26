package model

import "time"

type Site struct {
	ID            uint   `json:"id" gorm:"primaryKey"`
	Domain        string `json:"domain" gorm:"uniqueIndex;not null"`
	ConfigPath    string `json:"config_path"`
	Port          int    `json:"port" gorm:"default:443"`
	Upstream      string `json:"upstream"`
	SSLEnabled    bool   `json:"ssl_enabled" gorm:"default:1"`
	CertificateID *uint  `json:"certificate_id"`
	Certificate   *Certificate `json:"certificate,omitempty" gorm:"foreignKey:CertificateID"`
	Enabled       bool   `json:"enabled" gorm:"default:1"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

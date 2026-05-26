package model

import "time"

type Certificate struct {
	ID           uint      `json:"id" gorm:"primaryKey"`
	Domain       string    `json:"domain" gorm:"uniqueIndex;not null"`
	SANs         string    `json:"sans"`
	CertPath     string    `json:"cert_path"`
	KeyPath      string    `json:"key_path"`
	Issuer       string    `json:"issuer"`
	NotBefore    time.Time `json:"not_before"`
	NotAfter     time.Time `json:"not_after"`
	AutoRenew    bool      `json:"auto_renew" gorm:"default:1"`
	Status       string    `json:"status" gorm:"default:'active'"`
	LastRenewed  *time.Time `json:"last_renewed"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

func (c *Certificate) DaysUntilExpiry() int {
	return int(time.Until(c.NotAfter).Hours() / 24)
}

func (c *Certificate) UpdateStatus() {
	days := c.DaysUntilExpiry()
	switch {
	case days < 0:
		c.Status = "expired"
	case days <= 30:
		c.Status = "expiring"
	default:
		c.Status = "active"
	}
}

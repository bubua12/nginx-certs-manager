// Package model 定义了应用程序的数据模型结构体。
// 所有模型都使用 GORM 标签来定义数据库列属性，使用 JSON 标签来控制 API 序列化格式。
package model

import "time"

// Certificate 结构体表示一个 SSL/TLS 证书的完整信息。
// 对应数据库中的 certificates 表，存储从 Let's Encrypt 获取的证书元数据。
type Certificate struct {
	ID          uint       `json:"id" gorm:"primaryKey"`                    // 证书记录的唯一标识符（主键）
	Domain      string     `json:"domain" gorm:"uniqueIndex;not null"`      // 证书绑定的主域名，建立唯一索引
	SANs        string     `json:"sans"`                                    // 主题备用名称（Subject Alternative Names），JSON 数组格式存储
	CertPath    string     `json:"cert_path"`                               // 证书文件（fullchain.pem）的完整路径
	KeyPath     string     `json:"key_path"`                                // 私钥文件（privkey.pem）的完整路径
	Issuer      string     `json:"issuer"`                                  // 证书颁发机构（Issuer）名称
	NotBefore   time.Time  `json:"not_before"`                              // 证书生效时间
	NotAfter    time.Time  `json:"not_after"`                               // 证书过期时间
	AutoRenew   bool       `json:"auto_renew" gorm:"default:1"`             // 是否启用自动续期，默认为 true
	Status      string     `json:"status" gorm:"default:'active'"`          // 证书状态：active（正常）、expiring（即将过期）、expired（已过期）
	LastRenewed *time.Time `json:"last_renewed"`                            // 最后一次续期时间，指针类型可为 nil
	CreatedAt   time.Time  `json:"created_at"`                              // 记录创建时间（GORM 自动管理）
	UpdatedAt   time.Time  `json:"updated_at"`                              // 记录最后更新时间（GORM 自动管理）
}

// DaysUntilExpiry 计算距离证书过期还有多少天。
// 返回值为负数表示证书已过期。
// 返回值:
//   - int: 距离过期的天数（负数表示已过期）
func (c *Certificate) DaysUntilExpiry() int {
	return int(time.Until(c.NotAfter).Hours() / 24)
}

// UpdateStatus 根据距离过期的天数自动更新证书状态：
// - 已过期（天数 < 0）：状态设为 "expired"
// - 即将过期（天数 <= 30）：状态设为 "expiring"
// - 正常（天数 > 30）：状态设为 "active"
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

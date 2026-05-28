package model

import "time"

// Site 结构体表示一个 Nginx 站点（虚拟主机）的配置信息。
// 对应数据库中的 sites 表，记录从 Nginx 配置文件中解析出的站点数据。
type Site struct {
	ID            uint         `json:"id" gorm:"primaryKey"`               // 站点记录的唯一标识符（主键）
	Domain        string       `json:"domain" gorm:"uniqueIndex;not null"` // 站点绑定的域名，建立唯一索引
	ConfigPath    string       `json:"config_path"`                        // Nginx 配置文件的完整路径
	Port          int          `json:"port" gorm:"default:443"`            // 站点监听端口，默认 443（HTTPS）
	Upstream      string       `json:"upstream"`                           // 反向代理上游地址（如 http://127.0.0.1:3000）
	SSLEnabled    bool         `json:"ssl_enabled" gorm:"default:1"`       // 是否启用 SSL，默认为 true
	CertificateID *uint        `json:"certificate_id"`                     // 关联的证书 ID，外键指向 certificates 表
	Certificate   *Certificate `json:"certificate,omitempty" gorm:"foreignKey:CertificateID"` // 关联的证书对象（GORM 关联查询）
	Enabled       bool         `json:"enabled" gorm:"default:1"`           // 站点是否启用，默认为 true
	CreatedAt     time.Time    `json:"created_at"`                         // 记录创建时间（GORM 自动管理）
	UpdatedAt     time.Time    `json:"updated_at"`                         // 记录最后更新时间（GORM 自动管理）
}

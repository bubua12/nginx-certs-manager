package model

import "time"

// OperationLog 结构体表示一条操作日志记录。
// 对应数据库中的 operation_logs 表，记录系统中所有重要操作（如证书续期、站点启用等）。
type OperationLog struct {
	ID        uint      `json:"id" gorm:"primaryKey"`             // 日志记录的唯一标识符（主键）
	Type      string    `json:"type" gorm:"not null"`             // 操作类型，如 "cert_renew"、"site_enable"、"user_login" 等
	Target    string    `json:"target"`                           // 操作目标，通常是域名或用户名
	Status    string    `json:"status"`                           // 操作结果状态："success" 或 "failed"
	Message   string    `json:"message"`                          // 操作详情或错误信息
	Operator  string    `json:"operator" gorm:"default:'system'"` // 操作者，默认为 "system"（系统自动操作）
	CreatedAt time.Time `json:"created_at"`                       // 操作发生的时间（GORM 自动管理）
}

// Setting 结构体表示一条系统设置项。
// 对应数据库中的 settings 表，采用键值对方式存储配置。
type Setting struct {
	Key       string    `json:"key" gorm:"primaryKey"` // 设置项的键名（主键）
	Value     string    `json:"value"`                 // 设置项的值
	UpdatedAt time.Time `json:"updated_at"`            // 最后更新时间（GORM 自动管理）
}

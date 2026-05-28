package model

import "time"

// User 结构体表示一个系统用户。
// 对应数据库中的 users 表，存储用户认证和授权信息。
// 密码字段使用 bcrypt 哈希存储，JSON 序列化时排除密码字段。
type User struct {
	ID        uint      `json:"id" gorm:"primaryKey"`           // 用户的唯一标识符（主键）
	Username  string    `json:"username" gorm:"uniqueIndex;not null"` // 用户名，建立唯一索引，不允许为空
	Password  string    `json:"-" gorm:"not null"`              // 密码（bcrypt 哈希值），json:"-" 表示 JSON 序列化时排除该字段
	Role      string    `json:"role" gorm:"default:'user'"`     // 用户角色："admin"（管理员）或 "user"（普通用户），默认为 "user"
	CreatedAt time.Time `json:"created_at"`                     // 账户创建时间（GORM 自动管理）
	UpdatedAt time.Time `json:"updated_at"`                     // 账户最后更新时间（GORM 自动管理）
}

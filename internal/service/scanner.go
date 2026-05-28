// Package service 提供证书管理、Nginx 管理、自动扫描等核心业务逻辑
package service

import (
	"log"
	"time"

	"nginx-certs-manager/internal/database"
	"nginx-certs-manager/internal/model"
)

// Scanner 证书和站点自动扫描器
// 定期检测 Let's Encrypt 证书状态和 Nginx 站点配置，自动同步到数据库
type Scanner struct {
	certbot *CertbotService // Certbot 服务实例，用于获取证书信息
	nginx   *NginxService   // Nginx 服务实例，用于获取站点配置
}

// NewScanner 创建扫描器实例
// 参数 certbot：Certbot 服务，nginx：Nginx 服务
func NewScanner(certbot *CertbotService, nginx *NginxService) *Scanner {
	return &Scanner{certbot: certbot, nginx: nginx}
}

// ScanCertificates 扫描 Let's Encrypt 证书目录，将证书信息同步到数据库
// 遍历所有证书，计算有效期状态（active/expiring/expired），新建或更新数据库记录
func (s *Scanner) ScanCertificates() {
	if !s.certbot.IsAvailable() { // certbot 不可用时跳过扫描
		log.Println("certbot not available, skipping certificate scan")
		return
	}

	certs, err := s.certbot.ListCertificates() // 获取所有证书信息
	if err != nil {
		log.Printf("scan certificates error: %v", err)
		return
	}

	for _, cert := range certs {
		// 跳过过期日期无效的证书（未正确解析或数据损坏）
		if cert.NotAfter.IsZero() || cert.NotAfter.Year() < 2000 {
			log.Printf("skipping %s: invalid expiry date", cert.Domain)
			continue
		}

		var existing model.Certificate
		result := database.DB.Where("domain = ?", cert.Domain).First(&existing) // 查询数据库中是否已有记录

		// 根据剩余天数判断证书状态
		status := "active"
		days := int(time.Until(cert.NotAfter).Hours() / 24) // 计算距过期的剩余天数
		if days < 0 {
			status = "expired"  // 已过期
		} else if days <= 30 {
			status = "expiring" // 即将过期（30 天内）
		}

		sansJSON := "" // 将 SAN 列表序列化为 JSON 字符串存储
		if len(cert.SANs) > 0 {
			sansJSON = joinJSON(cert.SANs)
		}

		if result.Error != nil { // 数据库中无记录，创建新记录
			dbCert := model.Certificate{
				Domain:    cert.Domain,    // 主域名
				SANs:      sansJSON,       // 主题备用名称（JSON）
				CertPath:  cert.CertPath,  // 证书文件路径
				KeyPath:   cert.KeyPath,   // 私钥文件路径
				Issuer:    cert.Issuer,    // 颁发者
				NotBefore: cert.NotBefore, // 生效时间
				NotAfter:  cert.NotAfter,  // 过期时间
				AutoRenew: true,           // 默认开启自动续期
				Status:    status,         // 证书状态
			}
			database.DB.Create(&dbCert) // 插入新记录
		} else { // 数据库中已有记录，更新字段
			existing.CertPath = cert.CertPath
			existing.KeyPath = cert.KeyPath
			existing.Issuer = cert.Issuer
			existing.NotAfter = cert.NotAfter
			existing.NotBefore = cert.NotBefore
			existing.SANs = sansJSON
			existing.Status = status
			database.DB.Save(&existing) // 保存更新
		}
	}

	log.Printf("scanned %d certificates", len(certs)) // 记录扫描完成日志
}

// ScanSites 扫描 Nginx 配置目录，将站点信息同步到数据库
// 遍历所有 Nginx 站点配置，自动关联 SSL 证书，新建或更新数据库记录
func (s *Scanner) ScanSites() {
	if !s.nginx.IsAvailable() { // Nginx 不可用时跳过扫描
		log.Println("nginx not available, skipping site scan")
		return
	}

	sites, err := s.nginx.ListSites() // 从 Nginx 配置文件中获取所有站点
	if err != nil {
		log.Printf("scan sites error: %v", err)
		return
	}

	for _, site := range sites {
		var existing model.Site
		result := database.DB.Where("domain = ?", site.Domain).First(&existing) // 查询是否已有记录

		// 若站点启用了 SSL，尝试关联数据库中的证书记录
		var certID *uint
		if site.SSLEnabled {
			var cert model.Certificate
			if database.DB.Where("domain = ?", site.Domain).First(&cert).Error == nil {
				certID = &cert.ID // 找到匹配的证书，建立关联
			}
		}

		if result.Error != nil { // 数据库中无记录，创建新站点记录
			dbSite := model.Site{
				Domain:        site.Domain,        // 域名
				ConfigPath:    site.ConfigPath,     // Nginx 配置文件路径
				Port:          site.Port,           // 监听端口
				Upstream:      site.Upstream,       // 反向代理后端地址
				SSLEnabled:    site.SSLEnabled,     // 是否启用 SSL
				CertificateID: certID,              // 关联的证书 ID
				Enabled:       site.Enabled,        // 是否启用
			}
			database.DB.Create(&dbSite) // 插入新记录
		} else { // 数据库中已有记录，更新字段
			existing.ConfigPath = site.ConfigPath
			existing.Port = site.Port
			existing.Upstream = site.Upstream
			existing.SSLEnabled = site.SSLEnabled
			existing.CertificateID = certID
			existing.Enabled = site.Enabled
			database.DB.Save(&existing) // 保存更新
		}
	}

	log.Printf("scanned %d sites", len(sites)) // 记录扫描完成日志
}

// ScanAll 执行全量扫描：先扫描证书，再扫描站点
// 站点扫描依赖证书数据（用于关联 SSL 证书），因此必须先扫描证书
func (s *Scanner) ScanAll() {
	s.ScanCertificates() // 第一步：同步证书信息
	s.ScanSites()        // 第二步：同步站点信息（依赖证书数据）
}

// StartPeriodicScan 启动后台定时扫描协程
// 立即执行一次全量扫描，之后按指定间隔周期性重复执行
// 参数 interval：扫描间隔时间（如 30 分钟、1 小时）
func (s *Scanner) StartPeriodicScan(interval time.Duration) {
	go func() {
		s.ScanAll()                              // 启动时立即执行一次扫描
		ticker := time.NewTicker(interval)         // 创建定时器
		defer ticker.Stop()                        // 确保定时器在函数退出时被清理
		for range ticker.C {                       // 每隔 interval 触发一次扫描
			s.ScanAll()
		}
	}()
}

// joinJSON 将字符串切片转换为 JSON 数组格式字符串
// 手动拼接 JSON 以避免引入 encoding/json 包的额外开销
// 参数 items：字符串切片；返回值：如 ["a","b","c"] 格式的 JSON 字符串
func joinJSON(items []string) string {
	result := "["
	for i, item := range items {
		if i > 0 {
			result += "," // 非首个元素前加逗号分隔
		}
		result += `"` + item + `"` // 用双引号包裹每个元素
	}
	result += "]"
	return result
}

// Package service 提供证书管理、Nginx 管理、自动扫描等核心业务逻辑
package service

import (
	"bytes"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// CertbotService 封装 Certbot CLI 命令行工具，提供证书管理功能
type CertbotService struct {
	BinPath string // certbot 可执行文件路径
	CertDir string // Let's Encrypt 证书存储目录（默认 /etc/letsencrypt）
}

// NewCertbotService 创建 CertbotService 实例
// 自动探测 certbot 可执行文件路径，支持 /usr/bin、/usr/local/bin、/snap/bin 三个常见位置
// 参数 certDir：证书存储目录，为空时使用默认路径 /etc/letsencrypt
func NewCertbotService(certDir string) *CertbotService {
	binPath := "certbot" // 默认使用 PATH 环境变量中的 certbot
	// 依次检测常见安装路径，找到第一个存在的即采用
	for _, p := range []string{"/usr/bin/certbot", "/usr/local/bin/certbot", "/snap/bin/certbot"} {
		if _, err := os.Stat(p); err == nil {
			binPath = p
			break
		}
	}
	if certDir == "" {
		certDir = "/etc/letsencrypt"
	}
	return &CertbotService{BinPath: binPath, CertDir: certDir}
}

// CertInfo 证书信息结构体，存储从证书文件或 certbot 命令解析出的证书元数据
type CertInfo struct {
	Domain    string    // 主域名（证书 CN 通用名称）
	SANs      []string  // 主题备用名称列表（多域名证书支持）
	CertPath  string    // 证书文件完整路径（fullchain.pem）
	KeyPath   string    // 私钥文件完整路径（privkey.pem）
	Issuer    string    // 证书颁发者（CA 机构名称）
	NotBefore time.Time // 证书生效时间
	NotAfter  time.Time // 证书过期时间
}

// ListCertificates 获取所有已管理的证书列表
// 策略：优先直接读取证书文件（x509 解析），更可靠且不依赖 certbot 命令
// 若文件读取失败或无结果，则回退使用 certbot certificates 命令解析文本输出
// 返回值：证书信息列表，错误信息
func (s *CertbotService) ListCertificates() ([]CertInfo, error) {
	// 首选方案：直接读取证书文件并用 x509 解析，最为可靠
	certs, err := s.listViaFiles()
	if err == nil && len(certs) > 0 {
		return certs, nil
	}

	// 备选方案：通过 certbot certificates 命令获取证书列表
	return s.listViaCommand()
}

// listViaCommand 执行 certbot certificates 命令并解析文本输出获取证书列表
// 解析 certbot 的命令行输出格式，提取证书名称、域名、过期时间等信息
func (s *CertbotService) listViaCommand() ([]CertInfo, error) {
	cmd := exec.Command(s.BinPath, "certificates")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("certbot certificates: %s", stderr.String()) // 命令执行失败，返回标准错误输出
	}

	return s.parseCertificates(stdout.String()), nil // 解析命令输出文本
}

// listViaFiles 直接读取 Let's Encrypt 证书目录获取证书信息
// 遍历 /etc/letsencrypt/live/ 下的所有域名目录，读取 fullchain.pem 证书文件
// 使用标准库 crypto/x509 解析证书内容，提取域名、SAN、颁发者、有效期等
func (s *CertbotService) listViaFiles() ([]CertInfo, error) {
	liveDir := filepath.Join(s.CertDir, "live") // 构建证书 live 目录路径
	entries, err := os.ReadDir(liveDir)
	if err != nil {
		return nil, fmt.Errorf("read live dir: %w", err) // 目录不存在或无权限
	}

	var certs []CertInfo
	for _, entry := range entries {
		if !entry.IsDir() {
			continue // 跳过非目录项（如 README 文件）
		}

		domain := entry.Name() // 目录名即域名
		certPath := filepath.Join(liveDir, domain, "fullchain.pem") // 完整证书链路径
		keyPath := filepath.Join(liveDir, domain, "privkey.pem")    // 私钥路径

		if _, err := os.Stat(certPath); err != nil {
			continue // 证书文件不存在，跳过该域名
		}

		info, err := s.parseCertFile(certPath) // 解析证书文件
		if err != nil {
			continue // 解析失败（文件损坏或格式错误），跳过
		}

		info.Domain = domain
		info.CertPath = certPath
		info.KeyPath = keyPath
		certs = append(certs, *info)
	}

	return certs, nil
}

// parseCertFile 使用 Go 标准库解析单个 PEM 格式证书文件
// 读取证书文件内容，解码 PEM 块，再用 x509.ParseCertificate 解析 DER 编码数据
// 参数 certPath：PEM 证书文件的绝对路径
// 返回值：解析后的证书信息指针，错误信息
func (s *CertbotService) parseCertFile(certPath string) (*CertInfo, error) {
	data, err := os.ReadFile(certPath)
	if err != nil {
		return nil, err // 文件读取失败
	}

	block, _ := pem.Decode(data) // 从 PEM 数据中解码第一个证书块
	if block == nil {
		return nil, fmt.Errorf("no PEM block found") // 未找到有效的 PEM 编码块
	}

	cert, err := x509.ParseCertificate(block.Bytes) // 解析 DER 编码的 X.509 证书
	if err != nil {
		return nil, err // DER 数据解析失败
	}

	// 从解析后的证书对象中提取关键字段
	info := &CertInfo{
		Domain:    cert.Subject.CommonName,
		SANs:      cert.DNSNames,
		Issuer:    cert.Issuer.CommonName,
		NotBefore: cert.NotBefore,
		NotAfter:  cert.NotAfter,
	}

	return info, nil
}

// parseCertificates 解析 certbot certificates 命令的文本输出
// 逐行扫描输出文本，按 "Certificate Name:" 分隔不同证书块
// 提取域名、SAN 列表、过期日期、证书路径、私钥路径、颁发者等字段
// 返回值：解析出的证书信息列表
func (s *CertbotService) parseCertificates(output string) []CertInfo {
	var certs []CertInfo
	var current *CertInfo // 当前正在解析的证书信息

	for _, line := range strings.Split(output, "\n") { // 按行遍历输出
		line = strings.TrimSpace(line)

		if strings.HasPrefix(line, "Certificate Name:") { // 遇到新证书名称，开始新证书块
			if current != nil {
				certs = append(certs, *current) // 保存上一个证书信息
			}
			current = &CertInfo{
				Domain: strings.TrimSpace(strings.TrimPrefix(line, "Certificate Name:")), // 提取证书域名
			}
			continue
		}
		if current == nil {
			continue // 尚未遇到证书名称行，跳过
		}

		if strings.HasPrefix(line, "Domains:") { // 解析关联域名列表
			domains := strings.TrimSpace(strings.TrimPrefix(line, "Domains:"))
			current.SANs = strings.Fields(domains) // 按空格分割多个域名
			if len(current.SANs) > 0 && current.Domain == "" {
				current.Domain = current.SANs[0] // 若未设置主域名，使用第一个 SAN
			}
		}
		if strings.HasPrefix(line, "Expiry Date:") { // 解析证书过期日期
			dateStr := strings.TrimSpace(strings.TrimPrefix(line, "Expiry Date:"))
			dateStr = strings.Split(dateStr, "(")[0] // 去掉括号内的相对时间描述（如 "(VALID: 89 days)"）
			dateStr = strings.TrimSpace(dateStr)
			if t, err := time.Parse("2006-01-02 15:04:05", dateStr); err == nil { // Go 的参考时间格式
				current.NotAfter = t // 解析成功则设置过期时间
			}
		}
		if strings.HasPrefix(line, "Certificate Path:") { // 证书文件路径
			current.CertPath = strings.TrimSpace(strings.TrimPrefix(line, "Certificate Path:"))
		}
		if strings.HasPrefix(line, "Private Key Path:") { // 私钥文件路径
			current.KeyPath = strings.TrimSpace(strings.TrimPrefix(line, "Private Key Path:"))
		}
		if strings.HasPrefix(line, "Issuer:") { // 证书颁发者
			current.Issuer = strings.TrimSpace(strings.TrimPrefix(line, "Issuer:"))
		}
	}

	if current != nil {
		certs = append(certs, *current) // 追加最后一个证书信息（循环结束时不会被追加）
	}

	return certs
}

// Renew 续期指定域名的证书，或续期所有即将过期的证书
// 执行 certbot renew --non-interactive 命令
// 参数 domain：指定续期的证书域名，为空则续期所有证书
// 返回值：命令输出文本（含 stdout 和 stderr），错误信息
func (s *CertbotService) Renew(domain string) (string, error) {
	args := []string{"renew", "--non-interactive"} // 非交互模式，适合自动化调用
	if domain != "" {
		args = append(args, "--cert-name", domain) // 指定域名则只续期该证书
	}

	cmd := exec.Command(s.BinPath, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	output := stdout.String() + stderr.String() // 合并标准输出和错误输出
	if err != nil {
		return output, fmt.Errorf("certbot renew failed: %w", err) // 续期失败
	}
	return output, nil // 续期成功
}

// RequestCert 申请新域名的 SSL 证书
// 支持两种验证模式：
//   - webroot 模式：使用现有 Web 服务器的根目录进行 HTTP 验证（推荐）
//   - standalone 模式：启动临时 HTTP 服务器进行验证（需 80 端口空闲）
//
// 自动同意服务条款并使用不安全模式注册（无需邮箱）
// 参数 domain：要申请证书的域名；webroot：Web 服务器根目录路径，为空则使用 standalone 模式
// 返回值：命令输出文本，错误信息
func (s *CertbotService) RequestCert(domain string, webroot string) (string, error) {
	args := []string{"certonly", "--non-interactive", "--agree-tos", "--register-unsafely-without-email"}
	if webroot != "" {
		args = append(args, "--webroot", "-w", webroot, "-d", domain) // webroot 验证模式
	} else {
		args = append(args, "--standalone", "-d", domain) // standalone 验证模式
	}

	cmd := exec.Command(s.BinPath, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	output := stdout.String() + stderr.String()
	if err != nil {
		return output, fmt.Errorf("certbot certonly failed: %w", err)
	}
	return output, nil
}

// Revoke 吊销指定证书并删除本地文件
// 执行 certbot revoke --delete-after-revoke 命令
// 证书吊销后将从 CA 的证书透明度日志中标记为无效
// 参数 certPath：要吊销的证书文件路径
// 返回值：命令输出文本，错误信息
func (s *CertbotService) Revoke(certPath string) (string, error) {
	cmd := exec.Command(s.BinPath, "revoke", "--non-interactive", "--cert-path", certPath, "--delete-after-revoke")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	output := stdout.String() + stderr.String()
	if err != nil {
		return output, fmt.Errorf("certbot revoke failed: %w", err)
	}
	return output, nil
}

// IsAvailable 检测 certbot 工具是否可用
// 首先检查 certbot 可执行文件是否存在于 PATH 中
// 若不存在，再检查证书目录是否存在（作为降级判断）
// 返回值：true 表示 certbot 可用或证书目录存在
func (s *CertbotService) IsAvailable() bool {
	_, err := exec.LookPath(s.BinPath) // 在 PATH 中查找 certbot
	if err == nil {
		return true // certbot 可执行文件存在
	}
	// certbot 命令不存在时，检查证书目录是否存在作为降级判断
	liveDir := filepath.Join(s.CertDir, "live")
	_, err = os.Stat(liveDir)
	return err == nil
}

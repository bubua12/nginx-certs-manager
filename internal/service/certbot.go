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

type CertbotService struct {
	BinPath   string
	CertDir   string
}

func NewCertbotService(certDir string) *CertbotService {
	binPath := "certbot"
	// Try common certbot locations
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

type CertInfo struct {
	Domain    string
	SANs      []string
	CertPath  string
	KeyPath   string
	Issuer    string
	NotBefore time.Time
	NotAfter  time.Time
}

func (s *CertbotService) ListCertificates() ([]CertInfo, error) {
	// Primary: read certificate files directly (most reliable)
	certs, err := s.listViaFiles()
	if err == nil && len(certs) > 0 {
		return certs, nil
	}

	// Fallback: certbot certificates command
	return s.listViaCommand()
}

func (s *CertbotService) listViaCommand() ([]CertInfo, error) {
	cmd := exec.Command(s.BinPath, "certificates")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("certbot certificates: %s", stderr.String())
	}

	return s.parseCertificates(stdout.String()), nil
}

func (s *CertbotService) listViaFiles() ([]CertInfo, error) {
	liveDir := filepath.Join(s.CertDir, "live")
	entries, err := os.ReadDir(liveDir)
	if err != nil {
		return nil, fmt.Errorf("read live dir: %w", err)
	}

	var certs []CertInfo
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		domain := entry.Name()
		certPath := filepath.Join(liveDir, domain, "fullchain.pem")
		keyPath := filepath.Join(liveDir, domain, "privkey.pem")

		if _, err := os.Stat(certPath); err != nil {
			continue
		}

		info, err := s.parseCertFile(certPath)
		if err != nil {
			continue
		}

		info.Domain = domain
		info.CertPath = certPath
		info.KeyPath = keyPath
		certs = append(certs, *info)
	}

	return certs, nil
}

func (s *CertbotService) parseCertFile(certPath string) (*CertInfo, error) {
	data, err := os.ReadFile(certPath)
	if err != nil {
		return nil, err
	}

	block, _ := pem.Decode(data)
	if block == nil {
		return nil, fmt.Errorf("no PEM block found")
	}

	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil, err
	}

	info := &CertInfo{
		Domain:    cert.Subject.CommonName,
		SANs:      cert.DNSNames,
		Issuer:    cert.Issuer.CommonName,
		NotBefore: cert.NotBefore,
		NotAfter:  cert.NotAfter,
	}

	return info, nil
}

func (s *CertbotService) parseCertificates(output string) []CertInfo {
	var certs []CertInfo
	var current *CertInfo

	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)

		if strings.HasPrefix(line, "Certificate Name:") {
			if current != nil {
				certs = append(certs, *current)
			}
			current = &CertInfo{
				Domain: strings.TrimSpace(strings.TrimPrefix(line, "Certificate Name:")),
			}
			continue
		}
		if current == nil {
			continue
		}

		if strings.HasPrefix(line, "Domains:") {
			domains := strings.TrimSpace(strings.TrimPrefix(line, "Domains:"))
			current.SANs = strings.Fields(domains)
			if len(current.SANs) > 0 && current.Domain == "" {
				current.Domain = current.SANs[0]
			}
		}
		if strings.HasPrefix(line, "Expiry Date:") {
			dateStr := strings.TrimSpace(strings.TrimPrefix(line, "Expiry Date:"))
			dateStr = strings.Split(dateStr, "(")[0]
			dateStr = strings.TrimSpace(dateStr)
			if t, err := time.Parse("2006-01-02 15:04:05", dateStr); err == nil {
				current.NotAfter = t
			}
		}
		if strings.HasPrefix(line, "Certificate Path:") {
			current.CertPath = strings.TrimSpace(strings.TrimPrefix(line, "Certificate Path:"))
		}
		if strings.HasPrefix(line, "Private Key Path:") {
			current.KeyPath = strings.TrimSpace(strings.TrimPrefix(line, "Private Key Path:"))
		}
		if strings.HasPrefix(line, "Issuer:") {
			current.Issuer = strings.TrimSpace(strings.TrimPrefix(line, "Issuer:"))
		}
	}

	if current != nil {
		certs = append(certs, *current)
	}

	return certs
}

func (s *CertbotService) Renew(domain string) (string, error) {
	args := []string{"renew", "--non-interactive"}
	if domain != "" {
		args = append(args, "--cert-name", domain)
	}

	cmd := exec.Command(s.BinPath, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	output := stdout.String() + stderr.String()
	if err != nil {
		return output, fmt.Errorf("certbot renew failed: %w", err)
	}
	return output, nil
}

func (s *CertbotService) RequestCert(domain string, webroot string) (string, error) {
	args := []string{"certonly", "--non-interactive", "--agree-tos", "--register-unsafely-without-email"}
	if webroot != "" {
		args = append(args, "--webroot", "-w", webroot, "-d", domain)
	} else {
		args = append(args, "--standalone", "-d", domain)
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

func (s *CertbotService) IsAvailable() bool {
	_, err := exec.LookPath(s.BinPath)
	if err == nil {
		return true
	}
	// Also check if cert files exist
	liveDir := filepath.Join(s.CertDir, "live")
	_, err = os.Stat(liveDir)
	return err == nil
}

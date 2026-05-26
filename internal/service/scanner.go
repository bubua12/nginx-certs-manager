package service

import (
	"log"
	"time"

	"nginx-certs-manager/internal/database"
	"nginx-certs-manager/internal/model"
)

type Scanner struct {
	certbot *CertbotService
	nginx   *NginxService
}

func NewScanner(certbot *CertbotService, nginx *NginxService) *Scanner {
	return &Scanner{certbot: certbot, nginx: nginx}
}

func (s *Scanner) ScanCertificates() {
	if !s.certbot.IsAvailable() {
		log.Println("certbot not available, skipping certificate scan")
		return
	}

	certs, err := s.certbot.ListCertificates()
	if err != nil {
		log.Printf("scan certificates error: %v", err)
		return
	}

	for _, cert := range certs {
		var existing model.Certificate
		result := database.DB.Where("domain = ?", cert.Domain).First(&existing)

		status := "active"
		days := int(time.Until(cert.NotAfter).Hours() / 24)
		if days < 0 {
			status = "expired"
		} else if days <= 30 {
			status = "expiring"
		}

		sansJSON := ""
		if len(cert.SANs) > 0 {
			sansJSON = joinJSON(cert.SANs)
		}

		if result.Error != nil {
			dbCert := model.Certificate{
				Domain:    cert.Domain,
				SANs:      sansJSON,
				CertPath:  cert.CertPath,
				KeyPath:   cert.KeyPath,
				Issuer:    cert.Issuer,
				NotBefore: cert.NotBefore,
				NotAfter:  cert.NotAfter,
				AutoRenew: true,
				Status:    status,
			}
			database.DB.Create(&dbCert)
		} else {
			existing.CertPath = cert.CertPath
			existing.KeyPath = cert.KeyPath
			existing.Issuer = cert.Issuer
			existing.NotAfter = cert.NotAfter
			existing.NotBefore = cert.NotBefore
			existing.SANs = sansJSON
			existing.Status = status
			database.DB.Save(&existing)
		}
	}

	log.Printf("scanned %d certificates", len(certs))
}

func (s *Scanner) ScanSites() {
	if !s.nginx.IsAvailable() {
		log.Println("nginx not available, skipping site scan")
		return
	}

	sites, err := s.nginx.ListSites()
	if err != nil {
		log.Printf("scan sites error: %v", err)
		return
	}

	for _, site := range sites {
		var existing model.Site
		result := database.DB.Where("domain = ?", site.Domain).First(&existing)

		var certID *uint
		if site.SSLEnabled {
			var cert model.Certificate
			if database.DB.Where("domain = ?", site.Domain).First(&cert).Error == nil {
				certID = &cert.ID
			}
		}

		if result.Error != nil {
			dbSite := model.Site{
				Domain:        site.Domain,
				ConfigPath:    site.ConfigPath,
				Port:          site.Port,
				Upstream:      site.Upstream,
				SSLEnabled:    site.SSLEnabled,
				CertificateID: certID,
				Enabled:       site.Enabled,
			}
			database.DB.Create(&dbSite)
		} else {
			existing.ConfigPath = site.ConfigPath
			existing.Port = site.Port
			existing.Upstream = site.Upstream
			existing.SSLEnabled = site.SSLEnabled
			existing.CertificateID = certID
			existing.Enabled = site.Enabled
			database.DB.Save(&existing)
		}
	}

	log.Printf("scanned %d sites", len(sites))
}

func (s *Scanner) ScanAll() {
	s.ScanCertificates()
	s.ScanSites()
}

func (s *Scanner) StartPeriodicScan(interval time.Duration) {
	go func() {
		s.ScanAll()
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for range ticker.C {
			s.ScanAll()
		}
	}()
}

func joinJSON(items []string) string {
	result := "["
	for i, item := range items {
		if i > 0 {
			result += ","
		}
		result += `"` + item + `"`
	}
	result += "]"
	return result
}

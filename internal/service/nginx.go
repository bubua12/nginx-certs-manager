package service

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

type NginxService struct {
	ConfigDir string
}

func NewNginxService(configDir string) *NginxService {
	return &NginxService{ConfigDir: configDir}
}

type NginxStatus struct {
	Running bool   `json:"running"`
	Version string `json:"version"`
	PID     string `json:"pid"`
}

type SiteConfig struct {
	Domain     string
	ConfigPath string
	Enabled    bool
	Port       int
	SSLEnabled bool
	Upstream   string
}

func (s *NginxService) GetStatus() (*NginxStatus, error) {
	status := &NginxStatus{}

	cmd := exec.Command("nginx", "-v")
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err == nil {
		version := stderr.String()
		re := regexp.MustCompile(`nginx/([\d.]+)`)
		if m := re.FindStringSubmatch(version); len(m) > 1 {
			status.Version = m[1]
		}
	}

	cmd = exec.Command("pgrep", "-x", "nginx")
	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	if err := cmd.Run(); err == nil {
		pid := strings.TrimSpace(stdout.String())
		if pid != "" {
			status.Running = true
			status.PID = strings.Split(pid, "\n")[0]
		}
	}

	return status, nil
}

func (s *NginxService) Validate() (bool, string, error) {
	cmd := exec.Command("nginx", "-t")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	output := stdout.String() + stderr.String()
	return err == nil, output, nil
}

func (s *NginxService) Reload() (string, error) {
	cmd := exec.Command("nginx", "-s", "reload")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	output := stdout.String() + stderr.String()
	if err != nil {
		return output, fmt.Errorf("nginx reload failed: %w", err)
	}
	return output, nil
}

func (s *NginxService) ListSites() ([]SiteConfig, error) {
	sitesAvail := filepath.Join(s.ConfigDir, "sites-available")
	sitesEnabled := filepath.Join(s.ConfigDir, "sites-enabled")

	entries, err := os.ReadDir(sitesAvail)
	if err != nil {
		return nil, fmt.Errorf("read sites-available: %w", err)
	}

	enabledMap := make(map[string]bool)
	if enabledEntries, err := os.ReadDir(sitesEnabled); err == nil {
		for _, e := range enabledEntries {
			enabledMap[e.Name()] = true
		}
	}

	var sites []SiteConfig
	for _, entry := range entries {
		if entry.Name() == "default" {
			continue
		}
		info, err := entry.Info()
		if err != nil || info.IsDir() {
			continue
		}

		site := SiteConfig{
			Domain:     entry.Name(),
			ConfigPath: filepath.Join(sitesAvail, entry.Name()),
			Enabled:    enabledMap[entry.Name()],
		}

		s.parseSiteConfig(&site)
		sites = append(sites, site)
	}

	return sites, nil
}

func (s *NginxService) parseSiteConfig(site *SiteConfig) {
	data, err := os.ReadFile(site.ConfigPath)
	if err != nil {
		return
	}

	content := string(data)

	re := regexp.MustCompile(`server_name\s+([^;]+)`)
	if m := re.FindStringSubmatch(content); len(m) > 1 {
		site.Domain = strings.TrimSpace(m[1])
	}

	re = regexp.MustCompile(`listen\s+(\d+)`)
	if m := re.FindStringSubmatch(content); len(m) > 1 {
		fmt.Sscanf(m[1], "%d", &site.Port)
	}

	site.SSLEnabled = strings.Contains(content, "ssl_certificate")

	re = regexp.MustCompile(`proxy_pass\s+(https?://[^;]+)`)
	if m := re.FindStringSubmatch(content); len(m) > 1 {
		site.Upstream = m[1]
	}
}

func (s *NginxService) GetSiteConfig(domain string) (string, error) {
	path := filepath.Join(s.ConfigDir, "sites-available", domain)
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("read config: %w", err)
	}
	return string(data), nil
}

func (s *NginxService) SaveSiteConfig(domain string, content string) error {
	path := filepath.Join(s.ConfigDir, "sites-available", domain)
	return os.WriteFile(path, []byte(content), 0644)
}

func (s *NginxService) EnableSite(domain string) error {
	avail := filepath.Join(s.ConfigDir, "sites-available", domain)
	enabled := filepath.Join(s.ConfigDir, "sites-enabled", domain)

	if _, err := os.Stat(avail); err != nil {
		return fmt.Errorf("site config not found: %s", domain)
	}

	if _, err := os.Stat(enabled); err == nil {
		return nil
	}

	return os.Symlink(avail, enabled)
}

func (s *NginxService) DisableSite(domain string) error {
	enabled := filepath.Join(s.ConfigDir, "sites-enabled", domain)
	return os.Remove(enabled)
}

func (s *NginxService) IsAvailable() bool {
	_, err := exec.LookPath("nginx")
	return err == nil
}

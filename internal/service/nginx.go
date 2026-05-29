package service

import (
	"bufio"
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
	Domain     string `json:"domain"`
	ConfigPath string `json:"config_path"`
	Enabled    bool   `json:"enabled"`
	Port       int    `json:"port"`
	SSLEnabled bool   `json:"ssl_enabled"`
	Upstream   string `json:"upstream"`
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

// ListSites parses nginx config files to extract all server blocks
func (s *NginxService) ListSites() ([]SiteConfig, error) {
	var sites []SiteConfig

	// 1. Parse main nginx.conf
	mainConf := filepath.Join(s.ConfigDir, "nginx.conf")
	if data, err := os.ReadFile(mainConf); err == nil {
		blocks := extractServerBlocks(string(data))
		for _, block := range blocks {
			site := parseServerBlock(block, mainConf)
			if site.Domain != "" && site.Domain != "_" {
				sites = append(sites, site)
			}
		}
	}

	// 2. Parse conf.d/*.conf
	confD := filepath.Join(s.ConfigDir, "conf.d")
	if entries, err := os.ReadDir(confD); err == nil {
		for _, entry := range entries {
			if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".conf") {
				continue
			}
			path := filepath.Join(confD, entry.Name())
			data, err := os.ReadFile(path)
			if err != nil {
				continue
			}
			blocks := extractServerBlocks(string(data))
			for _, block := range blocks {
				site := parseServerBlock(block, path)
				if site.Domain != "" && site.Domain != "_" {
					sites = append(sites, site)
				}
			}
		}
	}

	// 3. Parse sites-available/ (traditional layout)
	sitesAvail := filepath.Join(s.ConfigDir, "sites-available")
	sitesEnabled := filepath.Join(s.ConfigDir, "sites-enabled")
	enabledMap := make(map[string]bool)
	if entries, err := os.ReadDir(sitesEnabled); err == nil {
		for _, e := range entries {
			enabledMap[e.Name()] = true
		}
	}
	if entries, err := os.ReadDir(sitesAvail); err == nil {
		for _, entry := range entries {
			info, err := entry.Info()
			if err != nil || info.IsDir() {
				continue
			}
			path := filepath.Join(sitesAvail, entry.Name())
			data, err := os.ReadFile(path)
			if err != nil {
				continue
			}
			blocks := extractServerBlocks(string(data))
			for _, block := range blocks {
				site := parseServerBlock(block, path)
				site.Enabled = enabledMap[entry.Name()]
				if site.Domain != "" && site.Domain != "_" {
					sites = append(sites, site)
				}
			}
		}
	}

	// Deduplicate by domain: prefer SSL-enabled block over non-SSL (HTTP redirect)
	seen := make(map[string]int) // domain -> index in result
	var result []SiteConfig
	for _, site := range sites {
		idx, exists := seen[site.Domain]
		if !exists {
			// First occurrence of this domain
			seen[site.Domain] = len(result)
			result = append(result, site)
		} else if site.SSLEnabled && !result[idx].SSLEnabled {
			// Replace non-SSL with SSL version (prefer the actual HTTPS server block)
			result[idx] = site
		}
		// If existing is already SSL, skip the new one (HTTP redirect block)
	}

	return result, nil
}

// GetSiteConfig returns the server block content for a domain
func (s *NginxService) GetSiteConfig(domain string) (string, error) {
	// Search in all config files
	files := s.getConfigFiles()
	for _, path := range files {
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		blocks := extractServerBlocks(string(data))
		for _, block := range blocks {
			site := parseServerBlock(block, path)
			if site.Domain == domain {
				return block, nil
			}
		}
	}
	return "", fmt.Errorf("server block for %s not found", domain)
}

// SaveSiteConfig updates a server block in the config file
func (s *NginxService) SaveSiteConfig(domain string, newBlock string) error {
	files := s.getConfigFiles()
	for _, path := range files {
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		content := string(data)
		blocks := extractServerBlocksWithPositions(content)
		for _, bp := range blocks {
			site := parseServerBlock(bp.Content, path)
			if site.Domain == domain {
				// Replace the old block with new one
				updated := content[:bp.Start] + newBlock + content[bp.End:]
				return os.WriteFile(path, []byte(updated), 0644)
			}
		}
	}
	return fmt.Errorf("server block for %s not found", domain)
}

// EnableSite enables a server block by uncommenting it
func (s *NginxService) EnableSite(domain string) error {
	return s.setSiteEnabled(domain, true)
}

// DisableSite disables a server block by commenting it out
func (s *NginxService) DisableSite(domain string) error {
	return s.setSiteEnabled(domain, false)
}

func (s *NginxService) setSiteEnabled(domain string, enabled bool) error {
	// Try sites-enabled first (traditional layout)
	avail := filepath.Join(s.ConfigDir, "sites-available", domain)
	if _, err := os.Stat(avail); err == nil {
		link := filepath.Join(s.ConfigDir, "sites-enabled", domain)
		if enabled {
			os.Symlink(avail, link)
		} else {
			os.Remove(link)
		}
		return nil
	}

	// For nginx.conf / conf.d, we note the site is managed in the main config
	// Enable/disable isn't applicable for inline server blocks
	return fmt.Errorf("site %s is managed in the main nginx config, enable/disable not supported for inline blocks", domain)
}

func (s *NginxService) IsAvailable() bool {
	_, err := exec.LookPath("nginx")
	return err == nil
}

func (s *NginxService) getConfigFiles() []string {
	var files []string
	mainConf := filepath.Join(s.ConfigDir, "nginx.conf")
	if _, err := os.Stat(mainConf); err == nil {
		files = append(files, mainConf)
	}
	confD := filepath.Join(s.ConfigDir, "conf.d")
	if entries, err := os.ReadDir(confD); err == nil {
		for _, e := range entries {
			if !e.IsDir() && strings.HasSuffix(e.Name(), ".conf") {
				files = append(files, filepath.Join(confD, e.Name()))
			}
		}
	}
	sitesAvail := filepath.Join(s.ConfigDir, "sites-available")
	if entries, err := os.ReadDir(sitesAvail); err == nil {
		for _, e := range entries {
			info, err := e.Info()
			if err == nil && !info.IsDir() {
				files = append(files, filepath.Join(sitesAvail, e.Name()))
			}
		}
	}
	return files
}

type blockPosition struct {
	Content string
	Start   int
	End     int
}

// extractServerBlocks extracts server { ... } blocks from config text
func extractServerBlocks(content string) []string {
	var blocks []string
	scanner := bufio.NewScanner(strings.NewReader(content))

	var currentBlock strings.Builder
	depth := 0
	inServer := false

	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)

		if !inServer {
			if strings.HasPrefix(trimmed, "#") && strings.Contains(trimmed, "server") {
				// Check if next non-comment line is a server block (commented out)
				continue
			}
			if strings.HasPrefix(trimmed, "server") && (trimmed == "server" || strings.HasPrefix(trimmed, "server{") || strings.HasPrefix(trimmed, "server {")) {
				inServer = true
				currentBlock.Reset()
				currentBlock.WriteString(line)
				currentBlock.WriteString("\n")
				depth = 0
				for _, ch := range line {
					if ch == '{' {
						depth++
					}
					if ch == '}' {
						depth--
					}
				}
				if depth == 0 && strings.Contains(trimmed, "{") && strings.Contains(trimmed, "}") {
					blocks = append(blocks, currentBlock.String())
					inServer = false
				}
				continue
			}
		}

		if inServer {
			currentBlock.WriteString(line)
			currentBlock.WriteString("\n")
			for _, ch := range line {
				if ch == '{' {
					depth++
				}
				if ch == '}' {
					depth--
				}
			}
			if depth <= 0 {
				blocks = append(blocks, currentBlock.String())
				inServer = false
			}
		}
	}

	return blocks
}

// extractServerBlocksWithPositions returns blocks with their byte positions
func extractServerBlocksWithPositions(content string) []blockPosition {
	var positions []blockPosition
	lines := strings.Split(content, "\n")

	bytePos := 0
	inServer := false
	depth := 0
	blockStart := 0
	var blockContent strings.Builder

	for _, line := range lines {
		lineWithNewline := line + "\n"
		trimmed := strings.TrimSpace(line)

		if !inServer {
			if strings.HasPrefix(trimmed, "server") && (trimmed == "server" || strings.HasPrefix(trimmed, "server{") || strings.HasPrefix(trimmed, "server {")) {
				inServer = true
				blockStart = bytePos
				blockContent.Reset()
				blockContent.WriteString(lineWithNewline)
				depth = 0
				for _, ch := range line {
					if ch == '{' {
						depth++
					}
					if ch == '}' {
						depth--
					}
				}
				if depth == 0 && strings.Contains(trimmed, "{") && strings.Contains(trimmed, "}") {
					positions = append(positions, blockPosition{
						Content: blockContent.String(),
						Start:   blockStart,
						End:     bytePos + len(lineWithNewline),
					})
					inServer = false
				}
			}
		} else {
			blockContent.WriteString(lineWithNewline)
			for _, ch := range line {
				if ch == '{' {
					depth++
				}
				if ch == '}' {
					depth--
				}
			}
			if depth <= 0 {
				positions = append(positions, blockPosition{
					Content: blockContent.String(),
					Start:   blockStart,
					End:     bytePos + len(lineWithNewline),
				})
				inServer = false
			}
		}

		bytePos += len(lineWithNewline)
	}

	return positions
}

// parseServerBlock extracts site info from a server block string
func parseServerBlock(block string, configPath string) SiteConfig {
	site := SiteConfig{
		ConfigPath: configPath,
		Enabled:    true,
	}

	re := regexp.MustCompile(`server_name\s+([^;]+)`)
	if m := re.FindStringSubmatch(block); len(m) > 1 {
		site.Domain = strings.TrimSpace(m[1])
		// Take first domain if multiple
		parts := strings.Fields(site.Domain)
		if len(parts) > 0 {
			site.Domain = parts[0]
		}
	}

	re = regexp.MustCompile(`listen\s+(\d+)`)
	if m := re.FindStringSubmatch(block); len(m) > 1 {
		fmt.Sscanf(m[1], "%d", &site.Port)
	}

	site.SSLEnabled = strings.Contains(block, "ssl_certificate")

	re = regexp.MustCompile(`proxy_pass\s+(https?://[^;]+)`)
	if m := re.FindStringSubmatch(block); len(m) > 1 {
		site.Upstream = m[1]
	}

	return site
}

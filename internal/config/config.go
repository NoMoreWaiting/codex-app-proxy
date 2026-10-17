// Package config loads, validates and persists the portable config.json.
package config

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ProxyType is the scheme passed to Chromium via --proxy-server.
type ProxyType string

const (
	ProxyHTTP   ProxyType = "http"
	ProxySOCKS5 ProxyType = "socks5"
)

// LogMode controls how much is written to the log file.
type LogMode string

const (
	LogErrors LogMode = "errors"
	LogAlways LogMode = "always"
	LogOff    LogMode = "off"
)

// DefaultPort matches the mixed port most Clash/mihomo builds expose.
const DefaultPort = 10808

// Proxy is the local proxy endpoint.
type Proxy struct {
	Type ProxyType `json:"type"`
	Host string    `json:"host"`
	Port int       `json:"port"`
}

// Config mirrors the documented config.json schema.
type Config struct {
	Version     int      `json:"_version"`
	Comment     string   `json:"_comment,omitempty"`
	Proxy       Proxy    `json:"proxy"`
	BypassList  []string `json:"bypass_list"`
	DisableQuic bool     `json:"disable_quic"`
	LogMode     LogMode  `json:"log_mode"`
}

// DefaultBypassList lists the ranges that must never go through the proxy.
//
// Note: "<-loopback>" is deliberately absent. That token tells Chromium to
// stop bypassing loopback addresses, so the app's own local exec-server and
// extension-host traffic gets pushed into the proxy. On the current App that
// breaks workspace-policy loading and surfaces as "Organization settings
// could not be loaded". See SanitizeBypass.
func DefaultBypassList() []string {
	list := []string{"localhost", "127.0.0.1", "::1", "10.*"}
	for i := 16; i <= 31; i++ {
		list = append(list, fmt.Sprintf("172.%d.*", i))
	}
	return append(list, "192.168.*")
}

// Default returns the factory configuration.
func Default() Config {
	return Config{
		Version:     1,
		Comment:     "ChatGPT/Codex Proxy Launcher configuration",
		Proxy:       Proxy{Type: ProxyHTTP, Host: "127.0.0.1", Port: DefaultPort},
		BypassList:  DefaultBypassList(),
		DisableQuic: true,
		LogMode:     LogErrors,
	}
}

// URL renders the proxy as scheme://host:port, bracketing IPv6 literals.
func (p Proxy) URL() string {
	return fmt.Sprintf("%s://%s", p.Type, p.Address())
}

// Address renders host:port, bracketing IPv6 literals.
func (p Proxy) Address() string {
	host := p.Host
	if strings.Contains(host, ":") && !strings.HasPrefix(host, "[") {
		host = "[" + host + "]"
	}
	return fmt.Sprintf("%s:%d", host, p.Port)
}

// SanitizeBypass drops entries that would break the app, de-duplicates the
// rest and reports how many dangerous entries were removed.
func SanitizeBypass(list []string) (clean []string, removed int) {
	seen := make(map[string]struct{}, len(list))
	clean = make([]string, 0, len(list))
	for _, entry := range list {
		item := strings.TrimSpace(entry)
		if item == "" {
			continue
		}
		if strings.EqualFold(item, "<-loopback>") {
			removed++
			continue
		}
		key := strings.ToLower(item)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		clean = append(clean, item)
	}
	return clean, removed
}

// Validate reports configuration problems the user must fix.
func (c Config) Validate() error {
	switch c.Proxy.Type {
	case ProxyHTTP, ProxySOCKS5:
	default:
		return fmt.Errorf("proxy.type 只能是 http 或 socks5，当前为 %q", c.Proxy.Type)
	}
	if strings.TrimSpace(c.Proxy.Host) == "" {
		return errors.New("proxy.host 不能为空")
	}
	if c.Proxy.Port < 1 || c.Proxy.Port > 65535 {
		return fmt.Errorf("proxy.port 必须在 1-65535 之间，当前为 %d", c.Proxy.Port)
	}
	switch c.LogMode {
	case LogErrors, LogAlways, LogOff:
	default:
		return fmt.Errorf("log_mode 只能是 errors、always 或 off，当前为 %q", c.LogMode)
	}
	return nil
}

// LoadResult carries what Load had to change on disk.
type LoadResult struct {
	Config    Config
	Created   bool
	Sanitized int
}

// Load reads config.json, creating it from Default when missing.
// Unknown keys are ignored so newer config files keep working.
func Load(path string) (LoadResult, error) {
	var result LoadResult

	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		cfg := Default()
		if saveErr := Save(path, cfg); saveErr != nil {
			return result, fmt.Errorf("无法创建默认配置 %s: %w", path, saveErr)
		}
		result.Config = cfg
		result.Created = true
		return result, nil
	}
	if err != nil {
		return result, fmt.Errorf("无法读取 %s: %w", path, err)
	}

	cfg := Default()
	if err := json.Unmarshal(data, &cfg); err != nil {
		return result, fmt.Errorf("config.json 不是合法 JSON: %w", err)
	}

	applyDefaults(&cfg)
	cfg.BypassList, result.Sanitized = SanitizeBypass(cfg.BypassList)
	if err := cfg.Validate(); err != nil {
		return result, err
	}

	result.Config = cfg
	return result, nil
}

func applyDefaults(c *Config) {
	if c.Version == 0 {
		c.Version = 1
	}
	if c.Proxy.Type == "" {
		c.Proxy.Type = ProxyHTTP
	}
	if strings.TrimSpace(c.Proxy.Host) == "" {
		c.Proxy.Host = "127.0.0.1"
	}
	if c.Proxy.Port == 0 {
		c.Proxy.Port = DefaultPort
	}
	if c.LogMode == "" {
		c.LogMode = LogErrors
	}
}

// Save writes config.json as readable UTF-8 without HTML escaping.
func Save(path string, cfg Config) error {
	if dir := filepath.Dir(path); dir != "" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
	}

	var buffer bytes.Buffer
	encoder := json.NewEncoder(&buffer)
	encoder.SetEscapeHTML(false)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(cfg); err != nil {
		return err
	}
	return os.WriteFile(path, buffer.Bytes(), 0o644)
}

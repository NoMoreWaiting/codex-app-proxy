package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultBypassListOmitsLoopbackOverride(t *testing.T) {
	for _, entry := range Default().BypassList {
		if entry == "<-loopback>" {
			t.Fatalf("默认 bypass_list 不能包含 <-loopback>，它会让 Chromium 把本地回环流量也送进代理")
		}
	}
}

func TestSanitizeBypassRemovesLoopbackOverride(t *testing.T) {
	clean, removed := SanitizeBypass([]string{
		"<-loopback>", "localhost", "127.0.0.1", "<-LOOPBACK>", "  ", "localhost",
	})
	if removed != 2 {
		t.Fatalf("removed = %d, want 2", removed)
	}
	want := []string{"localhost", "127.0.0.1"}
	if len(clean) != len(want) {
		t.Fatalf("clean = %v, want %v", clean, want)
	}
	for i := range want {
		if clean[i] != want[i] {
			t.Fatalf("clean = %v, want %v", clean, want)
		}
	}
}

func TestLoadMigratesLegacyLoopbackEntry(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	legacy := `{
  "_version": 1,
  "proxy": {"type": "http", "host": "127.0.0.1", "port": 10808},
  "bypass_list": ["<-loopback>", "localhost", "127.0.0.1"],
  "disable_quic": true,
  "log_mode": "errors"
}`
	if err := os.WriteFile(path, []byte(legacy), 0o644); err != nil {
		t.Fatal(err)
	}

	result, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if result.Sanitized != 1 {
		t.Fatalf("Sanitized = %d, want 1", result.Sanitized)
	}
	for _, entry := range result.Config.BypassList {
		if entry == "<-loopback>" {
			t.Fatal("加载后 bypass_list 仍包含 <-loopback>")
		}
	}
}

func TestLoadCreatesMissingFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	result, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if !result.Created {
		t.Fatal("Created = false, want true")
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("配置文件未写入: %v", err)
	}
	if result.Config.Proxy.Port != DefaultPort {
		t.Fatalf("Port = %d, want %d", result.Config.Proxy.Port, DefaultPort)
	}
}

func TestLoadFillsPartialProxyBlock(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(path, []byte(`{"proxy": {"port": 7897}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	result, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if result.Config.Proxy.Port != 7897 {
		t.Fatalf("Port = %d, want 7897", result.Config.Proxy.Port)
	}
	if result.Config.Proxy.Type != ProxyHTTP || result.Config.Proxy.Host != "127.0.0.1" {
		t.Fatalf("缺失字段未补默认值: %+v", result.Config.Proxy)
	}
}

func TestLoadRejectsInvalidPort(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(path, []byte(`{"proxy": {"port": 70000}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(path); err == nil {
		t.Fatal("端口越界应当报错")
	}
}

func TestProxyURLBracketsIPv6(t *testing.T) {
	got := Proxy{Type: ProxySOCKS5, Host: "::1", Port: 1080}.URL()
	if got != "socks5://[::1]:1080" {
		t.Fatalf("URL = %q", got)
	}
	got = Proxy{Type: ProxyHTTP, Host: "127.0.0.1", Port: 10808}.URL()
	if got != "http://127.0.0.1:10808" {
		t.Fatalf("URL = %q", got)
	}
}

func TestSaveLoadRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	want := Default()
	want.Proxy.Port = 7890
	want.LogMode = LogAlways
	if err := Save(path, want); err != nil {
		t.Fatalf("Save: %v", err)
	}
	result, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if result.Config.Proxy.Port != 7890 || result.Config.LogMode != LogAlways {
		t.Fatalf("往返后不一致: %+v", result.Config)
	}
	if result.Created {
		t.Fatal("Created = true, want false")
	}
}

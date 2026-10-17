package launch

import (
	"strings"
	"testing"

	"codex-app-proxy/internal/config"
)

func testConfig() config.Config {
	return config.Config{
		Proxy:       config.Proxy{Type: config.ProxyHTTP, Host: "127.0.0.1", Port: 10808},
		BypassList:  []string{"localhost", "127.0.0.1", "::1"},
		DisableQuic: true,
		LogMode:     config.LogErrors,
	}
}

func TestArgumentsBuildsProxySwitches(t *testing.T) {
	got := Arguments(testConfig())

	if !strings.Contains(got, `--proxy-server="http://127.0.0.1:10808"`) {
		t.Fatalf("缺少 --proxy-server：%s", got)
	}
	if !strings.Contains(got, `--proxy-bypass-list="localhost;127.0.0.1;::1"`) {
		t.Fatalf("bypass 列表不正确：%s", got)
	}
	if !strings.Contains(got, "--disable-quic") {
		t.Fatalf("缺少 --disable-quic：%s", got)
	}
}

func TestArgumentsNeverForwardsLoopbackOverride(t *testing.T) {
	cfg := testConfig()
	cfg.BypassList = []string{"<-loopback>", "localhost", "127.0.0.1"}

	got := Arguments(cfg)
	if strings.Contains(got, "loopback>") {
		t.Fatalf("<-loopback> 绝不能传给 Chromium：%s", got)
	}
	if !strings.Contains(got, `--proxy-bypass-list="localhost;127.0.0.1"`) {
		t.Fatalf("过滤后应保留其余条目：%s", got)
	}
}

func TestArgumentsOmitsBypassListWhenEmpty(t *testing.T) {
	cfg := testConfig()
	cfg.BypassList = []string{"<-loopback>", "   "}

	got := Arguments(cfg)
	if strings.Contains(got, "--proxy-bypass-list") {
		t.Fatalf("空 bypass 时不应输出该开关：%s", got)
	}
}

func TestArgumentsRespectsDisableQuicToggle(t *testing.T) {
	cfg := testConfig()
	cfg.DisableQuic = false

	if strings.Contains(Arguments(cfg), "--disable-quic") {
		t.Fatal("disable_quic=false 时不应添加 --disable-quic")
	}
}

func TestArgumentsUsesSocks5Scheme(t *testing.T) {
	cfg := testConfig()
	cfg.Proxy.Type = config.ProxySOCKS5
	cfg.Proxy.Port = 7897

	got := Arguments(cfg)
	if !strings.Contains(got, `--proxy-server="socks5://127.0.0.1:7897"`) {
		t.Fatalf("SOCKS5 方案不正确：%s", got)
	}
}

func TestFilterBypassTrimsAndDeduplicatesInput(t *testing.T) {
	got := FilterBypass([]string{" localhost ", "", "  ", "<-LOOPBACK>"})
	if len(got) != 1 || got[0] != "localhost" {
		t.Fatalf("FilterBypass = %v", got)
	}
}

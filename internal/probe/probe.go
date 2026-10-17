// Package probe checks that the configured endpoint really speaks proxy.
package probe

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"strconv"
	"strings"
	"syscall"
	"time"

	"codex-app-proxy/internal/config"
)

// Result classifies the probe outcome.
type Result int

const (
	// OK means the endpoint completed a proxy handshake.
	OK Result = iota
	// Unreachable means the TCP connection was refused or failed.
	Unreachable
	// Timeout means the TCP connection or handshake timed out.
	Timeout
	// NotAProxy means something is listening but it is not a proxy.
	NotAProxy
	// AuthRequired means the proxy rejected the anonymous handshake.
	AuthRequired
)

// Code returns the stable error code shown to the user.
func (r Result) Code() string {
	switch r {
	case OK:
		return "E_PROXY_OK"
	case NotAProxy:
		return "E_PROXY_NOT_A_PROXY"
	case AuthRequired:
		return "E_PROXY_AUTH_REQUIRED"
	case Timeout:
		return "E_PROXY_TIMEOUT"
	default:
		return "E_PROXY_UNREACHABLE"
	}
}

// Title returns a short human readable summary.
func (r Result) Title() string {
	switch r {
	case OK:
		return "本地代理可用"
	case NotAProxy:
		return "配置的端口不是代理端口"
	case AuthRequired:
		return "本地代理要求认证"
	case Timeout:
		return "本地代理握手超时"
	default:
		return "无法连接本地代理"
	}
}

// Advice returns actionable guidance for a failure.
func (r Result) Advice() string {
	switch r {
	case NotAProxy:
		return "这个端口能连通，但没有按 HTTP CONNECT 或 SOCKS5 协议应答。\n" +
			"常见原因是把「外部控制端口」「API 端口」或「订阅端口」当成了代理端口。\n" +
			"请填写代理软件界面显示的本地 HTTP / mixed / SOCKS5 监听端口。"
	case AuthRequired:
		return "代理要求认证。请在代理软件里关闭该端口的认证，" +
			"或改用不需要认证的 mixed 端口。"
	case Timeout:
		return "端口可以连接，但握手没有响应。请确认代理软件已启动且节点可用。"
	case OK:
		return ""
	default:
		return "请依次检查：\n" +
			"1. 代理软件是否正在运行；\n" +
			"2. 填写的是否是本地 HTTP、mixed 或 SOCKS5 监听端口；\n" +
			"3. 端口是否与代理软件界面显示的一致。\n\n" +
			"不要填写节点服务器端口、订阅端口或远程 443。"
	}
}

// Proxy performs a real handshake against the configured proxy.
// The second return value carries diagnostic detail for the log and dialog.
func Proxy(p config.Proxy, timeout time.Duration) (Result, string) {
	address := p.Address()

	conn, err := net.DialTimeout("tcp", address, timeout)
	if err != nil {
		if isTimeout(err) {
			return Timeout, fmt.Sprintf("连接 %s 超时：%v", address, err)
		}
		return Unreachable, fmt.Sprintf("无法连接 %s：%v", address, err)
	}
	defer func() { _ = conn.Close() }()
	_ = conn.SetDeadline(time.Now().Add(timeout))

	if p.Type == config.ProxySOCKS5 {
		return probeSOCKS5(conn)
	}
	return probeHTTP(conn)
}

func probeHTTP(conn net.Conn) (Result, string) {
	const request = "CONNECT example.com:443 HTTP/1.1\r\n" +
		"Host: example.com:443\r\n" +
		"Proxy-Connection: Keep-Alive\r\n\r\n"

	if _, err := io.WriteString(conn, request); err != nil {
		return NotAProxy, "端口接受连接但不接受 HTTP CONNECT：" + err.Error()
	}

	line, err := bufio.NewReader(conn).ReadString('\n')
	if err != nil {
		return NotAProxy, "端口没有返回 HTTP 响应：" + err.Error()
	}
	line = strings.TrimSpace(line)
	if !strings.HasPrefix(line, "HTTP/") {
		return NotAProxy, "返回内容不是 HTTP 代理响应：" + truncate(line, 80)
	}

	fields := strings.Fields(line)
	if len(fields) < 2 {
		return NotAProxy, "无法解析代理响应：" + truncate(line, 80)
	}
	status, convErr := strconv.Atoi(fields[1])
	if convErr != nil {
		return NotAProxy, "无法解析状态码：" + truncate(line, 80)
	}

	switch status {
	case 200:
		return OK, "HTTP CONNECT 握手成功（" + line + "）"
	case 407:
		return AuthRequired, "代理要求认证（" + line + "）"
	default:
		return NotAProxy, fmt.Sprintf("端口返回 %d，不是可用的 HTTP 代理：%s", status, truncate(line, 80))
	}
}

func probeSOCKS5(conn net.Conn) (Result, string) {
	// Version 5, one method offered: no authentication.
	if _, err := conn.Write([]byte{0x05, 0x01, 0x00}); err != nil {
		return NotAProxy, "无法写入 SOCKS5 握手：" + err.Error()
	}

	response := make([]byte, 2)
	if _, err := io.ReadFull(conn, response); err != nil {
		return NotAProxy, "没有收到 SOCKS5 响应：" + err.Error()
	}
	if response[0] != 0x05 {
		return NotAProxy, fmt.Sprintf("不是 SOCKS5 服务（版本字节 0x%02X）", response[0])
	}

	switch response[1] {
	case 0x00:
		return OK, "SOCKS5 握手成功（无需认证）"
	case 0xFF:
		return AuthRequired, "SOCKS5 代理拒绝了匿名握手，需要认证"
	default:
		return NotAProxy, fmt.Sprintf("SOCKS5 返回了不支持的认证方式 0x%02X", response[1])
	}
}

func isTimeout(err error) bool {
	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return true
	}
	return errors.Is(err, syscall.ETIMEDOUT) || errors.Is(err, os.ErrDeadlineExceeded)
}

func truncate(text string, limit int) string {
	if len(text) <= limit {
		return text
	}
	return text[:limit] + "…"
}

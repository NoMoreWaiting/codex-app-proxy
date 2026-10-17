package probe

import (
	"fmt"
	"io"
	"net"
	"testing"
	"time"

	"codex-app-proxy/internal/config"
)

func startFake(t *testing.T, handler func(net.Conn)) (config.Proxy, func()) {
	t.Helper()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}

	done := make(chan struct{})
	go func() {
		defer close(done)
		for {
			conn, acceptErr := listener.Accept()
			if acceptErr != nil {
				return
			}
			go func(c net.Conn) {
				defer func() { _ = c.Close() }()
				handler(c)
			}(conn)
		}
	}()

	addr, ok := listener.Addr().(*net.TCPAddr)
	if !ok {
		t.Fatal("unexpected listener address type")
	}
	proxy := config.Proxy{Type: config.ProxyHTTP, Host: "127.0.0.1", Port: addr.Port}
	return proxy, func() {
		_ = listener.Close()
		<-done
	}
}

func serveHTTP(status string) func(net.Conn) {
	return func(conn net.Conn) {
		// Drain the CONNECT request headers.
		buffer := make([]byte, 0, 512)
		chunk := make([]byte, 1)
		for {
			if _, err := conn.Read(chunk); err != nil {
				return
			}
			buffer = append(buffer, chunk[0])
			if len(buffer) >= 4 && string(buffer[len(buffer)-4:]) == "\r\n\r\n" {
				break
			}
		}
		_, _ = fmt.Fprintf(conn, "HTTP/1.1 %s\r\n\r\n", status)
		time.Sleep(20 * time.Millisecond)
	}
}

func TestHTTPProxyOK(t *testing.T) {
	proxy, stop := startFake(t, serveHTTP("200 Connection Established"))
	defer stop()

	result, detail := Proxy(proxy, time.Second)
	if result != OK {
		t.Fatalf("result = %v (%s), want OK", result, detail)
	}
}

func TestHTTPProxyAuthRequired(t *testing.T) {
	proxy, stop := startFake(t, serveHTTP("407 Proxy Authentication Required"))
	defer stop()

	result, _ := Proxy(proxy, time.Second)
	if result != AuthRequired {
		t.Fatalf("result = %v, want AuthRequired", result)
	}
}

func TestHTTPProxyWrongStatus(t *testing.T) {
	proxy, stop := startFake(t, serveHTTP("500 Internal Server Error"))
	defer stop()

	result, _ := Proxy(proxy, time.Second)
	if result != NotAProxy {
		t.Fatalf("result = %v, want NotAProxy", result)
	}
}

func TestNonHTTPServiceIsNotAProxy(t *testing.T) {
	proxy, stop := startFake(t, func(conn net.Conn) {
		_, _ = conn.Write([]byte("SSH-2.0-OpenSSH_9.0\r\n"))
		time.Sleep(20 * time.Millisecond)
	})
	defer stop()

	result, _ := Proxy(proxy, time.Second)
	if result != NotAProxy {
		t.Fatalf("result = %v, want NotAProxy", result)
	}
}

func TestSOCKS5OK(t *testing.T) {
	proxy, stop := startFake(t, func(conn net.Conn) {
		greeting := make([]byte, 2)
		if _, err := io.ReadFull(conn, greeting); err != nil {
			return
		}
		_, _ = conn.Write([]byte{0x05, 0x00})
		time.Sleep(20 * time.Millisecond)
	})
	defer stop()

	proxy.Type = config.ProxySOCKS5
	result, detail := Proxy(proxy, time.Second)
	if result != OK {
		t.Fatalf("result = %v (%s), want OK", result, detail)
	}
}

func TestSOCKS5AuthRequired(t *testing.T) {
	proxy, stop := startFake(t, func(conn net.Conn) {
		greeting := make([]byte, 2)
		if _, err := io.ReadFull(conn, greeting); err != nil {
			return
		}
		_, _ = conn.Write([]byte{0x05, 0xFF})
		time.Sleep(20 * time.Millisecond)
	})
	defer stop()

	proxy.Type = config.ProxySOCKS5
	result, _ := Proxy(proxy, time.Second)
	if result != AuthRequired {
		t.Fatalf("result = %v, want AuthRequired", result)
	}
}

func TestClosedPortIsUnreachable(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	addr, ok := listener.Addr().(*net.TCPAddr)
	if !ok {
		t.Fatal("unexpected listener address type")
	}
	_ = listener.Close()

	proxy := config.Proxy{Type: config.ProxyHTTP, Host: "127.0.0.1", Port: addr.Port}
	result, _ := Proxy(proxy, 500*time.Millisecond)
	if result != Unreachable {
		t.Fatalf("result = %v, want Unreachable", result)
	}
}

func TestResultCodesAreStable(t *testing.T) {
	cases := map[Result]string{
		OK:           "E_PROXY_OK",
		Unreachable:  "E_PROXY_UNREACHABLE",
		Timeout:      "E_PROXY_TIMEOUT",
		NotAProxy:    "E_PROXY_NOT_A_PROXY",
		AuthRequired: "E_PROXY_AUTH_REQUIRED",
	}
	for result, want := range cases {
		if got := result.Code(); got != want {
			t.Fatalf("Code() = %q, want %q", got, want)
		}
	}
}

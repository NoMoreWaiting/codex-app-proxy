// Package launch builds the app command line and starts the packaged app.
package launch

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"

	"codex-app-proxy/internal/app"
	"codex-app-proxy/internal/config"
	"codex-app-proxy/internal/win32"
)

// COM identities for IApplicationActivationManager.
const (
	clsidApplicationActivationManager = "{45BA127D-10A8-46EA-8AB7-56EA9078943C}"
	iidApplicationActivationManager   = "{2e941141-7f97-4756-ba1d-9decde894a3d}"
)

// Arguments builds the Chromium switch string passed to the app.
func Arguments(cfg config.Config) string {
	var builder strings.Builder

	builder.WriteString(`--proxy-server="`)
	builder.WriteString(cfg.Proxy.URL())
	builder.WriteString(`"`)

	if bypass := FilterBypass(cfg.BypassList); len(bypass) > 0 {
		builder.WriteString(` --proxy-bypass-list="`)
		builder.WriteString(strings.Join(bypass, ";"))
		builder.WriteString(`"`)
	}

	if cfg.DisableQuic {
		builder.WriteString(" --disable-quic")
	}
	return builder.String()
}

// FilterBypass removes entries that break the app's own local traffic.
//
// "<-loopback>" cancels Chromium's implicit loopback bypass. The current app
// talks to its own local exec-server and extension host, so pushing that
// traffic into the proxy makes workspace-policy loading fail and the UI shows
// "Organization settings could not be loaded". The switch is never forwarded,
// no matter what an older config.json still contains.
func FilterBypass(list []string) []string {
	filtered := make([]string, 0, len(list))
	for _, entry := range list {
		item := strings.TrimSpace(entry)
		if item == "" {
			continue
		}
		if strings.EqualFold(item, "<-loopback>") {
			continue
		}
		filtered = append(filtered, item)
	}
	return filtered
}

// Activate starts the app through packaged (AUMID) activation with arguments.
func Activate(exePath, arguments string) (uint32, error) {
	aumid, err := app.AUMID(exePath)
	if err != nil {
		return 0, err
	}

	clsid, err := win32.GUIDFromString(clsidApplicationActivationManager)
	if err != nil {
		return 0, err
	}
	iid, err := win32.GUIDFromString(iidApplicationActivationManager)
	if err != nil {
		return 0, err
	}

	// A COM apartment belongs to one OS thread. Without pinning the goroutine,
	// the scheduler may move it between CoInitializeEx and the actual call and
	// the activation fails with CO_E_NOTINITIALIZED (0x800401F0).
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	if err := win32.CoInitializeEx(); err != nil {
		return 0, err
	}
	defer win32.CoUninitialize()

	manager, err := win32.CoCreateInstance(clsid, iid)
	if err != nil {
		return 0, err
	}
	defer win32.Release(manager)

	pid, err := win32.ActivateApplication(manager, aumid, arguments)
	if err != nil {
		return 0, fmt.Errorf("激活 %s 失败：%w", aumid, err)
	}
	return pid, nil
}

// StartProcess is the fallback used when packaged activation is unavailable.
// Modern builds reject a bare CreateProcess on the packaged executable, so this
// is best effort only.
func StartProcess(exePath, arguments string) (uint32, error) {
	commandLine := `"` + exePath + `"`
	if arguments != "" {
		commandLine += " " + arguments
	}

	cmd := exec.Command(exePath)
	cmd.Dir = filepath.Dir(exePath)
	cmd.SysProcAttr = &syscall.SysProcAttr{CmdLine: commandLine}

	if err := cmd.Start(); err != nil {
		return 0, err
	}
	pid := uint32(cmd.Process.Pid)
	_ = cmd.Process.Release()
	return pid, nil
}

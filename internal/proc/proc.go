// Package proc finds and terminates the desktop app's resident processes.
package proc

import (
	"strings"

	"codex-app-proxy/internal/win32"
)

// appExecutables are the process names owned by the desktop app.
//
// The current app keeps a permanent background presence, so the launcher has
// to clear these before a cold start; otherwise packaged activation only
// wakes the existing instance and silently discards the proxy arguments.
var appExecutables = map[string]struct{}{
	"chatgpt.exe":                  {},
	"codex.exe":                    {},
	"codex-command-runner.exe":     {},
	"codex-computer-use-swift.exe": {},
}

// IsAppProcess reports whether p belongs to the desktop app.
func IsAppProcess(p win32.Process) bool {
	if _, ok := appExecutables[p.Name]; !ok {
		return false
	}
	// Without a readable path, fall back to the name alone: killing one
	// extra instance is far better than silently keeping a proxyless one.
	if p.Path == "" {
		return true
	}
	lower := strings.ToLower(p.Path)
	return strings.Contains(lower, `\windowsapps\openai.codex_`) ||
		strings.Contains(lower, `\appdata\local\openai\`)
}

// AppProcesses lists the live instances of the desktop app.
func AppProcesses() ([]win32.Process, error) {
	all, err := win32.Processes()
	if err != nil {
		return nil, err
	}
	matches := make([]win32.Process, 0, 8)
	for _, p := range all {
		if IsAppProcess(p) {
			matches = append(matches, p)
		}
	}
	return matches, nil
}

// TerminateAll kills pids and returns those that survived.
func TerminateAll(pids []uint32, waitMs uint32) []uint32 {
	remaining := make([]uint32, 0, len(pids))
	for _, pid := range pids {
		if err := win32.Terminate(pid, waitMs); err != nil && Alive(pid) {
			remaining = append(remaining, pid)
		}
	}
	return remaining
}

// Alive reports whether pid can still be opened.
func Alive(pid uint32) bool {
	handle, err := win32.OpenProcess(win32.PROCESS_QUERY_LIMITED_INFORMATION, pid)
	if err != nil {
		return false
	}
	win32.CloseHandle(handle)
	return true
}

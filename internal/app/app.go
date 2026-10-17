// Package app locates the installed Microsoft Store build of ChatGPT/Codex.
package app

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"codex-app-proxy/internal/win32"
)

const (
	// PackageFamilyName is the identity of the OpenAI desktop app.
	PackageFamilyName = "OpenAI.Codex_2p2nqsd0c76g0"
	// ExecutableName is the packaged entry point declared in AppxManifest.xml.
	ExecutableName = "ChatGPT.exe"
	// AppSubDir is the directory holding the executable inside the package.
	AppSubDir = "app"
)

// Find returns the ChatGPT.exe path for the newest installed version.
//
// The WindowsApps directory cannot be enumerated by a normal user, so the
// package database is queried instead. An explicit path always wins.
func Find(explicit string) (string, error) {
	if strings.TrimSpace(explicit) != "" {
		if !looksPresent(explicit) {
			return "", fmt.Errorf("指定的 App 路径不存在：%s", explicit)
		}
		return explicit, nil
	}

	if names, err := win32.PackageFullNames(PackageFamilyName); err == nil && len(names) > 0 {
		for _, fullName := range SortByVersionDesc(names) {
			root, pathErr := win32.PackagePath(fullName)
			if pathErr != nil {
				continue
			}
			exe := filepath.Join(root, AppSubDir, ExecutableName)
			if looksPresent(exe) {
				return exe, nil
			}
		}
	}

	// Fallback for setups where the package database lookup is unavailable.
	pattern := filepath.Join(os.Getenv("ProgramFiles"), "WindowsApps",
		"OpenAI.Codex_*", AppSubDir, ExecutableName)
	if matches, _ := filepath.Glob(pattern); len(matches) > 0 {
		return SortPathsByVersionDesc(matches)[0], nil
	}

	return "", errors.New("未找到 Microsoft Store 版 ChatGPT/Codex")
}

// AUMID builds the Application User Model ID used for packaged activation.
func AUMID(exePath string) (string, error) {
	fullName := PackageFullNameFromPath(exePath)
	if fullName == "" {
		return "", fmt.Errorf("路径不在 WindowsApps 下，无法推断包标识：%s", exePath)
	}

	family, err := win32.PackageFamilyFromFullName(fullName)
	if err != nil || family == "" {
		// Deriving the family from the full name is best-effort; the known
		// identity is a safe fallback because the app id is always "App".
		family = PackageFamilyName
	}
	return family + "!App", nil
}

// PackageFullNameFromPath extracts "Name_Version_Arch__Publisher" from a path.
func PackageFullNameFromPath(path string) string {
	lower := strings.ToLower(path)
	const marker = `\windowsapps\`
	index := strings.Index(lower, marker)
	if index < 0 {
		return ""
	}
	rest := path[index+len(marker):]
	if end := strings.IndexByte(rest, '\\'); end >= 0 {
		return rest[:end]
	}
	return rest
}

// parseVersion extracts the numeric version from a package full name,
// e.g. "OpenAI.Codex_26.1013.4894.0_x64__2p2nqsd0c76g0" -> [26 1013 4894 0].
func parseVersion(fullName string) []int {
	segments := strings.Split(fullName, "_")
	if len(segments) < 2 {
		return nil
	}
	parts := strings.Split(segments[1], ".")
	version := make([]int, 0, len(parts))
	for _, part := range parts {
		number, err := strconv.Atoi(part)
		if err != nil {
			return nil
		}
		version = append(version, number)
	}
	return version
}

// compareVersion compares two dotted version slices numerically.
func compareVersion(a, b []int) int {
	length := len(a)
	if len(b) > length {
		length = len(b)
	}
	for i := 0; i < length; i++ {
		var left, right int
		if i < len(a) {
			left = a[i]
		}
		if i < len(b) {
			right = b[i]
		}
		if left != right {
			if left < right {
				return -1
			}
			return 1
		}
	}
	return 0
}

// SortByVersionDesc orders package full names from newest to oldest.
func SortByVersionDesc(names []string) []string {
	sorted := append([]string(nil), names...)
	sort.SliceStable(sorted, func(i, j int) bool {
		return compareVersion(parseVersion(sorted[i]), parseVersion(sorted[j])) > 0
	})
	return sorted
}

// SortPathsByVersionDesc orders executable paths from newest to oldest.
func SortPathsByVersionDesc(paths []string) []string {
	sorted := append([]string(nil), paths...)
	sort.SliceStable(sorted, func(i, j int) bool {
		left := parseVersion(PackageFullNameFromPath(sorted[i]))
		right := parseVersion(PackageFullNameFromPath(sorted[j]))
		return compareVersion(left, right) > 0
	})
	return sorted
}

// looksPresent reports whether path exists, treating permission errors as
// "present" because package directories are readable but not always listable.
func looksPresent(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return !errors.Is(err, os.ErrNotExist)
	}
	return !info.IsDir()
}

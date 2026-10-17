package app

import "testing"

func TestParseVersion(t *testing.T) {
	got := parseVersion("OpenAI.Codex_26.1013.4894.0_x64__2p2nqsd0c76g0")
	want := []int{26, 1013, 4894, 0}
	if len(got) != len(want) {
		t.Fatalf("parseVersion = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("parseVersion = %v, want %v", got, want)
		}
	}
}

func TestParseVersionRejectsGarbage(t *testing.T) {
	if got := parseVersion("no-underscore"); got != nil {
		t.Fatalf("parseVersion = %v, want nil", got)
	}
}

func TestCompareVersionIsNumeric(t *testing.T) {
	// The C++ implementation compared these as strings and picked 1.9 over 1.10.
	if compareVersion([]int{1, 10}, []int{1, 9}) <= 0 {
		t.Fatal("1.10 应当大于 1.9")
	}
	if compareVersion([]int{26, 1013, 4894, 0}, []int{26, 1013, 4894, 0}) != 0 {
		t.Fatal("相同版本应返回 0")
	}
	if compareVersion([]int{26, 1013}, []int{26, 1013, 1}) >= 0 {
		t.Fatal("缺少的段应视为 0，26.1013 < 26.1013.1")
	}
}

func TestSortByVersionDesc(t *testing.T) {
	input := []string{
		"OpenAI.Codex_1.9.0.0_x64__2p2nqsd0c76g0",
		"OpenAI.Codex_1.10.0.0_x64__2p2nqsd0c76g0",
		"OpenAI.Codex_26.1013.4894.0_x64__2p2nqsd0c76g0",
	}
	got := SortByVersionDesc(input)
	if got[0] != input[2] {
		t.Fatalf("最新版本应排首位，实际 %v", got)
	}
	if got[1] != input[1] {
		t.Fatalf("1.10 应排在 1.9 之前，实际 %v", got)
	}
}

func TestPackageFullNameFromPath(t *testing.T) {
	path := `C:\Program Files\WindowsApps\OpenAI.Codex_26.1013.4894.0_x64__2p2nqsd0c76g0\app\ChatGPT.exe`
	got := PackageFullNameFromPath(path)
	want := "OpenAI.Codex_26.1013.4894.0_x64__2p2nqsd0c76g0"
	if got != want {
		t.Fatalf("PackageFullNameFromPath = %q, want %q", got, want)
	}
	if got := PackageFullNameFromPath(`C:\Temp\ChatGPT.exe`); got != "" {
		t.Fatalf("非打包路径应返回空，实际 %q", got)
	}
}

func TestSortPathsByVersionDesc(t *testing.T) {
	paths := []string{
		`C:\Program Files\WindowsApps\OpenAI.Codex_1.9.0.0_x64__2p2nqsd0c76g0\app\ChatGPT.exe`,
		`C:\Program Files\WindowsApps\OpenAI.Codex_1.10.0.0_x64__2p2nqsd0c76g0\app\ChatGPT.exe`,
	}
	got := SortPathsByVersionDesc(paths)
	if got[0] != paths[1] {
		t.Fatalf("1.10 应排在 1.9 之前，实际 %v", got)
	}
}

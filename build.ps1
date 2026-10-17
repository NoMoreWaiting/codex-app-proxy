param(
  [string]$Version = "",
  [switch]$Clean,
  [switch]$SkipTests
)

$ErrorActionPreference = "Stop"
$root = Split-Path -Parent $MyInvocation.MyCommand.Path
$output = Join-Path $root "out"

Push-Location $root
try {
  if (-not $Version) {
    $tag = ""
    try { $tag = (git describe --tags --abbrev=0 2>$null) } catch { $tag = "" }
    if ($tag) { $Version = $tag.TrimStart("v") }
  }
  # Without a tag and without an explicit -Version, keep main.go's default
  # (this script stays ASCII-only so Windows PowerShell 5.1 parses it safely).
  $ldflags = "-s -w"
  if ($Version) { $ldflags = "$ldflags -X main.version=$Version" }

  if ($Clean -and (Test-Path -LiteralPath $output)) {
    Remove-Item -LiteralPath $output -Recurse -Force
  }

  if (-not $SkipTests) {
    Write-Host "go vet ./..."
    go vet ./...
    if ($LASTEXITCODE -ne 0) { throw "go vet failed" }

    Write-Host "go test ./..."
    go test ./... -count=1
    if ($LASTEXITCODE -ne 0) { throw "go test failed" }

    node (Join-Path $root "tests\verify-start-here.mjs") `
      (Join-Path $root "resources\START-HERE.html") `
      (Join-Path $root "resources\default_config.json")
    if ($LASTEXITCODE -ne 0) { throw "START-HERE verification failed" }

    # The exe icon is embedded straight from this ICO, so a broken source file
    # would only surface as an ugly icon in the shipped binary.
    Write-Host "verify-icon"
    & (Join-Path $root "tests\verify-icon.ps1") -Path (Join-Path $root "resources\windows\codex-proxy.ico")
  }

  New-Item -ItemType Directory -Force -Path $output | Out-Null

  # The exe icon and manifest come from a COFF resource object in the package
  # directory, which the Go linker links automatically. Regenerate it only when
  # rsrc is installed so the committed copy keeps offline builds working.
  $syso = Join-Path $root "rsrc_windows_amd64.syso"
  $rsrc = Get-Command rsrc -ErrorAction SilentlyContinue
  if ($rsrc) {
    Write-Host "rsrc: regenerating $(Split-Path -Leaf $syso)"
    & $rsrc.Source -arch amd64 `
      -ico (Join-Path $root "resources\windows\codex-proxy.ico") `
      -manifest (Join-Path $root "resources\windows\launcher.manifest") `
      -o $syso
    if ($LASTEXITCODE -ne 0) { throw "rsrc failed" }
  }
  elseif (-not (Test-Path -LiteralPath $syso)) {
    throw "rsrc_windows_amd64.syso is missing and rsrc is not installed, so the exe would ship without an icon. Install it with: go install github.com/akavel/rsrc@latest"
  }

  $label = if ($Version) { $Version } else { "main.go default" }
  Write-Host "go build (version $label)"
  go build -trimpath -ldflags $ldflags -o (Join-Path $output "CodexProxyLauncher.exe") .
  if ($LASTEXITCODE -ne 0) { throw "go build failed" }
  & (Join-Path $output "CodexProxyLauncher.exe") --version

  Copy-Item -Force (Join-Path $root "resources\START-HERE.html") $output
  Copy-Item -Force (Join-Path $root "resources\default_config.json") (Join-Path $output "config.json")
  Copy-Item -Force (Join-Path $root "README.md") $output

  . (Join-Path $root "tests\verify-package.ps1") -Path $output

  Write-Host "Output written to $output"
}
finally {
  Pop-Location
}

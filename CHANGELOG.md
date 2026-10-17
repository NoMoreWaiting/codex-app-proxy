# Changelog

本项目的用户可见变更记录在此文件中。版本遵循语义化版本号。

> **版本线说明**：本仓库自 **v1.2.0** 起独立维护与发布。v1.1.5 及更早版本的发布包、标签与提交历史仍保留在[原项目 `leisair/codex-app-proxy`](https://github.com/leisair/codex-app-proxy/releases)，本仓库不再重新发布这些历史版本，也不对其安全修复负责。

## [1.2.0] - 2026-10-17

### Added

- EXE 恢复应用图标：`rsrc_windows_amd64.syso` 嵌入 9 个尺寸（16–256）的 ICO，与 v1.1.5 发布版的图标逐字节相同；同时嵌入 manifest（`asInvoker`、PerMonitorV2 DPI、Windows 10/11 支持声明），高 DPI 下错误弹窗不再模糊。
- `build.ps1` 在检测到 `rsrc` 时自动重新生成资源对象；未安装时沿用仓库内已提交的 `.syso`，离线也能构建。
- 把此前一直没被调用的 `tests/verify-icon.ps1` 接回构建流程与 CI：校验 `resources/windows/codex-proxy.ico` 的 9 个尺寸与透明边角。该脚本自 v1.1.4 修复透明角问题后就不再被引用，源图标损坏会直接变成发布版图标异常。
- `START-HERE.html` 新增第 4 节「为 WSL 生成 .env」：按当前填写的代理一键生成 `%USERPROFILE%\.codex\.env`（保存 / 下载 / 复制三种方式，带实时预览）。页面明确要求 WSL 使用镜像网络（`networkingMode=mirrored`），并给出 NAT 模式下改用 Windows 主机地址的做法与 `wslinfo --networking-mode` 自检命令。
- 向导校验脚本改为真正执行页面脚本（DOM stub），断言生成的 `.env` 与 config 预览内容，而不再只检查标记字符串。

### Changed

- **启动器用 Go 重写**（Go 1.24+，零第三方依赖，仅标准库 + Win32 syscall），产物仍是单文件、免 DLL 的 x64 exe。移除 C++/CMake 实现与手写 JSON 正则解析器。
- 配置解析改用 `encoding/json`：忽略未知字段、缺失字段补默认值，IPv6 地址自动加方括号。
- App 定位改用 `GetPackagesByPackageFamily` + `GetPackagePathByFullName` 查询包数据库，替代普通用户无权限枚举的 `WindowsApps` 目录；版本按数值比较（修复 `1.10` 被排在 `1.9` 之后）。
- 日志改为 UTF-8 明文，单文件超过 2 MB 自动轮转为 `.1`。
- 新增命令行参数：`--app`、`--config`、`--no-terminate`、`--quiet`、`--version`、`--help`。
- README 更正 WSL 相关说明：启动器不向 WSL 写入 `~/.codex/.env`，`runCodexInWindowsSubsystemForLinux` 只影响命令执行环境，Codex 运行时仍是 Windows 原生进程并读取 Windows 侧 `.env`。
- 仓库迁移：徽章、下载、Issue 与 Security 链接统一指向 `NoMoreWaiting/codex-app-proxy`；v1.1.5 及更早版本仍归档在原项目。

### Fixed

- **修复新版 App 卡在「无法加载组织设置 / Organization settings could not be loaded」的根因**：`bypass_list` 里的 `<-loopback>` 会取消 Chromium 对回环地址的默认绕过，把 App 自身的本地 exec-server 通信一并送进代理。默认配置已移除该项，参数构建与配置加载两处都会强制过滤，并把清理后的配置写回。
- 修复后台常驻实例导致代理参数失效：App 现为 Chromium（owl）内核且常驻后台，带参激活只会唤醒旧实例并丢弃参数。启动器现在先结束所有实例（`chatgpt.exe`、`codex.exe` 等）并确认退出，再冷启动。
- 修复 `CreateProcessW` 回退的假成功：启动 1.5 秒后复查进程是否存活，已退出则返回 `E_LAUNCH_FAILED`。
- 修复 COM 激活偶发 `CO_E_NOTINITIALIZED (0x800401F0)`：在 `CoInitializeEx` 前后用 `runtime.LockOSThread` 固定 OS 线程。
- 代理预检改为真实握手（HTTP `CONNECT` / SOCKS5 首包），新增 `E_PROXY_NOT_A_PROXY`、`E_PROXY_AUTH_REQUIRED`、`E_PROXY_TIMEOUT` 三个错误码。

### Removed

- 移除开发分支中基于「WSL 内 Codex 需要代理」假设的 `~/.codex/.env` 自动注入。实测推翻该假设：WSL 侧无需注入代理，卡点是 `<-loopback>`。
- 移除 C++ 测试（`ctest`），改由 `go test ./...` 覆盖配置解析、参数构建与代理探测。

## [1.1.4] - 2026-07-14

### Fixed

- 修复启动器 ICO 生成时透明背景被铺成白色的问题。
- 快捷方式四角现在保持透明，白色圆角底板可正确显示。

### Changed

- CI 增加 ICO 尺寸与透明角校验。
- 发布步骤支持重复运行时复用已有 Release，避免重复发布造成失败。

## [1.1.3] - 2026-07-14

### Changed

- 使用新的 Codex logo 替换启动器图标源文件。
- 重新生成多尺寸 Windows ICO，并嵌入启动器 EXE。

## [1.1.2] - 2026-07-12

### Changed

- 产品名称统一为 ChatGPT/Codex App Proxy Launcher，仓库 slug 保持 `codex-app-proxy`。
- README 增加中英文搜索描述，明确 Microsoft Store、Windows x64、HTTP/SOCKS5 和 per-app proxy 关键词。

### Added

- 新增 `CONTRIBUTING.md`、`SECURITY.md` 和 GitHub Issue 模板。
- 保留并核实仓库 About 描述与现有 Topics 配置。

## [1.1.1] - 2026-07-11

### Changed

- 修正 README 对发布包文件和运行时 `logs/` 目录的描述。
- 更新 START-HERE 配置向导截图。
- Release 文案改为按版本标签独立存放，避免后续发布误用旧版本说明。
- 标签发布缺少对应说明文件时，GitHub Actions 会明确失败。

## [1.1.0] - 2026-07-11

### Added

- 新增 `errors`、`always`、`off` 三档诊断日志模式。
- START-HERE 高级设置可直接选择日志模式并保存到 `config.json`。
- Codex 旧版图标以多尺寸 Windows ICO 嵌入启动器。

### Changed

- 默认日志模式为 `errors`：正常启动不创建日志目录，失败时回放本次启动上下文。
- 旧配置缺少 `log_mode` 时自动采用默认值，无需手动迁移。
- 同步完善错误提示、隐私说明、README、离线向导与测试。

## [1.0.0] - 2026-07-11

### Added

- 首个稳定版本。
- 便携式 Windows x64 启动器，通过 Chromium 启动参数为 Microsoft Store 版 ChatGPT/Codex 设置单应用代理。
- 单文件离线配置向导、常用代理预设、环境检查、错误提示和自动化发布流程。

[1.2.0]: https://github.com/NoMoreWaiting/codex-app-proxy/releases/tag/v1.2.0
[1.1.4]: https://github.com/leisair/codex-app-proxy/compare/v1.1.3...v1.1.4
[1.1.3]: https://github.com/leisair/codex-app-proxy/compare/v1.1.2...v1.1.3
[1.1.2]: https://github.com/leisair/codex-app-proxy/compare/v1.1.1...v1.1.2
[1.1.1]: https://github.com/leisair/codex-app-proxy/compare/v1.1.0...v1.1.1
[1.1.0]: https://github.com/leisair/codex-app-proxy/compare/v1.0.0...v1.1.0
[1.0.0]: https://github.com/leisair/codex-app-proxy/releases/tag/v1.0.0

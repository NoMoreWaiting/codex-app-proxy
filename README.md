# ChatGPT/Codex App Proxy Launcher

[![Windows Build](https://github.com/NoMoreWaiting/codex-app-proxy/actions/workflows/windows-build.yml/badge.svg)](https://github.com/NoMoreWaiting/codex-app-proxy/actions/workflows/windows-build.yml)
[![Latest Release](https://img.shields.io/github/v/release/NoMoreWaiting/codex-app-proxy?label=latest)](https://github.com/NoMoreWaiting/codex-app-proxy/releases/latest)
[![Windows x64](https://img.shields.io/badge/platform-Windows%20x64-0078D4)](#适用范围)

让 Microsoft Store 版 ChatGPT/Codex **只使用你指定的本地代理**，无需开启 Windows 系统代理、TUN 模式或额外后台服务。

*A portable Windows launcher for the Microsoft Store ChatGPT/Codex desktop app. It applies a per-app HTTP/SOCKS5 proxy without changing the Windows system proxy, enabling TUN, or running a background service.*

> [下载最新版](https://github.com/NoMoreWaiting/codex-app-proxy/releases/latest) · [版本记录](CHANGELOG.md) · 下载 `windows-x64.zip`，解压后打开 `START-HERE.html`

![START-HERE 离线配置向导](https://raw.githubusercontent.com/NoMoreWaiting/codex-app-proxy/main/docs/images/start-here.png)

## 适用范围

| 你的情况 | 是否适合 |
| --- | --- |
| Windows 10/11 x64，使用 Microsoft Store 版 ChatGPT/Codex | 是 |
| 已经有 V2RayN、Clash Verge、Mihomo 或其他本地代理 | 是 |
| 不希望开启 Windows 系统代理或 TUN | 是 |
| 希望本项目提供代理节点、订阅或网络服务 | 否，本项目只是启动器 |
| macOS、Linux、网页版 ChatGPT、其他任意程序 | 暂不支持 |

本项目不会启动或配置你的代理软件，也不提供节点。开始前请确保本地代理本身可以正常使用。

## 三步开始

1. 从 [Releases](https://github.com/NoMoreWaiting/codex-app-proxy/releases/latest) 下载最新版 ZIP，并解压一次。
2. 打开 `START-HERE.html`。V2RayN 默认 mixed `10808` 用户无需修改；其他用户选择预设并保存。
3. 启动代理软件，完全退出已有 ChatGPT/Codex，然后双击 `CodexProxyLauncher.exe`。

不需要开启系统代理，不需要开启 TUN。

## English summary

This project is a Windows x64 **ChatGPT/Codex desktop app proxy launcher** for the Microsoft Store app. It starts the app with Chromium per-app proxy flags, supports local HTTP/mixed and SOCKS5 listeners, and leaves Windows system proxy settings unchanged. It is a portable utility, not a VPN, proxy provider, subscription manager, or background service.

## 它实际做什么

```text
同目录 config.json
        ↓
检查配置、代理端口（真实握手）、已有 App 进程
        ↓
找到 Microsoft Store 包里的 ChatGPT.exe
        ↓
结束后台常驻实例（保证是冷启动）
        ↓
添加 --proxy-server / --proxy-bypass-list / --disable-quic
        ↓
以包身份（AUMID）激活 ChatGPT/Codex，然后启动器立即退出
```

它不会：

- 注入 DLL 或 Hook 进程。
- 修改 Windows 系统代理。
- 创建 TUN 网卡或改写系统路由。
- 读取节点、订阅、账号或对话内容。
- 上传配置、代理流量或诊断日志。
- 常驻托盘、后台服务或额外代理进程。

除自身目录下的 `config.json` 与日志文件外，它不写入任何位置。

## 常见代理预设

`START-HERE.html` 提供以下起始值。端口可以被用户修改，请以代理软件当前显示的本地监听端口为准。

| 软件/类型 | 连接方式 | 常见本地地址 | 备注 |
| --- | --- | --- | --- |
| V2RayN | HTTP / mixed | `127.0.0.1:10808` | 项目出厂配置 |
| Clash Verge Rev | HTTP / mixed | `127.0.0.1:7897` | 以软件界面为准 |
| Clash / Mihomo | HTTP / mixed | `127.0.0.1:7890` | 不同配置可能不同 |
| 通用 SOCKS5 | SOCKS5 | `127.0.0.1:1080` | 只在使用 SOCKS5 入口时选择 |

“本地端口”不是节点服务器端口、订阅端口或远程 `443`。

## 发布包里的文件

ZIP 根目录直接包含以下文件，没有第二层 ZIP：

| 文件 | 用途 | 普通用户是否需要打开 |
| --- | --- | --- |
| `START-HERE.html` | 离线使用说明、配置预设、字段解释和排错 | 首次使用建议打开 |
| `CodexProxyLauncher.exe` | 检查环境并启动 ChatGPT/Codex | 每次通过代理启动时双击 |
| `config.json` | 启动器实际读取的配置 | 通常通过向导生成，不必手改 |
| `README.md` | 本页面的离线副本 | 按需查看 |

运行后会在同目录生成 `proxy-YYYY-MM-DD.log`：默认仅在启动出错时写入，具体由 `log_mode` 控制（`errors` / `always` / `off`）。配置和日志都保存在这个便携文件夹内；项目不会创建 `%USERPROFILE%\.codex-proxy`。

## 配置项说明

| 配置项 | 出厂值 | 作用 | 建议 |
| --- | --- | --- | --- |
| `proxy.type` | `http` | ChatGPT/Codex 连接本地代理的方式 | mixed 端口通常保持 `http` |
| `proxy.host` | `127.0.0.1` | 本地代理所在地址；该值表示当前电脑 | 普通用户不要修改 |
| `proxy.port` | `10808` | 代理软件的本地/mixed 监听端口 | 必须与代理软件界面一致 |
| `disable_quic` | `true` | 添加 `--disable-quic`，让 App 优先使用 TCP | 建议保持开启 |
| `log_mode` | `errors` | 控制诊断日志：`errors` / `always` / `off` | 默认仅错误时记录，可在向导高级设置中修改 |
| `bypass_list` | 回环地址与常见私网 | 让本地服务和内网地址不经过代理 | 不熟悉 Chromium 语法时保持默认 |

向导会始终显示当前值、出厂值和恢复入口。高级用户可以展开“专家工具”检查完整 JSON 与等价 PowerShell 命令。

## 常见问题

### 为什么新版会卡在「无法加载组织设置」？

从 2026 年 9 月起，ChatGPT/Codex 桌面版启动时会先从服务端拉取组织/工作区设置，拉不到就一直显示「无法加载组织设置 / Organization settings could not be loaded」，并且只能点重试、退出登录或退出。此时网络请求还没进入主界面。

这个请求由 Chromium 网络栈发出，所以只要启动参数正确就应该能加载。实测最常踩的坑不是代理本身不可用，而是 `bypass_list` 里带了 `<-loopback>`：

```
--proxy-server=http://127.0.0.1:10808 --disable-quic            → 正常进主界面
--proxy-server=... --proxy-bypass-list="<-loopback>;localhost"  → 卡在「无法加载组织设置」
```

`<-loopback>` 会取消 Chromium 对回环地址的隐式绕过，把 App 与本地 exec-server / 扩展宿主之间的通信也塞进代理，于是工作区设置永远拉不完。启动器现在会强制剔除该项，并把清理后的配置写回。

如果 App 启用了「在 WSL 中运行 Codex」（`%USERPROFILE%\.codex\config.toml` 中 `runCodexInWindowsSubsystemForLinux = true`），可以用向导第 4 节「为 WSL 生成 .env」按当前填写的代理一键生成 `%USERPROFILE%\.codex\.env`（保存 / 下载 / 复制三种方式）。Codex 通过 `WSLENV` 把 `CODEX_HOME` 带进 WSL，WSL 内的进程会读到同一份配置。

这份文件生效的前提是 **WSL 使用镜像网络**：在 `%USERPROFILE%\.wslconfig` 中写 `[wsl2]` + `networkingMode=mirrored`，执行 `wsl --shutdown` 后生效（需要 Windows 11 22H2+ 与 WSL 2.0+）。默认的 NAT 模式下 `127.0.0.1` 指向 WSL 自己而不是 Windows，文件写得再对也连不上——此时需要把地址换成 Windows 主机地址（WSL 内 `ip route show default | awk '{print $3}'`），且该地址可能随重启变化。用 `wsl -e wslinfo --networking-mode` 可以确认当前模式。

启动器本身**不会**向 WSL 写入任何文件，`.env` 只由向导在用户点击时生成。如果你在 WSL 里单独安装并运行 Codex CLI（和桌面 App 无关），那需要你自行在 WSL 内配置代理。

### 启动器提示「配置的端口不是代理端口」？

这是新版增加的握手校验：启动器会对配置的端口真的发一次 `CONNECT` 或 SOCKS5 握手，而不只是看端口是否打开。

典型场景是把 Clash 的**外部控制端口**（外部控制器 / API 端口）或订阅端口当成了代理端口——那个端口确实能连上，但不对代理协议应答，于是 App 启动后所有请求都会失败。请改成代理软件界面里显示的 mixed / HTTP / SOCKS5 监听端口。

### 为什么启动器会自动结束后台进程？

新版会注册开机自启，并在关闭窗口后继续在后台常驻。代理启动参数只在 App 冷启动时读取一次，所以启动器会先结束同目录下的所有 ChatGPT/Codex 实例再启动；若结束失败，会提示你在任务管理器和「设置 → 应用 → 启动」里处理。

### 代理参数为什么只在冷启动时生效？

Chromium 启动参数通常只在第一个 App 进程启动时读取。如果桌面 App 已在后台运行，再次启动可能只会唤醒旧进程，新的代理参数不会生效。

终端中独立运行的 Codex CLI 不在桌面 App 目录内，不会被本项目关闭或代理。

### “无法连接本地代理”是什么意思？

启动器无法连接 `config.json` 中的地址和端口。依次检查：

1. 代理软件是否正在运行且节点可用。
2. 配置中的端口是否等于软件界面的本地/mixed 端口。
3. 新的 `config.json` 是否与启动器位于同一目录。

### 为什么默认禁用 QUIC？

QUIC 使用 UDP，而常见本地 HTTP/mixed/SOCKS5 代理主要承载 TCP。禁用 QUIC 后，App 更稳定地使用可由 `--proxy-server` 接管的 TCP 连接，减少 UDP 绕过代理或连接失败的可能。

### Windows 提示“已保护你的电脑”怎么办？

当前发布的 EXE 尚未购买代码签名证书，Windows SmartScreen 可能提示未知发布者。这不代表文件一定有问题，也不构成安全保证。请只从本仓库的 [Releases](https://github.com/NoMoreWaiting/codex-app-proxy/releases) 下载，并核对 Release 提供的 SHA-256；对二进制不放心时，请审查源码并自行构建。

### 在哪里看日志？

```text
logs\proxy-YYYYMMDD.log
```

出厂模式 `errors` 下，正常启动不会创建 `logs` 目录；若启动失败，启动器会把此前缓存在内存中的本次启动上下文和错误一起写入上述文件。`always` 从启动开始持续记录，适合反复排错；`off` 完全不写日志，错误弹窗会显示“日志已关闭”。旧配置缺少 `log_mode` 时按 `errors` 处理。

日志可能包含配置路径、目标 App 路径、代理地址和错误码，不记录账号、对话内容或代理流量。分享前仍应自行检查其中是否含不希望公开的信息。

<details>
<summary><strong>高级用户：默认配置与手动等价命令</strong></summary>

默认配置位于启动器同目录：

```json
{
  "_comment": "ChatGPT/Codex Proxy Launcher configuration",
  "_version": 1,
  "proxy": {
    "type": "http",
    "host": "127.0.0.1",
    "port": 10808
  },
  "bypass_list": ["localhost", "127.0.0.1", "::1", "10.*", "172.16.*", "192.168.*"],
  "disable_quic": true,
  "log_mode": "errors"
}
```

`bypass_list` 里**不要写 `<-loopback>`**。它会让 Chromium 取消对回环地址的默认绕过，把 App 自己的本地 exec-server 通信也送进代理，结果是新版 App 卡在「无法加载组织设置」。启动器会主动忽略这一项，即使旧配置里仍然写着。

默认配置的等价命令：

```powershell
& (Join-Path (Get-AppxPackage -Name OpenAI.Codex).InstallLocation "app\ChatGPT.exe") --proxy-server="http://127.0.0.1:10808" --proxy-bypass-list="localhost;127.0.0.1;::1;10.*;172.16.*;172.17.*;172.18.*;172.19.*;172.20.*;172.21.*;172.22.*;172.23.*;172.24.*;172.25.*;172.26.*;172.27.*;172.28.*;172.29.*;172.30.*;172.31.*;192.168.*" --disable-quic
```

</details>

<details>
<summary><strong>开发者：本地构建与测试</strong></summary>

项目使用 **Go 1.24+、零第三方依赖**（只用标准库 + Win32 syscall），产物是单文件、免 DLL 的 exe。需要 Windows x64 与 Go 工具链；运行完整向导校验还需要 Node.js 18+。应用图标源文件位于 `assets/branding/codex-logo.svg`，多尺寸 ICO 位于 `resources/windows/codex-proxy.ico`。

exe 的图标与 manifest 由仓库根目录的 `rsrc_windows_amd64.syso` 提供——Go 会自动把同目录的 `.syso` 链接进去，所以直接 `go build` 就有图标，不需要额外的资源编译器。改动图标或 manifest 后需要重新生成：

```powershell
go install github.com/akavel/rsrc@latest
rsrc -arch amd64 `
  -ico resources\windows\codex-proxy.ico `
  -manifest resources\windows\launcher.manifest `
  -o rsrc_windows_amd64.syso
```

`build.ps1` 在检测到 `rsrc` 时会自动重跑这一步；没装则沿用已提交的 `.syso`，离线也能构建。

这个 `.syso` **必须留在仓库根目录**：Go 只把 `.syso` 链接进它所在的那一个包。移到子目录后，若那里没有 Go 文件，`go build` 会成功但产物**不含任何资源**（exe 静默丢图标和 manifest）；若做成子包再用空导入引回来，则能生效但多一层无谓的间接。

```powershell
go vet ./...
go test ./... -count=1
go build -trimpath -ldflags "-s -w -X main.version=1.2.0" -o CodexProxyLauncher.exe .
```

或者一键跑完整流程（检查 + 测试 + 打包到 `out\`）：

```powershell
.\build.ps1
```

普通用户请下载 Release，不需要安装编译环境，也不需要进入 Actions 页面。

</details>

## 版本线与历史版本

本仓库自 **v1.2.0**（Go 重写）起独立维护与发布，下载与反馈都以[本仓库 Releases](https://github.com/NoMoreWaiting/codex-app-proxy/releases) 为准。

v1.1.5 及更早版本属于上游项目，其安装包、标签与提交历史请见 [leisair/codex-app-proxy](https://github.com/leisair/codex-app-proxy/releases)。本仓库不重新发布这些历史版本，也不对其做安全修复或兼容性维护；还在使用 v1.1.x 的用户请直接升级到 v1.2.0（旧 `config.json` 仍可读取）。

## 支持与反馈

发现可复现问题请使用 [Issue 模板](https://github.com/NoMoreWaiting/codex-app-proxy/issues/new/choose)；安全漏洞请阅读 [SECURITY.md](SECURITY.md)，不要公开粘贴敏感信息。开发者可参考 [CONTRIBUTING.md](CONTRIBUTING.md)。

提交 Issue 时请附上：

- Windows 版本。
- ChatGPT/Codex App 版本。
- 代理软件名称和本地监听端口。
- 最新日志中与错误相关的几行；先检查其中是否含不希望公开的信息。

项目只支持 Windows x64 与当前 Microsoft Store 版 ChatGPT/Codex。

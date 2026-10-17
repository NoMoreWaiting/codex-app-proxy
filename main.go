// Command codex-app-proxy starts the Microsoft Store build of ChatGPT/Codex
// with a Chromium proxy, so the desktop app can reach OpenAI through a local
// proxy without touching the system proxy settings.
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"codex-app-proxy/internal/app"
	"codex-app-proxy/internal/config"
	"codex-app-proxy/internal/launch"
	"codex-app-proxy/internal/logx"
	"codex-app-proxy/internal/probe"
	"codex-app-proxy/internal/proc"
	"codex-app-proxy/internal/win32"
)

// version can be overridden at build time:
//
//	go build -ldflags "-X main.version=1.2.0"
var version = "1.2.0"

const (
	dialogTitle = "ChatGPT/Codex 代理启动器"

	exitConfigInvalid    = 1
	exitAppRunning       = 2
	exitAppNotFound      = 3
	exitProxyUnreachable = 4
	exitLaunchFailed     = 10
)

type options struct {
	appPath     string
	configPath  string
	quiet       bool
	noTerminate bool
	showVersion bool
	showHelp    bool
}

func main() {
	opts := parseOptions()

	if opts.showHelp {
		printUsage()
		return
	}
	if opts.showVersion {
		fmt.Printf("codex-app-proxy %s\n", version)
		return
	}

	selfPath, err := win32.ModuleFileName()
	if err != nil {
		fail("E_SELF_PATH", "无法定位启动器自身路径", err.Error(), exitConfigInvalid, opts.quiet)
		return
	}
	baseDir := filepath.Dir(selfPath)

	logx.Init(baseDir)
	logx.SetEcho(!opts.quiet)
	logx.Info("codex-app-proxy %s 启动", version)

	configPath := opts.configPath
	if configPath == "" {
		configPath = filepath.Join(baseDir, "config.json")
	}

	loaded, err := config.Load(configPath)
	if err != nil {
		fail("E_CONFIG_INVALID", "config.json 配置无效",
			"当前配置文件：\n"+configPath+"\n\n"+err.Error()+
				"\n\n请打开 START-HERE.html，恢复出厂配置或重新保存。",
			exitConfigInvalid, opts.quiet)
		return
	}

	cfg := loaded.Config
	logx.SetMode(cfg.LogMode)

	if loaded.Created {
		logx.Info("已创建默认配置：%s", configPath)
	}
	if loaded.Sanitized > 0 {
		logx.Info("已从 bypass_list 移除 %d 个 <-loopback>：它会让回环流量也走代理，导致 App 卡在「无法加载组织设置」",
			loaded.Sanitized)
		if saveErr := config.Save(configPath, cfg); saveErr != nil {
			logx.Error("回写清理后的配置失败：%v", saveErr)
		}
	}
	logx.Info("代理配置：%s", cfg.Proxy.URL())

	target, findErr := app.Find(opts.appPath)
	if findErr != nil {
		fail("E_APP_NOT_FOUND", "未找到 Microsoft Store 版 ChatGPT",
			"本工具仅支持 Windows x64 的 Microsoft Store 版 ChatGPT/Codex。\n"+
				"请先从 Microsoft Store 安装或更新 App，然后重试。\n\n"+
				"也可以用 --app 参数指定 ChatGPT.exe 的完整路径。\n\n系统信息："+findErr.Error(),
			exitAppNotFound, opts.quiet)
		return
	}
	logx.Info("App 路径：%s", target)

	if !opts.noTerminate {
		if err := clearResident(opts.quiet); err != nil {
			fail("E_APP_RUNNING", "无法结束已运行的 ChatGPT/Codex", err.Error(), exitAppRunning, opts.quiet)
			return
		}
	}

	result, detail := probe.Proxy(cfg.Proxy, 2500*time.Millisecond)
	if result != probe.OK {
		fail(result.Code(), result.Title(),
			"当前配置："+cfg.Proxy.URL()+"\n\n"+detail+"\n\n"+result.Advice()+
				"\n\n详细日志：\n"+logx.Path(),
			exitProxyUnreachable, opts.quiet)
		return
	}
	logx.Info("代理探测通过：%s", detail)

	arguments := launch.Arguments(cfg)
	logx.Info("启动参数：%s", arguments)

	if pid, activateErr := launch.Activate(target, arguments); activateErr == nil {
		logx.Info("AppX 激活成功，PID %d", pid)
		fmt.Println("ChatGPT/Codex 已带代理启动。")
		return
	} else {
		logx.Error("AppX 激活失败，回退 CreateProcessW：%v", activateErr)

		pid, startErr := launch.StartProcess(target, arguments)
		if startErr != nil {
			fail("E_LAUNCH_FAILED", "无法启动 ChatGPT/Codex",
				"Windows 未能启动桌面 App。\n\n"+
					"激活错误："+activateErr.Error()+"\n"+
					"启动错误："+startErr.Error()+"\n\n"+
					"请确认商店版 App 能正常打开，然后重试。\n\n详细日志：\n"+logx.Path(),
				exitLaunchFailed, opts.quiet)
			return
		}
		logx.Info("已通过 CreateProcessW 启动，PID %d", pid)

		// Packaged executables often die right after CreateProcessW because they
		// lack package identity. Verify instead of reporting a false success.
		time.Sleep(1500 * time.Millisecond)
		if !proc.Alive(pid) {
			fail("E_LAUNCH_FAILED", "App 启动后立即退出",
				"启动器无法以带参数方式拉起这个版本的 App：\n"+
					"AppX 激活失败："+activateErr.Error()+"\n"+
					"回退启动的进程已退出。\n\n"+
					"请先手动打开一次 ChatGPT 确认安装完好，再重试。\n\n详细日志：\n"+logx.Path(),
				exitLaunchFailed, opts.quiet)
			return
		}
		fmt.Println("ChatGPT/Codex 已带代理启动。")
	}
}

// clearResident terminates every running instance so the next start is a cold
// start. Without this, packaged activation only wakes the existing process and
// the proxy switches are silently dropped.
func clearResident(quiet bool) error {
	running, err := proc.AppProcesses()
	if err != nil {
		logx.Error("枚举进程失败，跳过冷启动清理：%v", err)
		return nil
	}
	if len(running) == 0 {
		return nil
	}

	pids := make([]uint32, 0, len(running))
	for _, p := range running {
		pids = append(pids, p.PID)
	}
	logx.Info("检测到 %d 个常驻实例，结束后冷启动", len(pids))

	if remaining := proc.TerminateAll(pids, 5000); len(remaining) > 0 {
		return fmt.Errorf("有 %d 个实例仍在运行，可能是权限不足或进程受保护。\n\n"+
			"请在任务管理器中结束 ChatGPT/Codex（包括后台进程），"+
			"并在「设置 → 应用 → 启动」中关闭它的开机自启，然后重试。", len(remaining))
	}

	// Give the single-instance lock and file handles a moment to be released.
	time.Sleep(500 * time.Millisecond)
	logx.Info("常驻实例已全部结束")
	return nil
}

func fail(code, title, detail string, exitCode int, quiet bool) {
	logx.Error("%s %s | %s", code, title, detail)
	if quiet {
		os.Exit(exitCode)
	}

	message := code + "：" + title + "\n\n" + detail
	win32.MessageBox(message, dialogTitle,
		win32.MB_OK|win32.MB_ICONERROR|win32.MB_SETFOREGROUND)
	os.Exit(exitCode)
}

func parseOptions() options {
	var opts options
	args := os.Args[1:]

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--app", "-app":
			if i+1 < len(args) {
				opts.appPath = args[i+1]
				i++
			}
		case "--config", "-config":
			if i+1 < len(args) {
				opts.configPath = args[i+1]
				i++
			}
		case "--quiet", "-q":
			opts.quiet = true
		case "--no-terminate":
			opts.noTerminate = true
		case "--version", "-v":
			opts.showVersion = true
		case "--help", "-h", "/?":
			opts.showHelp = true
		}
	}
	return opts
}

func printUsage() {
	fmt.Printf(`codex-app-proxy %s

用法：
  CodexProxyLauncher.exe [选项]

选项：
  --app <path>      指定 ChatGPT.exe 的完整路径（默认自动查找商店版）
  --config <path>   指定 config.json 路径（默认读取同目录 config.json）
  --no-terminate    不结束已在运行的实例（仅调试用）
  --quiet, -q       不弹出错误对话框，只返回退出码
  --version, -v     显示版本号
  --help, -h        显示本帮助

退出码：
  1  配置无效    2  无法结束已运行实例    3  未找到 App
  4  代理不可用  10 启动失败
`, version)
}

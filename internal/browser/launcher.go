package browser

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"
)

// BrowserProcess represents a running headless browser instance.
type BrowserProcess struct {
	cmd     *exec.Cmd
	dataDir string
	wsURL   string
	port    int
}

// LauncherConfig specifies startup options for the browser.
type LauncherConfig struct {
	ExecutablePath string
	Headless       bool
	Stealth        bool
	WindowWidth    int
	WindowHeight   int
	UserDataDir    string
	ExtraArgs      []string
}

// DefaultLauncherConfig returns standard production defaults.
func DefaultLauncherConfig() LauncherConfig {
	return LauncherConfig{
		Headless:     true,
		Stealth:      true,
		WindowWidth:  1920,
		WindowHeight: 1080,
	}
}

// FindBrowserExecutable locates Chrome, Chromium, or Edge on the host system.
func FindBrowserExecutable() (string, error) {
	if envPath := os.Getenv("CHROME_PATH"); envPath != "" {
		if _, err := os.Stat(envPath); err == nil {
			return envPath, nil
		}
	}
	if envPath := os.Getenv("DOMDRIFT_CHROME_BIN"); envPath != "" {
		if _, err := os.Stat(envPath); err == nil {
			return envPath, nil
		}
	}

	var candidates []string
	switch runtime.GOOS {
	case "windows":
		candidates = []string{
			os.Getenv("ProgramFiles") + `\Google\Chrome\Application\chrome.exe`,
			os.Getenv("ProgramFiles(x86)") + `\Google\Chrome\Application\chrome.exe`,
			os.Getenv("LocalAppData") + `\Google\Chrome\Application\chrome.exe`,
			os.Getenv("ProgramFiles(x86)") + `\Microsoft\Edge\Application\msedge.exe`,
			os.Getenv("ProgramFiles") + `\Microsoft\Edge\Application\msedge.exe`,
		}
	case "darwin":
		candidates = []string{
			"/Applications/Google Chrome.app/Contents/MacOS/Google Chrome",
			"/Applications/Chromium.app/Contents/MacOS/Chromium",
			"/Applications/Microsoft Edge.app/Contents/MacOS/Microsoft Edge",
		}
	default: // Linux / BSD
		candidates = []string{
			"/usr/bin/google-chrome-stable",
			"/usr/bin/google-chrome",
			"/usr/bin/chromium-browser",
			"/usr/bin/chromium",
			"/usr/bin/microsoft-edge-stable",
		}
	}

	for _, path := range candidates {
		if path == "" {
			continue
		}
		if _, err := os.Stat(path); err == nil {
			return path, nil
		}
	}

	// Try finding in PATH
	execNames := []string{"google-chrome-stable", "google-chrome", "chromium-browser", "chromium", "chrome", "msedge"}
	for _, name := range execNames {
		if path, err := exec.LookPath(name); err == nil {
			return path, nil
		}
	}

	return "", errors.New("no compatible Chrome, Chromium, or Edge binary found on system")
}

// Launch starts a new headless browser process.
func Launch(ctx context.Context, cfg LauncherConfig) (*BrowserProcess, error) {
	exePath := cfg.ExecutablePath
	if exePath == "" {
		var err error
		exePath, err = FindBrowserExecutable()
		if err != nil {
			return nil, err
		}
	}

	dataDir := cfg.UserDataDir
	var autoCleanupDir bool
	if dataDir == "" {
		tmp, err := os.MkdirTemp("", "domdrift_chrome_*")
		if err != nil {
			return nil, fmt.Errorf("failed to create temp chrome profile: %w", err)
		}
		dataDir = tmp
		autoCleanupDir = true
	}

	args := []string{
		"--remote-debugging-port=0",
		fmt.Sprintf("--user-data-dir=%s", dataDir),
		"--no-first-run",
		"--no-default-browser-check",
		"--disable-background-networking",
		"--disable-background-timer-throttling",
		"--disable-client-side-phishing-detection",
		"--disable-default-apps",
		"--disable-hang-monitor",
		"--disable-popup-blocking",
		"--disable-prompt-on-repost",
		"--disable-sync",
		"--disable-translate",
		"--metrics-recording-only",
		"--safebrowsing-disable-auto-update",
		"--password-store=basic",
		"--use-mock-keychain",
		"--hide-scrollbars",
		"--mute-audio",
		fmt.Sprintf("--window-size=%d,%d", cfg.WindowWidth, cfg.WindowHeight),
	}

	if cfg.Headless {
		args = append(args, "--headless=new", "--disable-gpu")
	}

	if cfg.Stealth {
		args = append(args,
			"--disable-blink-features=AutomationControlled",
			"--disable-infobars",
			"--disable-features=IsolateOrigins,site-per-process",
			"--user-agent=Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/126.0.0.0 Safari/537.36",
		)
	}

	args = append(args, cfg.ExtraArgs...)
	args = append(args, "about:blank")

	cmd := exec.CommandContext(ctx, exePath, args...)

	if err := cmd.Start(); err != nil {
		if autoCleanupDir {
			_ = os.RemoveAll(dataDir)
		}
		return nil, fmt.Errorf("failed to spawn browser process: %w", err)
	}

	proc := &BrowserProcess{
		cmd:     cmd,
		dataDir: dataDir,
	}

	// Wait for DevToolsActivePort file to appear and parse the port
	portFile := filepath.Join(dataDir, "DevToolsActivePort")
	port, path, err := waitForDevToolsActivePort(ctx, portFile, 15*time.Second)
	if err != nil {
		_ = proc.Close()
		return nil, fmt.Errorf("failed reading browser devtools port: %w", err)
	}

	proc.port = port
	proc.wsURL = fmt.Sprintf("ws://127.0.0.1:%d%s", port, path)

	return proc, nil
}

func waitForDevToolsActivePort(ctx context.Context, portFile string, timeout time.Duration) (int, string, error) {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			return 0, "", ctx.Err()
		default:
		}

		if _, err := os.Stat(portFile); err == nil {
			f, err := os.Open(portFile)
			if err == nil {
				scanner := bufio.NewScanner(f)
				var lines []string
				for scanner.Scan() {
					lines = append(lines, strings.TrimSpace(scanner.Text()))
				}
				_ = f.Close()

				if len(lines) >= 2 {
					port, err := strconv.Atoi(lines[0])
					if err == nil && port > 0 {
						path := lines[1]
						if !strings.HasPrefix(path, "/") {
							path = "/" + path
						}
						return port, path, nil
					}
				}
			}
		}
		time.Sleep(50 * time.Millisecond)
	}

	return 0, "", fmt.Errorf("timed out after %v waiting for DevToolsActivePort file", timeout)
}

// WebSocketURL returns the root DevTools browser websocket endpoint.
func (p *BrowserProcess) WebSocketURL() string {
	return p.wsURL
}

// Port returns the assigned debugging port.
func (p *BrowserProcess) Port() int {
	return p.port
}

// Close terminates the browser process and removes temporary profile files.
func (p *BrowserProcess) Close() error {
	var errs []error
	if p.cmd != nil && p.cmd.Process != nil {
		if err := p.cmd.Process.Kill(); err != nil && !errors.Is(err, os.ErrProcessDone) {
			errs = append(errs, err)
		}
		_ = p.cmd.Wait()
	}

	if p.dataDir != "" && strings.Contains(p.dataDir, "domdrift_chrome_") {
		// Attempt removal of temp profile
		_ = os.RemoveAll(p.dataDir)
	}

	if len(errs) > 0 {
		return errs[0]
	}
	return nil
}

// IsAlive checks whether the browser process is still running.
func (p *BrowserProcess) IsAlive() bool {
	if p.cmd == nil || p.cmd.Process == nil {
		return false
	}
	// Try connecting to debug port
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("127.0.0.1:%d", p.port), 200*time.Millisecond)
	if err != nil {
		return false
	}
	_ = conn.Close()
	return true
}

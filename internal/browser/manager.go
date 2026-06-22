package browser

import (
    "bytes"
    "context"
    "crypto/rand"
    "fmt"
    "log/slog"
    "math/big"
    "os"
    "os/exec"
    "path/filepath"
    "strconv"
    "strings"
    "sync"
    "syscall"

    "github.com/chromedp/chromedp"
)

var chromiumPaths = []string{
    "google-chrome",
    "google-chrome-stable",
    "chromium",
    "chromium-browser",
    "brave-browser",
    "brave",
}

type Manager struct {
    allocCtx    context.Context
    allocCancel context.CancelFunc
    browserCtx  context.Context
    managerID   string
    userDataDir string
    mu          sync.Mutex
    closed      bool
}

func findChromium() (string, error) {
    for _, path := range chromiumPaths {
        if _, err := exec.LookPath(path); err == nil {
            return path, nil
        }
    }
    return "", fmt.Errorf("no chromium-based browser found (tried: %v)", chromiumPaths)
}

func newManagerID() string {
    n, err := rand.Int(rand.Reader, big.NewInt(1<<62))
    if err != nil {
        return fmt.Sprintf("gospy-%d", os.Getpid())
    }
    return fmt.Sprintf("gospy-%x", n)
}

func killProcByDataDir(dataDir string) {
    entries, err := os.ReadDir("/proc")
    if err != nil {
        return
    }
    for _, entry := range entries {
        pid, err := strconv.Atoi(entry.Name())
        if err != nil {
            continue
        }
        cmdline, err := os.ReadFile(fmt.Sprintf("/proc/%d/cmdline", pid))
        if err != nil {
            continue
        }
        if bytes.Contains(cmdline, []byte(dataDir)) && strings.Contains(string(cmdline), "headless") {
            syscall.Kill(pid, syscall.SIGKILL)
        }
    }
}

func NewManager(ctx context.Context) (*Manager, error) {
    browserPath, err := findChromium()
    if err != nil {
        return nil, err
    }

    slog.Debug("found browser", "path", browserPath)

    managerID := newManagerID()
    userDataDir := filepath.Join(os.TempDir(), managerID)

	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.DisableGPU,
		chromedp.NoSandbox,
		chromedp.Headless,
		chromedp.ExecPath(browserPath),
		chromedp.UserDataDir(userDataDir),
		chromedp.Flag("ignore-certificate-errors", true),
	)

    allocCtx, allocCancel := chromedp.NewExecAllocator(context.Background(), opts...)
    browserCtx, _ := chromedp.NewContext(allocCtx)

    if err := chromedp.Run(browserCtx); err != nil {
        allocCancel()
        return nil, fmt.Errorf("start browser: %w", err)
    }

    slog.Debug("browser started")

    return &Manager{
        allocCtx:    allocCtx,
        allocCancel: allocCancel,
        browserCtx:  browserCtx,
        managerID:   managerID,
        userDataDir: userDataDir,
    }, nil
}

func (m *Manager) NewContext() (context.Context, context.CancelFunc) {
    m.mu.Lock()
    defer m.mu.Unlock()

    if m.closed {
        return nil, nil
    }

    tabCtx, tabCancel := chromedp.NewContext(m.browserCtx)
    return tabCtx, tabCancel
}

func (m *Manager) Close() {
    m.mu.Lock()
    defer m.mu.Unlock()

    if m.closed {
        return
    }
    m.closed = true

    m.allocCancel()

    killProcByDataDir(m.managerID)

    os.RemoveAll(m.userDataDir)

    slog.Debug("browser closed")
}

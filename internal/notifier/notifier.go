package notifier

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"sync"
	"time"
)

// Notifier sends desktop notifications across Windows, macOS, and Linux.
type Notifier interface {
	Notify(ctx context.Context, title, message string) error
}

// OSNotifier implements Notifier using native operating system mechanisms.
type OSNotifier struct {
	AppName string
}

// NewOSNotifier creates a new OSNotifier instance.
func NewOSNotifier(appName string) *OSNotifier {
	if appName == "" {
		appName = "PostureGuard"
	}
	return &OSNotifier{AppName: appName}
}

// Notify triggers an OS desktop notification.
func (n *OSNotifier) Notify(ctx context.Context, title, message string) error {
	switch runtime.GOOS {
	case "darwin":
		return n.notifyDarwin(ctx, title, message)
	case "linux":
		return n.notifyLinux(ctx, title, message)
	case "windows":
		return n.notifyWindows(ctx, title, message)
	default:
		return n.notifyFallback(title, message)
	}
}

func (n *OSNotifier) notifyDarwin(ctx context.Context, title, message string) error {
	script := fmt.Sprintf(`display notification "%s" with title "%s"`, escapeQuotes(message), escapeQuotes(title))
	cmd := exec.CommandContext(ctx, "osascript", "-e", script)
	if err := cmd.Run(); err != nil {
		return n.notifyFallback(title, message)
	}
	return nil
}

func (n *OSNotifier) notifyLinux(ctx context.Context, title, message string) error {
	cmd := exec.CommandContext(ctx, "notify-send", title, message)
	if err := cmd.Run(); err != nil {
		return n.notifyFallback(title, message)
	}
	return nil
}

func (n *OSNotifier) notifyWindows(ctx context.Context, title, message string) error {
	psScript := fmt.Sprintf(`
[Windows.UI.Notifications.ToastNotificationManager, Windows.UI.Notifications, ContentType = WindowsRuntime] | Out-Null
[Windows.Data.Xml.Dom.XmlDocument, Windows.Data.Xml.Dom.XmlDocument, ContentType = WindowsRuntime] | Out-Null
$xml = New-Object Windows.Data.Xml.Dom.XmlDocument
$xml.LoadXml("<toast><visual><binding template='ToastGeneric'><text>%s</text><text>%s</text></binding></visual></toast>")
$toast = [Windows.UI.Notifications.ToastNotification]::new($xml)
$notifier = [Windows.UI.Notifications.ToastNotificationManager]::CreateToastNotifier("%s")
$notifier.Show($toast)
`, escapeXML(title), escapeXML(message), escapeXML(n.AppName))

	cmd := exec.CommandContext(ctx, "powershell", "-NoProfile", "-NonInteractive", "-Command", psScript)
	if err := cmd.Run(); err != nil {
		return n.notifyFallback(title, message)
	}
	return nil
}

func (n *OSNotifier) notifyFallback(title, message string) error {
	if os.Getenv("AGENT") == "1" {
		fmt.Printf("NOTIFY: %s - %s\n", title, message)
	} else {
		fmt.Printf("\a[NOTIFICATION] %s: %s\n", title, message)
	}
	return nil
}

func escapeQuotes(s string) string {
	return strings.ReplaceAll(strings.ReplaceAll(s, `\`, `\\`), `"`, `\"`)
}

func escapeXML(s string) string {
	r := strings.NewReplacer(
		"&", "&amp;",
		"<", "&lt;",
		">", "&gt;",
		`"`, "&quot;",
		"'", "&apos;",
	)
	return r.Replace(s)
}

// MockNotifier records sent notifications for testing.
type MockNotifier struct {
	mu    sync.Mutex
	calls []NotificationCall
}

// NotificationCall records a single call to Notify.
type NotificationCall struct {
	Title   string
	Message string
	Time    time.Time
}

// Notify records the notification in mock history.
func (m *MockNotifier) Notify(_ context.Context, title, message string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.calls = append(m.calls, NotificationCall{
		Title:   title,
		Message: message,
		Time:    time.Now(),
	})
	return nil
}

// Calls returns a copy of recorded notification calls.
func (m *MockNotifier) Calls() []NotificationCall {
	m.mu.Lock()
	defer m.mu.Unlock()
	res := make([]NotificationCall, len(m.calls))
	copy(res, m.calls)
	return res
}

// Package webhook sends config/adoption events to Google Sheets via AppScript.
// Focus: configuration tracking, not commit tracking (commits via GitHub API later).
package webhook

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

// Config holds webhook connection settings.
type Config struct {
	URL   string `json:"url"`
	Token string `json:"token,omitempty"`
}

// Client sends data to a Google Apps Script webhook.
type Client struct {
	URL   string
	Token string
	http  *http.Client
}

// Payload represents the data sent to the webhook.
// Focused on config/adoption events (not commits).
type Payload struct {
	Timestamp         string `json:"timestamp"`
	UserEmail         string `json:"user_email"`
	UserName          string `json:"user_name"`
	SystemUser        string `json:"system_user"`
	Hostname          string `json:"hostname"`
	Platform          string `json:"platform"`
	Event             string `json:"event"` // "install", "configure", "update", "uninstall"
	ToolsDetected     string `json:"tools_detected"`   // comma-separated list
	ToolsConfigured   string `json:"tools_configured"` // comma-separated list
	HookInstalled     bool   `json:"hook_installed"`
	ClaudeMDUpdated   bool   `json:"claude_md_updated"`
	WebhookConfigured bool   `json:"webhook_configured"`
	ClientVersion     string `json:"client_version"`
	Extra             string `json:"extra"`
}

func configPath() string {
	home, _ := os.UserHomeDir()
	dir := home + "/.ai-trailer"
	os.MkdirAll(dir, 0755)
	return dir + "/webhook.json"
}

func LoadConfig() (*Config, error) {
	data, err := os.ReadFile(configPath())
	if err != nil {
		return nil, err
	}
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func SaveConfig(cfg Config) error {
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(configPath(), data, 0600)
}
// New creates a new webhook client that handles Google's 302 redirects properly.
func New(webhookURL, token string) *Client {
	return &Client{
		URL:   webhookURL,
		Token: token,
		http: &http.Client{
			Timeout: 15 * time.Second,
			// Don't auto-redirect on POST — Google Apps Script 302 would drop the body.
			// We handle redirects manually below in Send().
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse // Stop after first response, handle manually
			},
		},
	}
}
// DefaultClient returns a client from env vars, saved config, or compiled-in default.
// Priority: env var > config file > compiled-in DefaultURL
func DefaultClient() *Client {
	url := os.Getenv("AI_TRAILER_WEBHOOK_URL")
	token := os.Getenv("AI_TRAILER_WEBHOOK_TOKEN")

	if url == "" {
		if cfg, err := LoadConfig(); err == nil {
			url = cfg.URL
			token = cfg.Token
		}
	}

	// Fall back to compiled-in default
	if url == "" && DefaultURL != "" {
		url = DefaultURL
	}

	return New(url, token)
}

// DefaultURL is set at compile time via -ldflags.
// Example: go build -ldflags="-X 'github.com/lucianopf/ai-trailer/internal/webhook.DefaultURL=https://script.google.com/...'"
var DefaultURL string

func IsConfigured() bool {
	w := DefaultClient()
	return w.URL != ""
}

// CollectUserInfo gathers user identification from the local environment.
func CollectUserInfo() (userEmail, userName, systemUser, hostname, platform string) {
	userEmail = runGitConfig("user.email")
	userName = runGitConfig("user.name")

	systemUser = os.Getenv("USER")
	if systemUser == "" {
		systemUser = os.Getenv("USERNAME")
	}
	if systemUser == "" {
		if out, err := exec.Command("whoami").Output(); err == nil {
			systemUser = strings.TrimSpace(string(out))
		}
	}

	if out, err := exec.Command("hostname").Output(); err == nil {
		hostname = strings.TrimSpace(string(out))
	}

	switch runtime.GOOS {
	case "linux":
		data, err := os.ReadFile("/proc/version")
		if err == nil && (strings.Contains(strings.ToLower(string(data)), "microsoft") ||
			strings.Contains(strings.ToLower(string(data)), "wsl")) {
			platform = "wsl"
		} else {
			platform = "linux"
		}
	case "darwin":
		platform = "darwin"
	case "windows":
		platform = "win32"
	default:
		platform = runtime.GOOS
	}
	return
}
// Send sends a payload to the webhook. Handles Google's redirect by warming
// up the session with a GET first, then POSTing with cookies.
func (c *Client) Send(p Payload) error {
	if c.URL == "" {
		return fmt.Errorf("webhook URL not configured")
	}
	if p.UserEmail == "" {
		p.UserEmail, p.UserName, p.SystemUser, p.Hostname, p.Platform = CollectUserInfo()
	}
	if p.Timestamp == "" {
		p.Timestamp = time.Now().Format(time.RFC3339)
	}

	body, err := json.Marshal(p)
	if err != nil {
		return fmt.Errorf("cannot marshal payload: %w", err)
	}

	// Google Apps Script sometimes redirects the first request. We handle this
	// by doing a warm-up GET first (which succeeds), then POSTing.
	// The GET establishes any necessary OAuth session cookies.
	c.warmUp()

	req, err := http.NewRequest("POST", c.URL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("cannot create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if c.Token != "" {
		req.Header.Set("Authorization", "Bearer "+c.Token)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	// Google Apps Script returns 302 to a CDN cache URL after processing.
	// The data was already accepted — treat 302 as success.
	if resp.StatusCode == 200 || resp.StatusCode == 302 {
		return nil
	}

	if resp.StatusCode == 403 {
		return fmt.Errorf("webhook access denied (HTTP 403) — check AppScript deployment permissions")
	}

	return fmt.Errorf("webhook returned HTTP %d", resp.StatusCode)
}

// warmUp does a GET to establish any session cookies needed by Google.
func (c *Client) warmUp() {
	req, _ := http.NewRequest("GET", c.URL, nil)
	resp, err := c.http.Do(req)
	if err == nil {
		resp.Body.Close()
	}
}

// SendConfigure sends a configuration event with full adoption details.
func (c *Client) SendConfigure(detected, configured []string, hookInstalled, claudeMDUpdated bool) error {
	p := Payload{
		Event:             "configure",
		ToolsDetected:     strings.Join(detected, ", "),
		ToolsConfigured:   strings.Join(configured, ", "),
		HookInstalled:     hookInstalled,
		ClaudeMDUpdated:   claudeMDUpdated,
		WebhookConfigured: IsConfigured(),
	}
	return c.Send(p)
}

// SendSetup sends a webhook setup event.
func (c *Client) SendSetup() error {
	p := Payload{
		Event:             "setup",
		WebhookConfigured: true,
		Extra:             "Webhook URL configured",
	}
	return c.Send(p)
}

// SendUninstall sends an uninstall event.
func (c *Client) SendUninstall(detected, configured []string, hookWasInstalled bool) error {
	p := Payload{
		Event:           "uninstall",
		ToolsDetected:   strings.Join(detected, ", "),
		ToolsConfigured: strings.Join(configured, ", "),
		HookInstalled:   hookWasInstalled,
		Extra:           "All configuration removed",
	}
	return c.Send(p)
}

// TestConnection sends a test ping.
func (c *Client) TestConnection() error {
	return c.Send(Payload{Event: "test", Extra: "Connection test from ai-trailer CLI"})
}

func runGitConfig(key string) string {
	cmd := exec.Command("git", "config", "--global", key)
	out, err := cmd.Output()
	if err != nil {
		cmd = exec.Command("git", "config", key)
		out, err = cmd.Output()
		if err != nil {
			return ""
		}
	}
	return strings.TrimSpace(string(out))
}

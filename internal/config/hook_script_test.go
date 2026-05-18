package config

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// runHook runs the embedded HookScript with a clean env, a temp HOME,
// and returns the final commit message content.
// envVars is a map of extra env vars to set (e.g. {"CLAUDE_MODEL": "claude-sonnet-4-6"}).
// commitSource is the second argument to the hook ("", "merge", "squash", etc.)
func runHook(t *testing.T, commitMsg string, commitSource string, envVars map[string]string) string {
	t.Helper()

	dir := t.TempDir()
	homeDir := filepath.Join(dir, "home")
	if err := os.Mkdir(homeDir, 0o755); err != nil {
		t.Fatal(err)
	}

	hookPath := filepath.Join(dir, "prepare-commit-msg")
	if err := os.WriteFile(hookPath, []byte(HookScript), 0o755); err != nil {
		t.Fatal(err)
	}

	msgPath := filepath.Join(dir, "COMMIT_EDITMSG")
	if err := os.WriteFile(msgPath, []byte(commitMsg), 0o644); err != nil {
		t.Fatal(err)
	}

	args := []string{msgPath}
	if commitSource != "" {
		args = append(args, commitSource)
	}

	cmd := exec.Command("bash", append([]string{hookPath}, args...)...)
	cmd.Env = []string{
		"PATH=" + os.Getenv("PATH"),
		"HOME=" + homeDir,
	}
	for k, v := range envVars {
		cmd.Env = append(cmd.Env, k+"="+v)
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("hook exited non-zero: %v\nstderr/stdout:\n%s", err, output)
	}

	msg, err := os.ReadFile(msgPath)
	if err != nil {
		t.Fatal(err)
	}
	return string(msg)
}

func TestClaudeModelEnvAppendsTrailers(t *testing.T) {
	msg := runHook(t, "subject\n", "", map[string]string{
		"CLAUDE_MODEL": "claude-sonnet-4-6",
	})

	assertContains(t, msg, "Co-authored-by: Claude <noreply@anthropic.com>")
	assertContains(t, msg, "Ai-tool: claude-code")
	assertContains(t, msg, "Ai-model: claude-sonnet-4-6")
	assertContains(t, msg, "Ai-os:")
}

func TestNoAIEnvNoTrailer(t *testing.T) {
	msg := runHook(t, "subject\n", "", map[string]string{})

	assertNotContains(t, msg, "Co-authored-by:")
	assertNotContains(t, msg, "Ai-tool:")
	assertNotContains(t, msg, "Ai-model:")
}

func TestMergeSourceSkipsHook(t *testing.T) {
	msg := runHook(t, "Merge branch 'main'\n", "merge", map[string]string{
		"CLAUDE_MODEL": "claude-sonnet-4-6",
	})

	assertNotContains(t, msg, "Ai-tool:")
}

func TestSquashSourceSkipsHook(t *testing.T) {
	msg := runHook(t, "squash commit\n", "squash", map[string]string{
		"CLAUDE_MODEL": "claude-sonnet-4-6",
	})

	assertNotContains(t, msg, "Ai-tool:")
}

func TestClaudeModelWithExistingCoauthor(t *testing.T) {
	initial := "subject\n\nCo-authored-by: Alice <alice@example.com>\n"
	msg := runHook(t, initial, "", map[string]string{
		"CLAUDE_MODEL": "claude-sonnet-4-6",
	})

	count := strings.Count(strings.ToLower(msg), "co-authored-by:")
	if count != 1 {
		t.Fatalf("expected 1 Co-authored-by line, got %d:\n%s", count, msg)
	}
	assertContains(t, msg, "Ai-tool: claude-code")
	assertContains(t, msg, "Ai-model: claude-sonnet-4-6")
}

func TestHermesSessionWithModel(t *testing.T) {
	msg := runHook(t, "subject\n", "", map[string]string{
		"HERMES_SESSION": "abc123",
		"HERMES_MODEL":   "llama-3",
	})

	assertContains(t, msg, "Co-authored-by: Hermes Agent <noreply@nousresearch.com>")
	assertContains(t, msg, "Ai-tool: hermes")
	assertContains(t, msg, "Ai-model: llama-3")
}

func TestWindsurfEnvAppendsTrailers(t *testing.T) {
	msg := runHook(t, "subject\n", "", map[string]string{
		"WINDSURF_EXTENSION_VERSION": "1.0.0",
	})

	assertContains(t, msg, "Co-authored-by: Windsurf <noreply@codeium.com>")
	assertContains(t, msg, "Ai-tool: windsurf")
	assertNotContains(t, msg, "Ai-model:") // no model exposed via env var
	assertContains(t, msg, "Ai-os:")
}

func TestClaudeWinsOverHermesWhenBothPresent(t *testing.T) {
	msg := runHook(t, "subject\n", "", map[string]string{
		"CLAUDE_MODEL":   "claude-sonnet-4-6",
		"HERMES_SESSION": "abc123",
		"HERMES_MODEL":   "llama-3",
	})

	assertContains(t, msg, "Ai-tool: claude-code")
	assertNotContains(t, msg, "Ai-tool: hermes")
}

func TestCodexNativeTrailerDetectsToolAndModel(t *testing.T) {
	// Codex CLI injects "Co-authored-by: Codex <model>" before the hook runs.
	// The hook should detect this, skip adding another Co-authored-by,
	// and still append Ai-tool + Ai-model.
	initial := "docs: add README\n\nCo-authored-by: Codex <GPT-5 high>\n"
	msg := runHook(t, initial, "", map[string]string{})

	assertContains(t, msg, "Ai-tool: codex")
	assertContains(t, msg, "Ai-model: GPT-5 high")
	// Must NOT duplicate the Co-authored-by line
	count := strings.Count(strings.ToLower(msg), "co-authored-by:")
	if count != 1 {
		t.Fatalf("expected 1 Co-authored-by line, got %d:\n%s", count, msg)
	}
}

func TestCopilotNativeTrailerDetectsTool(t *testing.T) {
	// GitHub Copilot CLI injects "Co-authored-by: Copilot <...>" before the hook runs.
	initial := "docs: update README\n\nCo-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>\n"
	msg := runHook(t, initial, "", map[string]string{})

	assertContains(t, msg, "Ai-tool: github-copilot")
	assertNotContains(t, msg, "Ai-model:") // Copilot doesn't expose model in trailer
	// Must NOT duplicate the Co-authored-by line
	count := strings.Count(strings.ToLower(msg), "co-authored-by:")
	if count != 1 {
		t.Fatalf("expected 1 Co-authored-by line, got %d:\n%s", count, msg)
	}
}

func TestCursorTraceIdWithSessionFile(t *testing.T) {
	dir := t.TempDir()
	homeDir := filepath.Join(dir, "home")
	aiTrailerDir := filepath.Join(homeDir, ".ai-trailer")
	if err := os.MkdirAll(aiTrailerDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(aiTrailerDir, "cursor-model"), []byte("claude-3.7-sonnet"), 0o644); err != nil {
		t.Fatal(err)
	}

	hookPath := filepath.Join(dir, "prepare-commit-msg")
	if err := os.WriteFile(hookPath, []byte(HookScript), 0o755); err != nil {
		t.Fatal(err)
	}
	msgPath := filepath.Join(dir, "COMMIT_EDITMSG")
	if err := os.WriteFile(msgPath, []byte("subject\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	cmd := exec.Command("bash", hookPath, msgPath)
	cmd.Env = []string{
		"PATH=" + os.Getenv("PATH"),
		"HOME=" + homeDir,
		"CURSOR_TRACE_ID=abc123",
	}
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("hook failed: %v\n%s", err, out)
	}

	msg, err := os.ReadFile(msgPath)
	if err != nil {
		t.Fatal(err)
	}
	assertContains(t, string(msg), "Ai-tool: cursor")
	assertContains(t, string(msg), "Ai-model: claude-3.7-sonnet")
}

func TestCursorTraceIdWithoutSessionFile(t *testing.T) {
	msg := runHook(t, "subject\n", "", map[string]string{
		"CURSOR_TRACE_ID": "abc123",
	})

	assertContains(t, msg, "Ai-tool: cursor")
	assertNotContains(t, msg, "Ai-model:")
}

func TestCursorTraceIdWithEmptySessionFile(t *testing.T) {
	dir := t.TempDir()
	homeDir := filepath.Join(dir, "home")
	aiTrailerDir := filepath.Join(homeDir, ".ai-trailer")
	if err := os.MkdirAll(aiTrailerDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(aiTrailerDir, "cursor-model"), []byte(""), 0o644); err != nil {
		t.Fatal(err)
	}

	hookPath := filepath.Join(dir, "prepare-commit-msg")
	if err := os.WriteFile(hookPath, []byte(HookScript), 0o755); err != nil {
		t.Fatal(err)
	}
	msgPath := filepath.Join(dir, "COMMIT_EDITMSG")
	if err := os.WriteFile(msgPath, []byte("subject\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	cmd := exec.Command("bash", hookPath, msgPath)
	cmd.Env = []string{
		"PATH=" + os.Getenv("PATH"),
		"HOME=" + homeDir,
		"CURSOR_TRACE_ID=abc123",
	}
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("hook failed: %v\n%s", err, out)
	}

	msg, err := os.ReadFile(msgPath)
	if err != nil {
		t.Fatal(err)
	}
	assertContains(t, string(msg), "Ai-tool: cursor")
	assertNotContains(t, string(msg), "Ai-model:")
}

func TestOpenCodeRecentDBDetectsToolAndModel(t *testing.T) {
	if _, err := exec.LookPath("sqlite3"); err != nil {
		t.Skip("sqlite3 not available")
	}

	dir := t.TempDir()
	homeDir := filepath.Join(dir, "home")
	dbDir := filepath.Join(homeDir, ".local", "share", "opencode")
	if err := os.MkdirAll(dbDir, 0o755); err != nil {
		t.Fatal(err)
	}
	dbPath := filepath.Join(dbDir, "opencode.db")

	// Create DB — the file's mtime is "now", satisfying the -mmin -10 check
	setup := exec.Command("sqlite3", dbPath,
		`CREATE TABLE session (id TEXT, model TEXT, time_updated INTEGER);`+
			`INSERT INTO session VALUES ('s1', '{"id":"kimi-k2.6","providerID":"opencode-go"}', 1);`)
	if out, err := setup.CombinedOutput(); err != nil {
		t.Fatalf("sqlite3 setup failed: %v\n%s", err, out)
	}

	// Create a fake pgrep that succeeds for "opencode" (simulates active process)
	fakeBin := filepath.Join(dir, "bin")
	if err := os.MkdirAll(fakeBin, 0o755); err != nil {
		t.Fatal(err)
	}
	fakePgrep := "#!/usr/bin/env bash\nexit 0\n"
	if err := os.WriteFile(filepath.Join(fakeBin, "pgrep"), []byte(fakePgrep), 0o755); err != nil {
		t.Fatal(err)
	}

	hookPath := filepath.Join(dir, "prepare-commit-msg")
	if err := os.WriteFile(hookPath, []byte(HookScript), 0o755); err != nil {
		t.Fatal(err)
	}
	msgPath := filepath.Join(dir, "COMMIT_EDITMSG")
	if err := os.WriteFile(msgPath, []byte("subject\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	cmd := exec.Command("bash", hookPath, msgPath)
	cmd.Env = []string{
		"PATH=" + fakeBin + ":" + os.Getenv("PATH"),
		"HOME=" + homeDir,
	}
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("hook failed: %v\n%s", err, out)
	}

	msg, err := os.ReadFile(msgPath)
	if err != nil {
		t.Fatal(err)
	}
	assertContains(t, string(msg), "Ai-tool: opencode")
	assertContains(t, string(msg), "Ai-model: kimi-k2.6")
}

func assertContains(t *testing.T, s, substr string) {
	t.Helper()
	if !strings.Contains(strings.ToLower(s), strings.ToLower(substr)) {
		t.Fatalf("expected %q to contain %q\nFull message:\n%s", s, substr, s)
	}
}

func assertNotContains(t *testing.T, s, substr string) {
	t.Helper()
	if strings.Contains(strings.ToLower(s), strings.ToLower(substr)) {
		t.Fatalf("expected %q NOT to contain %q\nFull message:\n%s", s, substr, s)
	}
}

package config

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestHookPrefersParentProcessOverInheritedClaudeEnv(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("fake process tree in this test targets the Darwin ps path")
	}

	msg := runHookScript(t, "codex", "CLAUDE_CODE_NO_FLICKER=1")

	if !strings.Contains(msg, "Co-authored-by: OpenAI Codex <noreply@openai.com>") {
		t.Fatalf("expected Codex trailer, got:\n%s", msg)
	}
	if strings.Contains(msg, "Co-authored-by: Claude <noreply@anthropic.com>") {
		t.Fatalf("did not expect Claude trailer, got:\n%s", msg)
	}
}

func TestHookIgnoresClaudeCodeNoFlickerAsRuntimeMarker(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("fake process tree in this test targets the Darwin ps path")
	}

	msg := runHookScript(t, "none", "CLAUDE_CODE_NO_FLICKER=1")

	if strings.Contains(strings.ToLower(msg), "co-authored-by:") {
		t.Fatalf("expected no trailer, got:\n%s", msg)
	}
}

func TestHookUsesExplicitClaudeRuntimeMarkerAsFallback(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("fake process tree in this test targets the Darwin ps path")
	}

	msg := runHookScript(t, "none", "CLAUDE_CODE_SIMPLE=1")

	if !strings.Contains(msg, "Co-authored-by: Claude <noreply@anthropic.com>") {
		t.Fatalf("expected Claude trailer, got:\n%s", msg)
	}
}

func runHookScript(t *testing.T, processTree string, env ...string) string {
	t.Helper()

	dir := t.TempDir()
	binDir := filepath.Join(dir, "bin")
	if err := os.Mkdir(binDir, 0o755); err != nil {
		t.Fatal(err)
	}

	writeExecutable(t, filepath.Join(binDir, "uname"), `#!/usr/bin/env bash
if [ "${1:-}" = "-s" ]; then
  echo Darwin
else
  echo Darwin
fi
`)
	writeExecutable(t, filepath.Join(binDir, "ps"), `#!/usr/bin/env bash
field=""
pid=""
while [ "$#" -gt 0 ]; do
  case "$1" in
    -o)
      field="$2"
      shift 2
      ;;
    -p)
      pid="$2"
      shift 2
      ;;
    *)
      shift
      ;;
  esac
done

case "${AI_TRAILER_TEST_PROCESS_TREE:-none}:$field" in
  codex:comm=)
    if [ "$pid" = "200" ]; then echo codex; else echo zsh; fi
    ;;
  codex:ppid=)
    if [ "$pid" = "200" ]; then echo 1; else echo 200; fi
    ;;
  none:comm=)
    echo zsh
    ;;
  none:ppid=)
    echo 1
    ;;
  *)
    echo 1
    ;;
esac
`)

	hookPath := filepath.Join(dir, "prepare-commit-msg")
	writeExecutable(t, hookPath, HookScript)

	msgPath := filepath.Join(dir, "COMMIT_EDITMSG")
	if err := os.WriteFile(msgPath, []byte("subject\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	cmd := exec.Command("bash", hookPath, msgPath, "message")
	cmd.Env = append(os.Environ(),
		"PATH="+binDir+string(os.PathListSeparator)+os.Getenv("PATH"),
		"AI_TRAILER_TEST_PROCESS_TREE="+processTree,
	)
	cmd.Env = append(cmd.Env, env...)

	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("hook failed: %v\n%s", err, output)
	}

	msg, err := os.ReadFile(msgPath)
	if err != nil {
		t.Fatal(err)
	}
	return string(msg)
}

func writeExecutable(t *testing.T, path string, content string) {
	t.Helper()

	if err := os.WriteFile(path, []byte(content), 0o755); err != nil {
		t.Fatal(err)
	}
}

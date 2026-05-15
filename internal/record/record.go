// Package record provides the recording/audit mechanism for AI tool usage.
// It maintains a JSONL log of all AI-assisted commits and configurations,
// queryable via the CLI for metrics and reporting.
package record

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Entry represents a single recorded event.
type Entry struct {
	Timestamp  string `json:"timestamp"`
	Event      string `json:"event"`       // "detect", "configure", "commit", "uninstall"
	Tool       string `json:"tool"`        // Tool ID (e.g., "claude-code")
	ToolName   string `json:"tool_name"`   // Human-readable name
	Action     string `json:"action"`      // What was done
	Repo       string `json:"repo"`        // Git repo path (for commit events)
	CommitHash string `json:"commit_hash"` // Git commit hash (for commit events)
	Trailer    string `json:"trailer"`     // The trailer that was added
	Extra      string `json:"extra"`       // Additional info
}

// Recorder manages the JSONL log file.
type Recorder struct {
	Path string
}

// New creates a new Recorder with the log file at the default location.
func New() (*Recorder, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("cannot find home directory: %w", err)
	}
	dir := filepath.Join(home, ".ai-trailer")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("cannot create config directory: %w", err)
	}
	return &Recorder{Path: filepath.Join(dir, "records.jsonl")}, nil
}

// Log appends an entry to the JSONL log file.
func (r *Recorder) Log(entry Entry) error {
	entry.Timestamp = time.Now().Format(time.RFC3339)

	f, err := os.OpenFile(r.Path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("cannot open log file: %w", err)
	}
	defer f.Close()

	data, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("cannot marshal entry: %w", err)
	}

	if _, err := f.Write(append(data, '\n')); err != nil {
		return fmt.Errorf("cannot write entry: %w", err)
	}
	return nil
}

// LogConfigure records a configuration event.
func (r *Recorder) LogConfigure(toolID, toolName, action string) error {
	return r.Log(Entry{
		Event:    "configure",
		Tool:     toolID,
		ToolName: toolName,
		Action:   action,
	})
}

// LogDetect records a detection event for all found tools.
func (r *Recorder) LogDetect(found []string) error {
	return r.Log(Entry{
		Event: "detect",
		Extra: fmt.Sprintf("Found %d tools: %s", len(found), strings.Join(found, ", ")),
	})
}

// LogCommit records a commit event (called from the git hook).
func (r *Recorder) LogCommit(toolID, trailer, repo, commitHash string) error {
	return r.Log(Entry{
		Event:      "commit",
		Tool:       toolID,
		Trailer:    trailer,
		Repo:       repo,
		CommitHash: commitHash,
	})
}

// Summary represents aggregated statistics.
type Summary struct {
	TotalEvents  int               `json:"total_events"`
	TotalCommits int               `json:"total_commits"`
	ByTool       map[string]int    `json:"by_tool"`
	ByEvent      map[string]int    `json:"by_event"`
	RecentEvents []Entry           `json:"recent_events"`
}

// Query reads the log and returns a summary.
func (r *Recorder) Query(limit int) (*Summary, error) {
	data, err := os.ReadFile(r.Path)
	if err != nil {
		if os.IsNotExist(err) {
			return &Summary{}, nil
		}
		return nil, err
	}

	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	summary := &Summary{
		ByTool:  make(map[string]int),
		ByEvent: make(map[string]int),
	}

	var entries []Entry
	for _, line := range lines {
		if line == "" {
			continue
		}
		var e Entry
		if err := json.Unmarshal([]byte(line), &e); err != nil {
			continue
		}
		entries = append(entries, e)
		summary.TotalEvents++
		summary.ByEvent[e.Event]++
		if e.Tool != "" {
			summary.ByTool[e.Tool]++
		}
		if e.Event == "commit" {
			summary.TotalCommits++
		}
	}

	// Recent events (last N)
	if limit > 0 && len(entries) > limit {
		entries = entries[len(entries)-limit:]
	}
	summary.RecentEvents = entries

	return summary, nil
}

// FormatSummary returns a human-readable summary string.
func (r *Recorder) FormatSummary(limit int) (string, error) {
	s, err := r.Query(limit)
	if err != nil {
		return "", err
	}
	if s.TotalEvents == 0 {
		return "No records yet. Run 'ai-trailer configure' to get started.", nil
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("📊 AI Trailer Records\n"))
	sb.WriteString(fmt.Sprintf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n"))
	sb.WriteString(fmt.Sprintf("Total events:    %d\n", s.TotalEvents))
	sb.WriteString(fmt.Sprintf("AI-assisted commits: %d\n", s.TotalCommits))
	sb.WriteString(fmt.Sprintf("\nBy tool:\n"))
	for tool, count := range s.ByTool {
		sb.WriteString(fmt.Sprintf("  %-20s %d\n", tool+":", count))
	}
	sb.WriteString(fmt.Sprintf("\nBy event type:\n"))
	for event, count := range s.ByEvent {
		sb.WriteString(fmt.Sprintf("  %-20s %d\n", event+":", count))
	}
	if len(s.RecentEvents) > 0 {
		sb.WriteString(fmt.Sprintf("\nRecent events:\n"))
		for _, e := range s.RecentEvents {
			sb.WriteString(fmt.Sprintf("  [%s] %s: %s — %s\n",
				e.Timestamp[:19], e.Event, e.Tool, e.Action))
		}
	}
	return sb.String(), nil
}

package agentcontext

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

const (
	LedgerPath           = ".agents/agent_context_ledger.jsonl"
	ReferenceOnlyPolicy  = "reference_only_no_repo_write"
	DelegatedWritePolicy = "delegated_repo_write_codex_validated"
)

type Record struct {
	Cycle                   int      `json:"cycle"`
	AgentID                 string   `json:"agent_id"`
	Nickname                string   `json:"nickname"`
	TaskKind                string   `json:"task_kind"`
	Scope                   string   `json:"scope"`
	Status                  string   `json:"status"`
	UsagePolicy             string   `json:"usage_policy"`
	Summary                 string   `json:"summary"`
	Evidence                []string `json:"evidence"`
	Risks                   []string `json:"risks"`
	IntegrationDecision     string   `json:"integration_decision"`
	RepoWriteAllowed        bool     `json:"repo_write_allowed"`
	CodexValidationRequired bool     `json:"codex_validation_required"`
	ClosedBeforeCheckpoint  bool     `json:"closed_before_checkpoint"`
	RecordedAt              string   `json:"recorded_at"`
}

type Entry struct {
	Line   int
	Record Record
}

type Issue struct {
	Code    string
	Message string
}

type Report struct {
	Issues []Issue
}

var (
	agentIDPattern  = regexp.MustCompile(`^[a-z0-9-]{8,}$`)
	taskKindPattern = regexp.MustCompile(`^[a-z0-9]+(_[a-z0-9]+)*$`)
)

func Validate(root string) Report {
	records, loadReport := LoadRecords(root)
	issues := append([]Issue{}, loadReport.Issues...)
	if !loadReport.Passed() {
		return Report{Issues: issues}
	}
	if len(records) == 0 {
		issues = append(issues, Issue{Code: "agent_context_ledger_empty", Message: LedgerPath})
	}
	seen := make(map[string]int)
	for _, entry := range records {
		report := ValidateRecord(entry.Record)
		for _, issue := range report.Issues {
			issue.Message = fmt.Sprintf("line=%d %s", entry.Line, issue.Message)
			issues = append(issues, issue)
		}
		key := fmt.Sprintf("%d:%s:%s", entry.Record.Cycle, entry.Record.AgentID, entry.Record.TaskKind)
		if previous := seen[key]; previous > 0 {
			issues = append(issues, Issue{Code: "agent_context_duplicate_record", Message: fmt.Sprintf("line=%d previous_line=%d key=%s", entry.Line, previous, key)})
		}
		seen[key] = entry.Line
	}
	return Report{Issues: issues}
}

func LoadRecords(root string) ([]Entry, Report) {
	projectRoot, err := findProjectRoot(root)
	if err != nil {
		return nil, Report{Issues: []Issue{{Code: "project_root_not_found", Message: err.Error()}}}
	}
	path := filepath.Join(projectRoot, ".agents", "agent_context_ledger.jsonl")
	file, err := os.Open(path)
	if err != nil {
		return nil, Report{Issues: []Issue{{Code: "agent_context_ledger_missing", Message: err.Error()}}}
	}
	defer file.Close()

	entries := make([]Entry, 0)
	issues := make([]Issue, 0)
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 0, 4096), 65536)
	lineNumber := 0
	for scanner.Scan() {
		lineNumber++
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var record Record
		if err := json.Unmarshal([]byte(line), &record); err != nil {
			issues = append(issues, Issue{Code: "agent_context_invalid_json", Message: fmt.Sprintf("line=%d %s", lineNumber, err.Error())})
			continue
		}
		entries = append(entries, Entry{Line: lineNumber, Record: record})
	}
	if err := scanner.Err(); err != nil {
		issues = append(issues, Issue{Code: "agent_context_scan_failed", Message: err.Error()})
	}
	return entries, Report{Issues: issues}
}

func ValidateRecord(record Record) Report {
	issues := make([]Issue, 0)
	if record.Cycle <= 0 {
		issues = append(issues, Issue{Code: "agent_context_invalid_cycle", Message: fmt.Sprintf("%d", record.Cycle)})
	}
	if record.AgentID == "" || !agentIDPattern.MatchString(record.AgentID) {
		issues = append(issues, Issue{Code: "agent_context_invalid_agent_id", Message: record.AgentID})
	}
	if strings.TrimSpace(record.Nickname) == "" {
		issues = append(issues, Issue{Code: "agent_context_missing_nickname", Message: record.AgentID})
	}
	if record.TaskKind == "" || !taskKindPattern.MatchString(record.TaskKind) {
		issues = append(issues, Issue{Code: "agent_context_invalid_task_kind", Message: record.TaskKind})
	}
	if strings.TrimSpace(record.Scope) == "" {
		issues = append(issues, Issue{Code: "agent_context_missing_scope", Message: record.AgentID})
	}
	if record.Status != "completed" && record.Status != "closed" && record.Status != "integrated" {
		issues = append(issues, Issue{Code: "agent_context_status_invalid", Message: record.Status})
	}
	switch record.UsagePolicy {
	case ReferenceOnlyPolicy:
		if record.RepoWriteAllowed {
			issues = append(issues, Issue{Code: "agent_context_repo_write_allowed", Message: record.AgentID})
		}
	case DelegatedWritePolicy:
		if !record.RepoWriteAllowed {
			issues = append(issues, Issue{Code: "agent_context_delegated_write_flag_missing", Message: record.AgentID})
		}
	default:
		issues = append(issues, Issue{Code: "agent_context_usage_policy_invalid", Message: record.UsagePolicy})
		if record.RepoWriteAllowed {
			issues = append(issues, Issue{Code: "agent_context_repo_write_allowed", Message: record.AgentID})
		}
	}
	if len(strings.Fields(record.Summary)) < 8 {
		issues = append(issues, Issue{Code: "agent_context_summary_too_short", Message: record.AgentID})
	}
	if len(record.Evidence) == 0 {
		issues = append(issues, Issue{Code: "agent_context_missing_evidence", Message: record.AgentID})
	}
	if len(record.Risks) == 0 {
		issues = append(issues, Issue{Code: "agent_context_missing_risk", Message: record.AgentID})
	}
	if strings.TrimSpace(record.IntegrationDecision) == "" {
		issues = append(issues, Issue{Code: "agent_context_missing_integration_decision", Message: record.AgentID})
	}
	if !record.CodexValidationRequired {
		issues = append(issues, Issue{Code: "agent_context_validation_not_required", Message: record.AgentID})
	}
	if !record.ClosedBeforeCheckpoint {
		issues = append(issues, Issue{Code: "agent_context_not_closed_before_checkpoint", Message: record.AgentID})
	}
	if strings.TrimSpace(record.RecordedAt) == "" {
		issues = append(issues, Issue{Code: "agent_context_missing_recorded_at", Message: record.AgentID})
	}
	return Report{Issues: issues}
}

func (report Report) Passed() bool {
	return len(report.Issues) == 0
}

func (report Report) Messages() []string {
	messages := make([]string, 0, len(report.Issues))
	for _, issue := range report.Issues {
		if issue.Message == "" {
			messages = append(messages, issue.Code)
			continue
		}
		messages = append(messages, issue.Code+":"+issue.Message)
	}
	return messages
}

func (report Report) Codes() []string {
	codes := make([]string, 0, len(report.Issues))
	for _, issue := range report.Issues {
		codes = append(codes, issue.Code)
	}
	return codes
}

func (report Report) HasIssue(code string) bool {
	for _, issue := range report.Issues {
		if issue.Code == code {
			return true
		}
	}
	return false
}

func findProjectRoot(start string) (string, error) {
	dir, err := filepath.Abs(start)
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("go.mod not found from %s", start)
		}
		dir = parent
	}
}

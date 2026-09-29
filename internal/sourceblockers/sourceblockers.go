package sourceblockers

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

type Record struct {
	TermID                string   `json:"term_id"`
	Term                  string   `json:"term"`
	SourceReviewStatus    string   `json:"source_review_status"`
	ApprovalAllowed       bool     `json:"approval_allowed"`
	PublicationAllowed    bool     `json:"publication_allowed"`
	PublicPath            string   `json:"public_path"`
	CurrentSourceID       string   `json:"current_source_id"`
	CurrentSourceURL      string   `json:"current_source_url"`
	RequiredSourceTypes   []string `json:"required_source_types"`
	MissingRequirements   []string `json:"missing_requirements"`
	SpecificityReason     string   `json:"specificity_reason"`
	CheckedAt             string   `json:"checked_at"`
	NextResearchDirection string   `json:"next_research_direction"`
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

var termIDPattern = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)

func Validate(root string) Report {
	entries, report := LoadRecords(root)
	if !report.Passed() {
		return report
	}
	issues := make([]Issue, 0)
	seen := make(map[string]bool)
	for _, entry := range entries {
		report := ValidateRecord(entry.Record)
		for _, issue := range report.Issues {
			issue.Message = fmt.Sprintf("line=%d %s", entry.Line, issue.Message)
			issues = append(issues, issue)
		}
		if seen[entry.Record.TermID] {
			issues = append(issues, Issue{Code: "duplicate_source_blocker", Message: entry.Record.TermID})
		}
		seen[entry.Record.TermID] = true
	}
	return Report{Issues: issues}
}

func HasBlockerForTerm(root string, termID string) bool {
	entries, report := LoadRecords(root)
	if !report.Passed() {
		return false
	}
	for _, entry := range entries {
		if entry.Record.TermID == termID && ValidateRecord(entry.Record).Passed() {
			return true
		}
	}
	return false
}

func LoadRecords(root string) ([]Entry, Report) {
	projectRoot, err := findProjectRoot(root)
	if err != nil {
		return nil, Report{Issues: []Issue{{Code: "project_root_not_found", Message: err.Error()}}}
	}
	path := filepath.Join(projectRoot, "data", "editorial", "source_blockers.jsonl")
	file, err := os.Open(path)
	if err != nil {
		return nil, Report{Issues: []Issue{{Code: "source_blockers_missing", Message: err.Error()}}}
	}
	defer file.Close()

	issues := make([]Issue, 0)
	entries := make([]Entry, 0)
	scanner := bufio.NewScanner(file)
	lineNumber := 0
	for scanner.Scan() {
		lineNumber++
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var record Record
		if err := json.Unmarshal([]byte(line), &record); err != nil {
			issues = append(issues, Issue{Code: "source_blocker_invalid_json", Message: fmt.Sprintf("line=%d %s", lineNumber, err.Error())})
			continue
		}
		entries = append(entries, Entry{Line: lineNumber, Record: record})
	}
	if err := scanner.Err(); err != nil {
		issues = append(issues, Issue{Code: "source_blocker_scan_failed", Message: err.Error()})
	}
	return entries, Report{Issues: issues}
}

func ValidateRecord(record Record) Report {
	issues := make([]Issue, 0)
	if record.TermID == "" || !termIDPattern.MatchString(record.TermID) {
		issues = append(issues, Issue{Code: "invalid_source_blocker_term_id", Message: record.TermID})
	}
	if record.Term == "" {
		issues = append(issues, Issue{Code: "source_blocker_missing_term", Message: record.TermID})
	}
	if record.SourceReviewStatus != "needs_specific_source" {
		issues = append(issues, Issue{Code: "source_blocker_not_blocking", Message: record.TermID + ":" + record.SourceReviewStatus})
	}
	if record.ApprovalAllowed {
		issues = append(issues, Issue{Code: "source_blocker_approval_allowed", Message: record.TermID})
	}
	if record.PublicationAllowed {
		issues = append(issues, Issue{Code: "source_blocker_publication_allowed", Message: record.TermID})
	}
	if record.PublicPath != "" {
		issues = append(issues, Issue{Code: "source_blocker_has_public_path", Message: record.PublicPath})
	}
	if record.CurrentSourceID == "" || record.CurrentSourceURL == "" {
		issues = append(issues, Issue{Code: "source_blocker_without_current_source", Message: record.TermID})
	}
	if !isOfficialURL(record.CurrentSourceURL) {
		issues = append(issues, Issue{Code: "source_blocker_untrusted_source_url", Message: record.CurrentSourceURL})
	}
	if len(record.RequiredSourceTypes) == 0 {
		issues = append(issues, Issue{Code: "source_blocker_without_required_types", Message: record.TermID})
	}
	if len(record.MissingRequirements) == 0 {
		issues = append(issues, Issue{Code: "source_blocker_without_missing_requirements", Message: record.TermID})
	}
	if len(strings.Fields(record.SpecificityReason)) < 8 {
		issues = append(issues, Issue{Code: "source_blocker_thin_reason", Message: record.TermID})
	}
	if record.CheckedAt == "" {
		issues = append(issues, Issue{Code: "source_blocker_without_checked_at", Message: record.TermID})
	}
	if len(strings.Fields(record.NextResearchDirection)) < 6 {
		issues = append(issues, Issue{Code: "source_blocker_without_next_research", Message: record.TermID})
	}
	return Report{Issues: issues}
}

func isOfficialURL(value string) bool {
	prefixes := []string{
		"https://www.gov.br/",
		"https://www.cnj.jus.br/",
		"https://www.planalto.gov.br/",
	}
	for _, prefix := range prefixes {
		if strings.HasPrefix(value, prefix) {
			return true
		}
	}
	return false
}

func (r Report) Passed() bool {
	return len(r.Issues) == 0
}

func (r Report) HasIssue(code string) bool {
	for _, issue := range r.Issues {
		if issue.Code == code {
			return true
		}
	}
	return false
}

func (r Report) Codes() []string {
	codes := make([]string, 0, len(r.Issues))
	for _, issue := range r.Issues {
		codes = append(codes, issue.Code)
	}
	return codes
}

func (r Report) Messages() []string {
	messages := make([]string, 0, len(r.Issues))
	for _, issue := range r.Issues {
		messages = append(messages, issue.Code+": "+issue.Message)
	}
	return messages
}

func findProjectRoot(start string) (string, error) {
	current, err := filepath.Abs(start)
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Stat(filepath.Join(current, "go.mod")); err == nil {
			return current, nil
		}
		parent := filepath.Dir(current)
		if parent == current {
			return "", os.ErrNotExist
		}
		current = parent
	}
}

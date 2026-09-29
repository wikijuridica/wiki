package approvals

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"portaljuridico/internal/reviewqueue"
	"portaljuridico/internal/storage"
)

type Metadata struct {
	Reviewer   string
	ApprovedAt string
	Reason     string
}

type Record struct {
	TermID             string  `json:"term_id"`
	Term               string  `json:"term"`
	EditorialStatus    string  `json:"editorial_status"`
	IndexPolicy        string  `json:"index_policy"`
	PublicationAllowed bool    `json:"publication_allowed"`
	PublicPath         string  `json:"public_path"`
	SourceID           string  `json:"source_id"`
	SourceURL          string  `json:"source_url"`
	Author             string  `json:"author"`
	Reviewer           string  `json:"reviewer"`
	ApprovedAt         string  `json:"approved_at"`
	Reason             string  `json:"reason"`
	History            []Event `json:"history"`
}

type Event struct {
	Event string `json:"event"`
	At    string `json:"at"`
	Actor string `json:"actor"`
	Note  string `json:"note"`
}

type Issue struct {
	Code    string
	Message string
}

type Report struct {
	Issues []Issue
}

func Approve(root string, queued reviewqueue.Record, metadata Metadata) (bool, error) {
	records, report := LoadRecords(root)
	if !report.Passed() {
		return false, fmt.Errorf("load_approvals_failed=%s", strings.Join(report.Messages(), " | "))
	}
	for _, record := range records {
		if record.TermID == queued.TermID {
			return false, nil
		}
	}
	record := Record{
		TermID:             queued.TermID,
		Term:               queued.Term,
		EditorialStatus:    "approved",
		IndexPolicy:        "noindex",
		PublicationAllowed: false,
		PublicPath:         "",
		SourceID:           queued.SourceID,
		SourceURL:          queued.SourceURL,
		Author:             queued.Author,
		Reviewer:           metadata.Reviewer,
		ApprovedAt:         metadata.ApprovedAt,
		Reason:             metadata.Reason,
		History: []Event{
			{Event: "approved_editorial_only", At: metadata.ApprovedAt, Actor: metadata.Reviewer, Note: metadata.Reason},
		},
	}
	if report := ValidateRecord(record); !report.Passed() {
		return false, fmt.Errorf("invalid_approval_record=%s", strings.Join(report.Messages(), " | "))
	}
	return true, storage.AppendJSONL(root, "approved_drafts", record)
}

func Validate(root string) Report {
	records, report := LoadRecords(root)
	if !report.Passed() {
		return report
	}
	issues := make([]Issue, 0)
	for _, record := range records {
		report := ValidateRecord(record)
		issues = append(issues, report.Issues...)
	}
	return Report{Issues: issues}
}

func LoadRecords(root string) ([]Record, Report) {
	projectRoot, err := findProjectRoot(root)
	if err != nil {
		return nil, Report{Issues: []Issue{{Code: "project_root_not_found", Message: err.Error()}}}
	}
	path := filepath.Join(projectRoot, "data", "editorial", "approved_drafts.jsonl")
	file, err := os.Open(path)
	if err != nil {
		return nil, Report{Issues: []Issue{{Code: "approvals_missing", Message: err.Error()}}}
	}
	defer file.Close()

	issues := make([]Issue, 0)
	records := make([]Record, 0)
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
			issues = append(issues, Issue{Code: "approval_invalid_json", Message: fmt.Sprintf("line=%d %s", lineNumber, err.Error())})
			continue
		}
		records = append(records, record)
	}
	if err := scanner.Err(); err != nil {
		issues = append(issues, Issue{Code: "approval_scan_failed", Message: err.Error()})
	}
	return records, Report{Issues: issues}
}

func ValidateRecord(record Record) Report {
	issues := make([]Issue, 0)
	if record.TermID == "" || record.Term == "" {
		issues = append(issues, Issue{Code: "approval_missing_term", Message: record.TermID})
	}
	if record.EditorialStatus != "approved" || record.IndexPolicy != "noindex" {
		issues = append(issues, Issue{Code: "approval_not_approved_noindex", Message: record.EditorialStatus + "/" + record.IndexPolicy})
	}
	if record.PublicationAllowed {
		issues = append(issues, Issue{Code: "approval_publication_allowed", Message: record.TermID})
	}
	if record.PublicPath != "" {
		issues = append(issues, Issue{Code: "approval_has_public_path", Message: record.PublicPath})
	}
	if record.SourceID == "" || record.SourceURL == "" {
		issues = append(issues, Issue{Code: "approval_without_source", Message: record.TermID})
	}
	if record.Author == "" || record.Reviewer == "" || record.ApprovedAt == "" || record.Reason == "" {
		issues = append(issues, Issue{Code: "approval_missing_editorial_metadata", Message: record.TermID})
	}
	if len(record.History) == 0 {
		issues = append(issues, Issue{Code: "approval_without_history", Message: record.TermID})
	}
	return Report{Issues: issues}
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

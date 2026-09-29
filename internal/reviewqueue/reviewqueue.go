package reviewqueue

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"portaljuridico/internal/draftlab"
	"portaljuridico/internal/storage"
)

type Metadata struct {
	Author    string
	CreatedAt string
	Reason    string
}

type Record struct {
	TermID             string  `json:"term_id"`
	Term               string  `json:"term"`
	DraftStatus        string  `json:"draft_status"`
	IndexPolicy        string  `json:"index_policy"`
	ReviewStatus       string  `json:"review_status"`
	PublicationAllowed bool    `json:"publication_allowed"`
	PublicPath         string  `json:"public_path"`
	SourceID           string  `json:"source_id"`
	SourceURL          string  `json:"source_url"`
	Author             string  `json:"author"`
	Reviewer           string  `json:"reviewer"`
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

func EnqueueDraft(root string, draft draftlab.Draft, metadata Metadata) (bool, error) {
	records, report := LoadRecords(root)
	if !report.Passed() {
		return false, fmt.Errorf("load_review_queue_failed=%s", strings.Join(report.Messages(), " | "))
	}
	record := Record{
		TermID:             draft.TermID,
		Term:               draft.Term,
		DraftStatus:        draft.Status,
		IndexPolicy:        draft.IndexPolicy,
		ReviewStatus:       "needs_review",
		PublicationAllowed: false,
		PublicPath:         "",
		SourceID:           draft.SourceID,
		SourceURL:          draft.SourceURL,
		Author:             metadata.Author,
		Reviewer:           "pending",
		Reason:             metadata.Reason,
		History: []Event{
			{Event: "queued_for_review", At: metadata.CreatedAt, Actor: "codex", Note: metadata.Reason},
		},
	}
	if report := ValidateRecord(record); !report.Passed() {
		return false, fmt.Errorf("invalid_review_record=%s", strings.Join(report.Messages(), " | "))
	}
	for i, existing := range records {
		if existing.TermID == draft.TermID {
			record.History = mergeHistory(existing.History, record.History)
			records[i] = record
			return false, rewriteRecords(root, records)
		}
	}
	return true, storage.AppendJSONL(root, "review_queue", record)
}

func mergeHistory(existing []Event, next []Event) []Event {
	if len(existing) == 0 {
		return next
	}
	return existing
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
	path := filepath.Join(projectRoot, "data", "editorial", "review_queue.jsonl")
	file, err := os.Open(path)
	if err != nil {
		return nil, Report{Issues: []Issue{{Code: "review_queue_missing", Message: err.Error()}}}
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
			issues = append(issues, Issue{Code: "review_queue_invalid_json", Message: fmt.Sprintf("line=%d %s", lineNumber, err.Error())})
			continue
		}
		records = append(records, record)
	}
	if err := scanner.Err(); err != nil {
		issues = append(issues, Issue{Code: "review_queue_scan_failed", Message: err.Error()})
	}
	return records, Report{Issues: issues}
}

func rewriteRecords(root string, records []Record) error {
	projectRoot, err := findProjectRoot(root)
	if err != nil {
		return err
	}
	path := filepath.Join(projectRoot, "data", "editorial", "review_queue.jsonl")
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	defer file.Close()
	for _, record := range records {
		if report := ValidateRecord(record); !report.Passed() {
			return fmt.Errorf("invalid_review_record=%s", strings.Join(report.Messages(), " | "))
		}
		data, err := json.Marshal(record)
		if err != nil {
			return err
		}
		if _, err := file.Write(append(data, '\n')); err != nil {
			return err
		}
	}
	return nil
}

func ValidateRecord(record Record) Report {
	issues := make([]Issue, 0)
	if record.TermID == "" || record.Term == "" {
		issues = append(issues, Issue{Code: "review_queue_missing_term", Message: record.TermID})
	}
	if record.DraftStatus != "draft" || record.IndexPolicy != "noindex" {
		issues = append(issues, Issue{Code: "review_queue_not_draft_noindex", Message: record.DraftStatus + "/" + record.IndexPolicy})
	}
	if record.ReviewStatus != "needs_review" {
		issues = append(issues, Issue{Code: "review_queue_invalid_status", Message: record.ReviewStatus})
	}
	if record.PublicationAllowed {
		issues = append(issues, Issue{Code: "review_queue_publication_allowed", Message: record.TermID})
	}
	if record.PublicPath != "" {
		issues = append(issues, Issue{Code: "review_queue_has_public_path", Message: record.PublicPath})
	}
	if record.SourceID == "" || record.SourceURL == "" {
		issues = append(issues, Issue{Code: "review_queue_without_source", Message: record.TermID})
	}
	if record.Author == "" || record.Reviewer == "" || record.Reason == "" {
		issues = append(issues, Issue{Code: "review_queue_missing_editorial_metadata", Message: record.TermID})
	}
	if len(record.History) == 0 {
		issues = append(issues, Issue{Code: "review_queue_without_history", Message: record.TermID})
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

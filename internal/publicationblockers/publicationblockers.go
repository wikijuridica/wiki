package publicationblockers

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"portaljuridico/internal/approvals"
	"portaljuridico/internal/storage"
)

type Record struct {
	TermID              string   `json:"term_id"`
	Term                string   `json:"term"`
	PublicationStatus   string   `json:"publication_status"`
	PublicationAllowed  bool     `json:"publication_allowed"`
	PublicPath          string   `json:"public_path"`
	SourceID            string   `json:"source_id"`
	SourceURL           string   `json:"source_url"`
	MissingRequirements []string `json:"missing_requirements"`
	CheckedAt           string   `json:"checked_at"`
}

type Issue struct {
	Code    string
	Message string
}

type Report struct {
	Issues []Issue
}

var defaultMissingRequirements = []string{
	"specific_source_research",
	"legal_reviewer_approval",
	"unique_intent_confirmation",
	"canonical_public_path",
	"seo_metadata_final",
	"whatsapp_cta_policy_final",
	"publication_batch_capacity",
}

func AppendFromApproval(root string, approval approvals.Record, checkedAt string) (bool, error) {
	records, report := LoadRecords(root)
	if !report.Passed() {
		return false, fmt.Errorf("load_publication_blockers_failed=%s", strings.Join(report.Messages(), " | "))
	}
	for _, record := range records {
		if record.TermID == approval.TermID {
			return false, nil
		}
	}
	record := Record{
		TermID:              approval.TermID,
		Term:                approval.Term,
		PublicationStatus:   "blocked",
		PublicationAllowed:  false,
		PublicPath:          "",
		SourceID:            approval.SourceID,
		SourceURL:           approval.SourceURL,
		MissingRequirements: append([]string{}, defaultMissingRequirements...),
		CheckedAt:           checkedAt,
	}
	if report := ValidateRecord(record); !report.Passed() {
		return false, fmt.Errorf("invalid_publication_blocker=%s", strings.Join(report.Messages(), " | "))
	}
	return true, storage.AppendJSONL(root, "publication_blockers", record)
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
	path := filepath.Join(projectRoot, "data", "editorial", "publication_blockers.jsonl")
	file, err := os.Open(path)
	if err != nil {
		return nil, Report{Issues: []Issue{{Code: "publication_blockers_missing", Message: err.Error()}}}
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
			issues = append(issues, Issue{Code: "publication_blocker_invalid_json", Message: fmt.Sprintf("line=%d %s", lineNumber, err.Error())})
			continue
		}
		records = append(records, record)
	}
	if err := scanner.Err(); err != nil {
		issues = append(issues, Issue{Code: "publication_blocker_scan_failed", Message: err.Error()})
	}
	return records, Report{Issues: issues}
}

func ValidateRecord(record Record) Report {
	issues := make([]Issue, 0)
	if record.TermID == "" || record.Term == "" {
		issues = append(issues, Issue{Code: "publication_blocker_missing_term", Message: record.TermID})
	}
	if record.PublicationStatus != "blocked" {
		issues = append(issues, Issue{Code: "publication_blocker_not_blocked", Message: record.PublicationStatus})
	}
	if record.PublicationAllowed {
		issues = append(issues, Issue{Code: "publication_blocker_allows_publication", Message: record.TermID})
	}
	if record.PublicPath != "" {
		issues = append(issues, Issue{Code: "publication_blocker_has_public_path", Message: record.PublicPath})
	}
	if len(record.MissingRequirements) == 0 {
		issues = append(issues, Issue{Code: "publication_blocker_without_requirements", Message: record.TermID})
	}
	if record.CheckedAt == "" {
		issues = append(issues, Issue{Code: "publication_blocker_without_checked_at", Message: record.TermID})
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

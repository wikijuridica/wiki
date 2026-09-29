package editorialdrafts

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"portaljuridico/internal/draftlab"
	"portaljuridico/internal/quality"
	"portaljuridico/internal/storage"
)

type Record struct {
	TermID      string `json:"term_id"`
	Term        string `json:"term"`
	Status      string `json:"status"`
	IndexPolicy string `json:"index_policy"`
	PublicPath  string `json:"public_path"`
	SourceID    string `json:"source_id"`
	SourceURL   string `json:"source_url"`
	Text        string `json:"text"`
}

func (r Record) AsDraft() draftlab.Draft {
	return draftlab.Draft{
		TermID:      r.TermID,
		Term:        r.Term,
		Status:      r.Status,
		IndexPolicy: r.IndexPolicy,
		PublicPath:  r.PublicPath,
		SourceID:    r.SourceID,
		SourceURL:   r.SourceURL,
		Text:        r.Text,
	}
}

type Issue struct {
	Code    string
	Message string
}

type Report struct {
	Issues []Issue
}

func Append(root string, draft draftlab.Draft) error {
	record := Record{
		TermID:      draft.TermID,
		Term:        draft.Term,
		Status:      draft.Status,
		IndexPolicy: draft.IndexPolicy,
		PublicPath:  draft.PublicPath,
		SourceID:    draft.SourceID,
		SourceURL:   draft.SourceURL,
		Text:        draft.Text,
	}
	if report := ValidateRecord(record); !report.Passed() {
		return fmt.Errorf("invalid_editorial_draft=%s", strings.Join(report.Messages(), " | "))
	}
	return storage.AppendJSONL(root, "editorial_drafts", record)
}

func AppendIfMissing(root string, draft draftlab.Draft) (bool, error) {
	records, report := LoadRecords(root)
	if !report.Passed() {
		return false, fmt.Errorf("load_editorial_drafts_failed=%s", strings.Join(report.Messages(), " | "))
	}
	for _, record := range records {
		if record.TermID == draft.TermID {
			return false, nil
		}
	}
	return true, Append(root, draft)
}

func UpsertAll(root string, drafts []draftlab.Draft) error {
	records, report := LoadRecords(root)
	if !report.Passed() {
		return fmt.Errorf("load_editorial_drafts_failed=%s", strings.Join(report.Messages(), " | "))
	}
	byID := make(map[string]Record)
	order := make([]string, 0)
	for _, record := range records {
		if _, exists := byID[record.TermID]; !exists {
			order = append(order, record.TermID)
		}
		byID[record.TermID] = record
	}
	for _, draft := range drafts {
		record := recordFromDraft(draft)
		if report := ValidateRecord(record); !report.Passed() {
			return fmt.Errorf("invalid_editorial_draft=%s", strings.Join(report.Messages(), " | "))
		}
		if _, exists := byID[record.TermID]; !exists {
			order = append(order, record.TermID)
		}
		byID[record.TermID] = record
	}
	return rewriteRecords(root, order, byID)
}

func Validate(root string) Report {
	records, report := LoadRecords(root)
	if !report.Passed() {
		return report
	}
	issues := make([]Issue, 0)
	for _, record := range records {
		report := ValidateRecord(record)
		for _, issue := range report.Issues {
			issues = append(issues, issue)
		}
	}
	return Report{Issues: issues}
}

func recordFromDraft(draft draftlab.Draft) Record {
	return Record{
		TermID:      draft.TermID,
		Term:        draft.Term,
		Status:      draft.Status,
		IndexPolicy: draft.IndexPolicy,
		PublicPath:  draft.PublicPath,
		SourceID:    draft.SourceID,
		SourceURL:   draft.SourceURL,
		Text:        draft.Text,
	}
}

func rewriteRecords(root string, order []string, byID map[string]Record) error {
	projectRoot, err := findProjectRoot(root)
	if err != nil {
		return err
	}
	path := filepath.Join(projectRoot, "data", "editorial", "drafts.jsonl")
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	defer file.Close()
	for _, termID := range order {
		record, ok := byID[termID]
		if !ok {
			continue
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

func LoadRecords(root string) ([]Record, Report) {
	projectRoot, err := findProjectRoot(root)
	if err != nil {
		return nil, Report{Issues: []Issue{{Code: "project_root_not_found", Message: err.Error()}}}
	}
	path := filepath.Join(projectRoot, "data", "editorial", "drafts.jsonl")
	file, err := os.Open(path)
	if err != nil {
		return nil, Report{Issues: []Issue{{Code: "editorial_drafts_missing", Message: err.Error()}}}
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
			issues = append(issues, Issue{Code: "editorial_draft_invalid_json", Message: fmt.Sprintf("line=%d %s", lineNumber, err.Error())})
			continue
		}
		records = append(records, record)
	}
	if err := scanner.Err(); err != nil {
		issues = append(issues, Issue{Code: "editorial_draft_scan_failed", Message: err.Error()})
	}
	return records, Report{Issues: issues}
}

func ValidateRecord(record Record) Report {
	issues := make([]Issue, 0)
	if record.TermID == "" || record.Term == "" {
		issues = append(issues, Issue{Code: "editorial_draft_missing_term", Message: record.TermID})
	}
	if record.Status != "draft" || record.IndexPolicy != "noindex" {
		issues = append(issues, Issue{Code: "editorial_draft_not_draft_noindex", Message: record.Status + "/" + record.IndexPolicy})
	}
	if record.PublicPath != "" {
		issues = append(issues, Issue{Code: "editorial_draft_has_public_path", Message: record.PublicPath})
	}
	if record.SourceID == "" || record.SourceURL == "" {
		issues = append(issues, Issue{Code: "editorial_draft_without_source", Message: record.TermID})
	}
	if analysis := quality.AnalyzeText(record.Text); !analysis.Passed() {
		issues = append(issues, Issue{Code: "editorial_draft_quality_failed", Message: strings.Join(analysis.Messages(), " | ")})
	}
	return Report{Issues: issues}
}

func (r Report) Passed() bool {
	return len(r.Issues) == 0
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

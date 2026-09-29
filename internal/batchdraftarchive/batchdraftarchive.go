package batchdraftarchive

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"portaljuridico/internal/batchdrafts"
)

type Issue struct {
	Code    string
	Message string
}

type Report struct {
	Issues []Issue
}

func Validate(root string) Report {
	entries, report := LoadRecords(root)
	if !report.Passed() {
		return report
	}
	issues := make([]Issue, 0)
	if len(entries) < 600 {
		issues = append(issues, Issue{Code: "batch_draft_archive_too_few_drafts", Message: fmt.Sprintf("drafts=%d", len(entries))})
	}
	seen := make(map[string]int)
	perBatch := make(map[string]int)
	for _, entry := range entries {
		recordReport := batchdrafts.ValidateRecord(entry.Record)
		for _, issue := range recordReport.Issues {
			issues = append(issues, Issue{Code: "batch_draft_archive_invalid_record", Message: fmt.Sprintf("line=%d %s:%s", entry.Line, issue.Code, issue.Message)})
		}
		if entry.Record.SourceMatrixID == "" {
			issues = append(issues, Issue{Code: "batch_draft_archive_without_source_matrix", Message: entry.Record.UniqueIntentID})
		}
		if previous := seen[entry.Record.UniqueIntentID]; previous > 0 {
			issues = append(issues, Issue{Code: "batch_draft_archive_duplicate_intent", Message: fmt.Sprintf("line=%d previous_line=%d id=%s", entry.Line, previous, entry.Record.UniqueIntentID)})
		}
		seen[entry.Record.UniqueIntentID] = entry.Line
		perBatch[entry.Record.BatchID]++
	}
	for batchID, count := range perBatch {
		if count < 100 {
			issues = append(issues, Issue{Code: "batch_draft_archive_too_few_per_batch", Message: fmt.Sprintf("%s=%d", batchID, count)})
		}
	}
	if batchdrafts.RewrittenCount(entries) < 300 {
		issues = append(issues, Issue{Code: "batch_draft_archive_too_few_rewrites", Message: fmt.Sprintf("rewritten=%d", batchdrafts.RewrittenCount(entries))})
	}
	if max := batchdrafts.MaximumPairSimilarity(entries); max > 0.64 {
		issues = append(issues, Issue{Code: "batch_draft_archive_similarity_too_high", Message: fmt.Sprintf("max=%.4f", max)})
	}
	matrixReport := batchdrafts.ValidateSourceMatrixDiversity(entries)
	for _, issue := range matrixReport.Issues {
		issues = append(issues, Issue{Code: "batch_draft_archive_" + issue.Code, Message: issue.Message})
	}
	return Report{Issues: issues}
}

func LoadRecords(root string) ([]batchdrafts.Entry, Report) {
	projectRoot, err := findProjectRoot(root)
	if err != nil {
		return nil, Report{Issues: []Issue{{Code: "project_root_not_found", Message: err.Error()}}}
	}
	path := filepath.Join(projectRoot, "data", "editorial", "batch_draft_expansion_archive.jsonl")
	file, err := os.Open(path)
	if err != nil {
		return nil, Report{Issues: []Issue{{Code: "batch_draft_archive_missing", Message: err.Error()}}}
	}
	defer file.Close()

	issues := make([]Issue, 0)
	entries := make([]batchdrafts.Entry, 0)
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 0, 4096), 65536)
	lineNumber := 0
	for scanner.Scan() {
		lineNumber++
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var record batchdrafts.Record
		if err := json.Unmarshal([]byte(line), &record); err != nil {
			issues = append(issues, Issue{Code: "batch_draft_archive_invalid_json", Message: fmt.Sprintf("line=%d %s", lineNumber, err.Error())})
			continue
		}
		entries = append(entries, batchdrafts.Entry{Line: lineNumber, Record: record})
	}
	if err := scanner.Err(); err != nil {
		issues = append(issues, Issue{Code: "batch_draft_archive_scan_failed", Message: err.Error()})
	}
	return entries, Report{Issues: issues}
}

func (r Report) Passed() bool { return len(r.Issues) == 0 }

func (r Report) Codes() []string {
	codes := make([]string, 0, len(r.Issues))
	for _, issue := range r.Issues {
		codes = append(codes, issue.Code)
	}
	sort.Strings(codes)
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

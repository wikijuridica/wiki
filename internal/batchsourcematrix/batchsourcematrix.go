package batchsourcematrix

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"portaljuridico/internal/batchdrafts"
	"portaljuridico/internal/scalablebatches"
)

type Record struct {
	MatrixID                string   `json:"matrix_id"`
	BatchID                 string   `json:"batch_id"`
	LegalArea               string   `json:"legal_area"`
	SubthemeID              string   `json:"subtheme_id"`
	SourceStatus            string   `json:"source_status"`
	SourceURLs              []string `json:"source_urls"`
	SourceTypes             []string `json:"source_types"`
	SourceSpecificityScore  int      `json:"source_specificity_score"`
	OfficialSourcesVerified bool     `json:"official_sources_verified"`
	RobotsReviewRequired    bool     `json:"robots_review_required"`
	UsePolicy               string   `json:"use_policy"`
	RenderAllowed           bool     `json:"render_allowed"`
	SitemapAllowed          bool     `json:"sitemap_allowed"`
	PublicationAllowed      bool     `json:"publication_allowed"`
	PublicPath              string   `json:"public_path"`
	CheckedAt               string   `json:"checked_at"`
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

func Validate(root string) Report {
	entries, report := LoadRecords(root)
	if !report.Passed() {
		return report
	}
	issues := make([]Issue, 0)
	if len(entries) < 30 {
		issues = append(issues, Issue{Code: "source_matrix_too_few_subthemes", Message: fmt.Sprintf("records=%d", len(entries))})
	}
	validBatches := validBatchIDs(root)
	seen := make(map[string]int)
	perBatch := make(map[string]int)
	for _, entry := range entries {
		recordReport := ValidateRecord(entry.Record)
		for _, issue := range recordReport.Issues {
			issue.Message = fmt.Sprintf("line=%d %s", entry.Line, issue.Message)
			issues = append(issues, issue)
		}
		if !validBatches[entry.Record.BatchID] {
			issues = append(issues, Issue{Code: "source_matrix_unknown_batch", Message: fmt.Sprintf("line=%d batch=%s", entry.Line, entry.Record.BatchID)})
		}
		if previous := seen[entry.Record.MatrixID]; previous > 0 {
			issues = append(issues, Issue{Code: "source_matrix_duplicate_id", Message: fmt.Sprintf("line=%d previous_line=%d id=%s", entry.Line, previous, entry.Record.MatrixID)})
		}
		seen[entry.Record.MatrixID] = entry.Line
		perBatch[entry.Record.BatchID]++
	}
	for batchID, count := range perBatch {
		if count < 5 {
			issues = append(issues, Issue{Code: "source_matrix_too_few_batch_subthemes", Message: fmt.Sprintf("%s=%d", batchID, count)})
		}
	}
	return Report{Issues: issues}
}

func LoadRecords(root string) ([]Entry, Report) {
	projectRoot, err := findProjectRoot(root)
	if err != nil {
		return nil, Report{Issues: []Issue{{Code: "project_root_not_found", Message: err.Error()}}}
	}
	path := filepath.Join(projectRoot, "data", "editorial", "batch_source_matrix.jsonl")
	file, err := os.Open(path)
	if err != nil {
		return nil, Report{Issues: []Issue{{Code: "source_matrix_missing", Message: err.Error()}}}
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
			issues = append(issues, Issue{Code: "source_matrix_invalid_json", Message: fmt.Sprintf("line=%d %s", lineNumber, err.Error())})
			continue
		}
		entries = append(entries, Entry{Line: lineNumber, Record: record})
	}
	if err := scanner.Err(); err != nil {
		issues = append(issues, Issue{Code: "source_matrix_scan_failed", Message: err.Error()})
	}
	return entries, Report{Issues: issues}
}

func ValidateRecord(record Record) Report {
	issues := make([]Issue, 0)
	if record.MatrixID == "" || record.BatchID == "" || record.LegalArea == "" || record.SubthemeID == "" {
		issues = append(issues, Issue{Code: "source_matrix_missing_identity", Message: record.MatrixID})
	}
	if record.SourceStatus != "source_matrix_blocked" {
		issues = append(issues, Issue{Code: "source_matrix_invalid_status", Message: record.SourceStatus})
	}
	if officialURLCount(record.SourceURLs) < 2 {
		issues = append(issues, Issue{Code: "source_matrix_too_few_official_urls", Message: record.MatrixID})
	}
	if len(record.SourceTypes) < 2 {
		issues = append(issues, Issue{Code: "source_matrix_too_few_source_types", Message: record.MatrixID})
	}
	if record.SourceSpecificityScore < 80 {
		issues = append(issues, Issue{Code: "source_matrix_specificity_too_low", Message: fmt.Sprintf("%s=%d", record.MatrixID, record.SourceSpecificityScore)})
	}
	if !record.OfficialSourcesVerified {
		issues = append(issues, Issue{Code: "source_matrix_not_verified", Message: record.MatrixID})
	}
	if !record.RobotsReviewRequired {
		issues = append(issues, Issue{Code: "source_matrix_robots_review_missing", Message: record.MatrixID})
	}
	if record.UsePolicy != "reference_only_no_scraping" {
		issues = append(issues, Issue{Code: "source_matrix_use_policy_invalid", Message: record.UsePolicy})
	}
	if record.RenderAllowed {
		issues = append(issues, Issue{Code: "source_matrix_render_allowed", Message: record.MatrixID})
	}
	if record.SitemapAllowed {
		issues = append(issues, Issue{Code: "source_matrix_sitemap_allowed", Message: record.MatrixID})
	}
	if record.PublicationAllowed {
		issues = append(issues, Issue{Code: "source_matrix_publication_allowed", Message: record.MatrixID})
	}
	if record.PublicPath != "" {
		issues = append(issues, Issue{Code: "source_matrix_has_public_path", Message: record.PublicPath})
	}
	if record.CheckedAt == "" {
		issues = append(issues, Issue{Code: "source_matrix_without_checked_at", Message: record.MatrixID})
	}
	return Report{Issues: issues}
}

func ValidateDraftCoverage(entries []Entry, drafts []batchdrafts.Record) Report {
	matrixByID := make(map[string]Record)
	for _, entry := range entries {
		matrixByID[entry.Record.MatrixID] = entry.Record
	}
	issues := make([]Issue, 0)
	for _, draft := range drafts {
		if draft.SourceMatrixID == "" {
			issues = append(issues, Issue{Code: "draft_without_source_matrix", Message: draft.UniqueIntentID})
			continue
		}
		matrix, ok := matrixByID[draft.SourceMatrixID]
		if !ok {
			issues = append(issues, Issue{Code: "draft_unknown_source_matrix", Message: draft.UniqueIntentID + ":" + draft.SourceMatrixID})
			continue
		}
		if matrix.BatchID != draft.BatchID || matrix.LegalArea != draft.LegalArea {
			issues = append(issues, Issue{Code: "draft_source_matrix_mismatch", Message: draft.UniqueIntentID + ":" + draft.SourceMatrixID})
		}
		if report := ValidateRecord(matrix); !report.Passed() {
			for _, issue := range report.Issues {
				issues = append(issues, Issue{Code: "draft_source_matrix_invalid", Message: draft.UniqueIntentID + ":" + issue.Code})
			}
		}
	}
	return Report{Issues: issues}
}

func officialURLCount(urls []string) int {
	count := 0
	for _, url := range urls {
		if isOfficialURL(url) {
			count++
		}
	}
	return count
}

func isOfficialURL(url string) bool {
	prefixes := []string{
		"https://www.gov.br/",
		"https://www.planalto.gov.br/",
		"https://www.cnj.jus.br/",
		"https://atos.cnj.jus.br/",
		"https://www.bcb.gov.br/",
		"https://www.tst.jus.br/",
		"https://www.stj.jus.br/",
		"https://www.stf.jus.br/",
	}
	for _, prefix := range prefixes {
		if strings.HasPrefix(url, prefix) {
			return true
		}
	}
	return false
}

func validBatchIDs(root string) map[string]bool {
	entries, report := scalablebatches.LoadRecords(root)
	if !report.Passed() {
		return map[string]bool{}
	}
	valid := make(map[string]bool)
	for _, entry := range entries {
		if scalablebatches.ValidateRecord(entry.Record).Passed() {
			valid[entry.Record.BatchID] = true
		}
	}
	return valid
}

func (r Report) Passed() bool { return len(r.Issues) == 0 }

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

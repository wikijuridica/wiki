package batchprepublication

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"portaljuridico/internal/batchcandidatereviews"
	"portaljuridico/internal/content"
	"portaljuridico/internal/router"
	"portaljuridico/internal/seo"
)

type Record struct {
	PrepublicationID         string   `json:"prepublication_id"`
	ReviewID                 string   `json:"review_id"`
	GateID                   string   `json:"gate_id"`
	BatchID                  string   `json:"batch_id"`
	UniqueIntentID           string   `json:"unique_intent_id"`
	SourceMatrixID           string   `json:"source_matrix_id"`
	Term                     string   `json:"term"`
	CandidatePath            string   `json:"candidate_path"`
	CandidateCanonicalURL    string   `json:"candidate_canonical_url"`
	CandidateRobots          string   `json:"candidate_robots"`
	CandidateTitle           string   `json:"candidate_title"`
	CandidateMetaDescription string   `json:"candidate_meta_description"`
	SourceSpecificityStatus  string   `json:"source_specificity_status"`
	RemainingGates           []string `json:"remaining_gates"`
	RenderAllowed            bool     `json:"render_allowed"`
	SitemapAllowed           bool     `json:"sitemap_allowed"`
	PublicationAllowed       bool     `json:"publication_allowed"`
	PublicPath               string   `json:"public_path"`
	CheckedAt                string   `json:"checked_at"`
}

func (r Record) CanonicalUsesOfficialBase(baseURL string) bool {
	return strings.TrimRight(baseURL, "/")+r.CandidatePath == r.CandidateCanonicalURL
}

type Entry struct {
	Line   int
	Record Record
}

type ReviewedCandidate struct {
	ReviewID       string
	GateID         string
	BatchID        string
	UniqueIntentID string
	CandidatePath  string
	SourceMatrixID string
	Term           string
}

type ReviewIndex struct {
	BaseURL          string
	ReviewedByIntent map[string]ReviewedCandidate
}

type Issue struct {
	Code    string
	Message string
}

type Report struct {
	Issues []Issue
}

var idPattern = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)

func Validate(root string) Report {
	entries, report := LoadRecords(root)
	if !report.Passed() {
		return report
	}
	index, indexReport := BuildReviewIndex(root)
	issues := append([]Issue{}, indexReport.Issues...)
	if len(entries) != len(index.ReviewedByIntent) {
		issues = append(issues, Issue{Code: "batch_prepublication_count_mismatch", Message: fmt.Sprintf("gates=%d reviews=%d", len(entries), len(index.ReviewedByIntent))})
	}

	seen := make(map[string]int)
	for _, entry := range entries {
		recordReport := ValidateRecordAgainstReviewIndex(entry.Record, index)
		for _, issue := range recordReport.Issues {
			issue.Message = fmt.Sprintf("line=%d %s", entry.Line, issue.Message)
			issues = append(issues, issue)
		}
		if previous := seen[entry.Record.UniqueIntentID]; previous > 0 {
			issues = append(issues, Issue{Code: "batch_prepublication_duplicate_intent", Message: fmt.Sprintf("line=%d previous_line=%d id=%s", entry.Line, previous, entry.Record.UniqueIntentID)})
		}
		seen[entry.Record.UniqueIntentID] = entry.Line
	}
	for intentID := range index.ReviewedByIntent {
		if seen[intentID] == 0 {
			issues = append(issues, Issue{Code: "batch_prepublication_missing_reviewed_intent", Message: intentID})
		}
	}
	return Report{Issues: issues}
}

func BuildReviewIndex(root string) (ReviewIndex, Report) {
	reviewReport := batchcandidatereviews.Validate(root)
	issues := convertReviewIssues(reviewReport)
	repo, err := content.LoadRepository(root)
	if err != nil {
		issues = append(issues, Issue{Code: "batch_prepublication_site_config_unavailable", Message: err.Error()})
	}
	baseURL := "https://wikijuridica.com.br"
	if repo.BaseURL != "" {
		baseURL = repo.BaseURL
	}

	reviewEntries, loadReport := batchcandidatereviews.LoadRecords(root)
	issues = append(issues, convertReviewIssues(loadReport)...)
	index := ReviewIndex{
		BaseURL:          strings.TrimRight(baseURL, "/"),
		ReviewedByIntent: make(map[string]ReviewedCandidate),
	}
	for _, entry := range reviewEntries {
		record := entry.Record
		if record.ReviewStatus != "batch_candidate_review_blocked" || record.PublicationAllowed || record.RenderAllowed || record.SitemapAllowed || record.PublicPath != "" {
			continue
		}
		index.ReviewedByIntent[record.UniqueIntentID] = ReviewedCandidate{
			ReviewID:       record.ReviewID,
			GateID:         record.GateID,
			BatchID:        record.BatchID,
			UniqueIntentID: record.UniqueIntentID,
			CandidatePath:  record.CandidatePath,
			SourceMatrixID: record.SourceMatrixID,
			Term:           record.Term,
		}
	}
	return index, Report{Issues: issues}
}

func ValidateRecordAgainstReviewIndex(record Record, index ReviewIndex) Report {
	issues := make([]Issue, 0)
	for _, id := range []struct {
		code  string
		value string
	}{
		{"batch_prepublication_invalid_id", record.PrepublicationID},
		{"batch_prepublication_invalid_review_id", record.ReviewID},
		{"batch_prepublication_invalid_gate_id", record.GateID},
		{"batch_prepublication_invalid_batch_id", record.BatchID},
		{"batch_prepublication_invalid_intent", record.UniqueIntentID},
		{"batch_prepublication_invalid_source_matrix", record.SourceMatrixID},
	} {
		if id.value == "" || !idPattern.MatchString(id.value) {
			issues = append(issues, Issue{Code: id.code, Message: id.value})
		}
	}

	reviewed, ok := index.ReviewedByIntent[record.UniqueIntentID]
	if !ok {
		issues = append(issues, Issue{Code: "batch_prepublication_missing_review", Message: record.UniqueIntentID})
	} else {
		if record.ReviewID != reviewed.ReviewID {
			issues = append(issues, Issue{Code: "batch_prepublication_review_mismatch", Message: fmt.Sprintf("record=%s review=%s", record.ReviewID, reviewed.ReviewID)})
		}
		if record.GateID != reviewed.GateID {
			issues = append(issues, Issue{Code: "batch_prepublication_gate_mismatch", Message: fmt.Sprintf("record=%s review=%s", record.GateID, reviewed.GateID)})
		}
		if record.BatchID != reviewed.BatchID {
			issues = append(issues, Issue{Code: "batch_prepublication_batch_mismatch", Message: fmt.Sprintf("record=%s review=%s", record.BatchID, reviewed.BatchID)})
		}
		if record.CandidatePath != reviewed.CandidatePath {
			issues = append(issues, Issue{Code: "batch_prepublication_candidate_path_mismatch", Message: fmt.Sprintf("record=%s review=%s", record.CandidatePath, reviewed.CandidatePath)})
		}
		if record.SourceMatrixID != reviewed.SourceMatrixID {
			issues = append(issues, Issue{Code: "batch_prepublication_source_matrix_mismatch", Message: fmt.Sprintf("record=%s review=%s", record.SourceMatrixID, reviewed.SourceMatrixID)})
		}
		if record.Term != reviewed.Term {
			issues = append(issues, Issue{Code: "batch_prepublication_term_mismatch", Message: record.UniqueIntentID})
		}
	}
	if strings.Contains(record.CandidatePath, "://") || !router.IsCleanPublicPath(record.CandidatePath) || strings.ContainsAny(record.CandidatePath, "?#") {
		issues = append(issues, Issue{Code: "batch_prepublication_candidate_path_not_clean", Message: record.CandidatePath})
	}
	expectedCanonical := strings.TrimRight(index.BaseURL, "/") + record.CandidatePath
	if !seo.IsAbsoluteHTTPSURL(record.CandidateCanonicalURL) {
		issues = append(issues, Issue{Code: "batch_prepublication_canonical_not_https", Message: record.CandidateCanonicalURL})
	} else if record.CandidateCanonicalURL != expectedCanonical {
		issues = append(issues, Issue{Code: "batch_prepublication_canonical_mismatch", Message: record.CandidateCanonicalURL})
	}
	if record.CandidateRobots != "noindex,follow" {
		issues = append(issues, Issue{Code: "batch_prepublication_not_noindex", Message: record.CandidateRobots})
	}
	titleLen := len([]rune(record.CandidateTitle))
	if titleLen < seo.TitleMinCharacters {
		issues = append(issues, Issue{Code: "batch_prepublication_title_too_short", Message: fmt.Sprintf("%s:%d", record.UniqueIntentID, titleLen)})
	}
	if titleLen > seo.TitleMaxCharacters {
		issues = append(issues, Issue{Code: "batch_prepublication_title_too_long", Message: fmt.Sprintf("%s:%d", record.UniqueIntentID, titleLen)})
	}
	if looksTruncatedFocus(record.CandidateTitle) {
		issues = append(issues, Issue{Code: "batch_prepublication_title_truncated_focus", Message: record.UniqueIntentID})
	}
	metaLen := len([]rune(record.CandidateMetaDescription))
	if metaLen < seo.MetaDescriptionMinCharacters {
		issues = append(issues, Issue{Code: "batch_prepublication_meta_too_short", Message: fmt.Sprintf("%s:%d", record.UniqueIntentID, metaLen)})
	}
	if metaLen > seo.MetaDescriptionMaxCharacters {
		issues = append(issues, Issue{Code: "batch_prepublication_meta_too_long", Message: fmt.Sprintf("%s:%d", record.UniqueIntentID, metaLen)})
	}
	if looksTruncatedFocus(record.CandidateMetaDescription) {
		issues = append(issues, Issue{Code: "batch_prepublication_meta_truncated_focus", Message: record.UniqueIntentID})
	}
	if record.SourceSpecificityStatus != "matrix_audited_final_source_pending" {
		issues = append(issues, Issue{Code: "batch_prepublication_source_specificity_not_pending", Message: record.SourceSpecificityStatus})
	}
	if len(record.RemainingGates) < 3 {
		issues = append(issues, Issue{Code: "batch_prepublication_missing_remaining_gates", Message: record.UniqueIntentID})
	}
	if record.RenderAllowed {
		issues = append(issues, Issue{Code: "batch_prepublication_render_allowed", Message: record.UniqueIntentID})
	}
	if record.SitemapAllowed {
		issues = append(issues, Issue{Code: "batch_prepublication_sitemap_allowed", Message: record.UniqueIntentID})
	}
	if record.PublicationAllowed {
		issues = append(issues, Issue{Code: "batch_prepublication_publication_allowed", Message: record.UniqueIntentID})
	}
	if record.PublicPath != "" {
		issues = append(issues, Issue{Code: "batch_prepublication_has_public_path", Message: record.PublicPath})
	}
	if record.CheckedAt == "" {
		issues = append(issues, Issue{Code: "batch_prepublication_without_checked_at", Message: record.UniqueIntentID})
	}
	return Report{Issues: issues}
}

func LoadRecords(root string) ([]Entry, Report) {
	projectRoot, err := findProjectRoot(root)
	if err != nil {
		return nil, Report{Issues: []Issue{{Code: "project_root_not_found", Message: err.Error()}}}
	}
	path := filepath.Join(projectRoot, "data", "editorial", "batch_prepublication_gates.jsonl")
	file, err := os.Open(path)
	if err != nil {
		return nil, Report{Issues: []Issue{{Code: "batch_prepublication_gates_missing", Message: err.Error()}}}
	}
	defer file.Close()

	issues := make([]Issue, 0)
	entries := make([]Entry, 0)
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
			issues = append(issues, Issue{Code: "batch_prepublication_invalid_json", Message: fmt.Sprintf("line=%d %s", lineNumber, err.Error())})
			continue
		}
		entries = append(entries, Entry{Line: lineNumber, Record: record})
	}
	if err := scanner.Err(); err != nil {
		issues = append(issues, Issue{Code: "batch_prepublication_scan_failed", Message: err.Error()})
	}
	return entries, Report{Issues: issues}
}

func convertReviewIssues(report batchcandidatereviews.Report) []Issue {
	issues := make([]Issue, 0, len(report.Issues))
	for _, issue := range report.Issues {
		issues = append(issues, Issue{Code: "batch_prepublication_review_" + issue.Code, Message: issue.Message})
	}
	return issues
}

func looksTruncatedFocus(value string) bool {
	normalized := strings.ToLower(strings.TrimSpace(value))
	normalized = strings.TrimRight(normalized, ".:;- ")
	return strings.HasSuffix(normalized, "com foco em") || strings.HasSuffix(normalized, "com foco")
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

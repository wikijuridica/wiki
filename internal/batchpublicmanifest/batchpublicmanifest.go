package batchpublicmanifest

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"portaljuridico/internal/batchprepublication"
	"portaljuridico/internal/batchsourcespecificity"
	"portaljuridico/internal/content"
	"portaljuridico/internal/router"
	"portaljuridico/internal/seo"
)

const (
	SEOReviewPendingStatus = "public_manifest_blocked_seo_review_pending"
	SourceBlockedStatus    = "public_manifest_blocked_source_specificity"
	UsePolicy              = "reference_only_no_scraping_no_ingestion"
)

type Record struct {
	ManifestGateID           string   `json:"manifest_gate_id"`
	SourceResolutionID       string   `json:"source_resolution_id"`
	PrepublicationID         string   `json:"prepublication_id"`
	ReviewID                 string   `json:"review_id"`
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
	ManifestGateStatus       string   `json:"manifest_gate_status"`
	SelectedSourceURLs       []string `json:"selected_source_urls"`
	BlockingReason           string   `json:"blocking_reason"`
	NeededSourceDetail       string   `json:"needed_source_detail"`
	IndexPolicy              string   `json:"index_policy"`
	UsePolicy                string   `json:"use_policy"`
	SEOReviewRequired        bool     `json:"seo_review_required"`
	ContentDraftRequired     bool     `json:"content_draft_required"`
	ManifestAllowed          bool     `json:"manifest_allowed"`
	RenderAllowed            bool     `json:"render_allowed"`
	SitemapAllowed           bool     `json:"sitemap_allowed"`
	PublicationAllowed       bool     `json:"publication_allowed"`
	PublicPath               string   `json:"public_path"`
	CheckedAt                string   `json:"checked_at"`
}

type Entry struct {
	Line   int
	Record Record
}

type SourceResolution struct {
	ResolutionID            string
	PrepublicationID        string
	ReviewID                string
	BatchID                 string
	UniqueIntentID          string
	SourceMatrixID          string
	Term                    string
	CandidatePath           string
	CandidateCanonicalURL   string
	CandidateRobots         string
	SourceSpecificityStatus string
	SelectedSourceURLs      []string
	BlockingReason          string
	NeededSourceDetail      string
}

type PrepublicationSEO struct {
	CandidateTitle           string
	CandidateMetaDescription string
}

type ManifestIndex struct {
	BaseURL                string
	SourceByIntent         map[string]SourceResolution
	PrepublicationByIntent map[string]PrepublicationSEO
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
	index, indexReport := BuildManifestIndex(root)
	issues := append([]Issue{}, indexReport.Issues...)
	if len(entries) != len(index.SourceByIntent) {
		issues = append(issues, Issue{Code: "batch_public_manifest_count_mismatch", Message: fmt.Sprintf("manifest=%d source_resolutions=%d", len(entries), len(index.SourceByIntent))})
	}

	seenIntent := make(map[string]int)
	seenGate := make(map[string]int)
	for _, entry := range entries {
		recordReport := ValidateRecordAgainstManifestIndex(entry.Record, index)
		for _, issue := range recordReport.Issues {
			issue.Message = fmt.Sprintf("line=%d %s", entry.Line, issue.Message)
			issues = append(issues, issue)
		}
		if previous := seenIntent[entry.Record.UniqueIntentID]; previous > 0 {
			issues = append(issues, Issue{Code: "batch_public_manifest_duplicate_intent", Message: fmt.Sprintf("line=%d previous_line=%d id=%s", entry.Line, previous, entry.Record.UniqueIntentID)})
		}
		seenIntent[entry.Record.UniqueIntentID] = entry.Line
		if previous := seenGate[entry.Record.ManifestGateID]; previous > 0 {
			issues = append(issues, Issue{Code: "batch_public_manifest_duplicate_gate", Message: fmt.Sprintf("line=%d previous_line=%d id=%s", entry.Line, previous, entry.Record.ManifestGateID)})
		}
		seenGate[entry.Record.ManifestGateID] = entry.Line
	}
	for intentID := range index.SourceByIntent {
		if seenIntent[intentID] == 0 {
			issues = append(issues, Issue{Code: "batch_public_manifest_missing_source_resolution", Message: intentID})
		}
	}
	return Report{Issues: issues}
}

func BuildManifestIndex(root string) (ManifestIndex, Report) {
	sourceReport := batchsourcespecificity.Validate(root)
	issues := convertSourceIssues(sourceReport)
	sourceEntries, sourceLoadReport := batchsourcespecificity.LoadRecords(root)
	issues = append(issues, convertSourceIssues(sourceLoadReport)...)
	preEntries, preLoadReport := batchprepublication.LoadRecords(root)
	issues = append(issues, convertPrepublicationIssues(preLoadReport)...)
	repo, err := content.LoadRepository(root)
	if err != nil {
		issues = append(issues, Issue{Code: "batch_public_manifest_site_config_unavailable", Message: err.Error()})
	}
	baseURL := "https://wikijuridica.com.br"
	if repo.BaseURL != "" {
		baseURL = repo.BaseURL
	}

	index := ManifestIndex{
		BaseURL:                strings.TrimRight(baseURL, "/"),
		SourceByIntent:         make(map[string]SourceResolution),
		PrepublicationByIntent: make(map[string]PrepublicationSEO),
	}
	if index.BaseURL == "" {
		index.BaseURL = "https://wikijuridica.com.br"
	}
	for _, entry := range sourceEntries {
		record := entry.Record
		index.SourceByIntent[record.UniqueIntentID] = SourceResolution{
			ResolutionID:            record.ResolutionID,
			PrepublicationID:        record.PrepublicationID,
			ReviewID:                record.ReviewID,
			BatchID:                 record.BatchID,
			UniqueIntentID:          record.UniqueIntentID,
			SourceMatrixID:          record.SourceMatrixID,
			Term:                    record.Term,
			CandidatePath:           record.CandidatePath,
			CandidateCanonicalURL:   record.CandidateCanonicalURL,
			CandidateRobots:         record.CandidateRobots,
			SourceSpecificityStatus: record.SourceSpecificityStatus,
			SelectedSourceURLs:      append([]string{}, record.SelectedSourceURLs...),
			BlockingReason:          record.BlockingReason,
			NeededSourceDetail:      record.NeededSourceDetail,
		}
	}
	for _, entry := range preEntries {
		record := entry.Record
		index.PrepublicationByIntent[record.UniqueIntentID] = PrepublicationSEO{
			CandidateTitle:           record.CandidateTitle,
			CandidateMetaDescription: record.CandidateMetaDescription,
		}
	}
	return index, Report{Issues: issues}
}

func ValidateRecordAgainstManifestIndex(record Record, index ManifestIndex) Report {
	issues := make([]Issue, 0)
	for _, id := range []struct {
		code  string
		value string
	}{
		{"batch_public_manifest_invalid_id", record.ManifestGateID},
		{"batch_public_manifest_invalid_source_resolution_id", record.SourceResolutionID},
		{"batch_public_manifest_invalid_prepublication_id", record.PrepublicationID},
		{"batch_public_manifest_invalid_review_id", record.ReviewID},
		{"batch_public_manifest_invalid_batch_id", record.BatchID},
		{"batch_public_manifest_invalid_intent", record.UniqueIntentID},
		{"batch_public_manifest_invalid_source_matrix", record.SourceMatrixID},
	} {
		if id.value == "" || !idPattern.MatchString(id.value) {
			issues = append(issues, Issue{Code: id.code, Message: id.value})
		}
	}

	source, ok := index.SourceByIntent[record.UniqueIntentID]
	if !ok {
		issues = append(issues, Issue{Code: "batch_public_manifest_missing_source_resolution", Message: record.UniqueIntentID})
	} else {
		issues = append(issues, compareSource(record, source)...)
	}
	prepublication, ok := index.PrepublicationByIntent[record.UniqueIntentID]
	if !ok {
		issues = append(issues, Issue{Code: "batch_public_manifest_missing_prepublication", Message: record.UniqueIntentID})
	} else {
		if record.CandidateTitle != prepublication.CandidateTitle {
			issues = append(issues, Issue{Code: "batch_public_manifest_title_prepublication_mismatch", Message: record.UniqueIntentID})
		}
		if record.CandidateMetaDescription != prepublication.CandidateMetaDescription {
			issues = append(issues, Issue{Code: "batch_public_manifest_meta_prepublication_mismatch", Message: record.UniqueIntentID})
		}
	}

	if strings.Contains(record.CandidatePath, "://") || !router.IsCleanPublicPath(record.CandidatePath) || strings.ContainsAny(record.CandidatePath, "?#") {
		issues = append(issues, Issue{Code: "batch_public_manifest_candidate_path_not_clean", Message: record.CandidatePath})
	}
	expectedCanonical := strings.TrimRight(index.BaseURL, "/") + record.CandidatePath
	if !seo.IsAbsoluteHTTPSURL(record.CandidateCanonicalURL) {
		issues = append(issues, Issue{Code: "batch_public_manifest_canonical_not_https", Message: record.CandidateCanonicalURL})
	} else if record.CandidateCanonicalURL != expectedCanonical {
		issues = append(issues, Issue{Code: "batch_public_manifest_canonical_mismatch", Message: record.CandidateCanonicalURL})
	}
	if record.CandidateRobots != "noindex,follow" {
		issues = append(issues, Issue{Code: "batch_public_manifest_not_noindex", Message: record.CandidateRobots})
	}
	titleLen := len([]rune(record.CandidateTitle))
	if titleLen < seo.TitleMinCharacters {
		issues = append(issues, Issue{Code: "batch_public_manifest_title_too_short", Message: fmt.Sprintf("%s:%d", record.UniqueIntentID, titleLen)})
	}
	if titleLen > seo.TitleMaxCharacters {
		issues = append(issues, Issue{Code: "batch_public_manifest_title_too_long", Message: fmt.Sprintf("%s:%d", record.UniqueIntentID, titleLen)})
	}
	if looksTruncatedFocus(record.CandidateTitle) {
		issues = append(issues, Issue{Code: "batch_public_manifest_title_truncated_focus", Message: record.UniqueIntentID})
	}
	metaLen := len([]rune(record.CandidateMetaDescription))
	if metaLen < seo.MetaDescriptionMinCharacters {
		issues = append(issues, Issue{Code: "batch_public_manifest_meta_too_short", Message: fmt.Sprintf("%s:%d", record.UniqueIntentID, metaLen)})
	}
	if metaLen > seo.MetaDescriptionMaxCharacters {
		issues = append(issues, Issue{Code: "batch_public_manifest_meta_too_long", Message: fmt.Sprintf("%s:%d", record.UniqueIntentID, metaLen)})
	}
	if looksTruncatedFocus(record.CandidateMetaDescription) {
		issues = append(issues, Issue{Code: "batch_public_manifest_meta_truncated_focus", Message: record.UniqueIntentID})
	}
	if record.ManifestGateStatus != SEOReviewPendingStatus && record.ManifestGateStatus != SourceBlockedStatus {
		issues = append(issues, Issue{Code: "batch_public_manifest_invalid_status", Message: record.ManifestGateStatus})
	}
	if record.ManifestGateStatus == SEOReviewPendingStatus {
		if record.SourceSpecificityStatus != batchsourcespecificity.LockedStatus {
			issues = append(issues, Issue{Code: "batch_public_manifest_source_not_locked_for_seo", Message: record.UniqueIntentID})
		}
		if !record.SEOReviewRequired {
			issues = append(issues, Issue{Code: "batch_public_manifest_missing_seo_review", Message: record.UniqueIntentID})
		}
		if !record.ContentDraftRequired {
			issues = append(issues, Issue{Code: "batch_public_manifest_missing_content_draft", Message: record.UniqueIntentID})
		}
	}
	if record.ManifestGateStatus == SourceBlockedStatus {
		if record.SourceSpecificityStatus != batchsourcespecificity.BlockedStatus {
			issues = append(issues, Issue{Code: "batch_public_manifest_locked_source_still_blocked", Message: record.UniqueIntentID})
		}
		if strings.TrimSpace(record.BlockingReason) == "" {
			issues = append(issues, Issue{Code: "batch_public_manifest_missing_blocking_reason", Message: record.UniqueIntentID})
		}
		if strings.TrimSpace(record.NeededSourceDetail) == "" {
			issues = append(issues, Issue{Code: "batch_public_manifest_missing_needed_source_detail", Message: record.UniqueIntentID})
		}
		if record.SEOReviewRequired || record.ContentDraftRequired {
			issues = append(issues, Issue{Code: "batch_public_manifest_review_before_source", Message: record.UniqueIntentID})
		}
	}
	if record.IndexPolicy != "noindex" {
		issues = append(issues, Issue{Code: "batch_public_manifest_invalid_index_policy", Message: record.IndexPolicy})
	}
	if record.UsePolicy != UsePolicy {
		issues = append(issues, Issue{Code: "batch_public_manifest_invalid_use_policy", Message: record.UsePolicy})
	}
	if record.ManifestAllowed {
		issues = append(issues, Issue{Code: "batch_public_manifest_manifest_allowed", Message: record.UniqueIntentID})
	}
	if record.RenderAllowed {
		issues = append(issues, Issue{Code: "batch_public_manifest_render_allowed", Message: record.UniqueIntentID})
	}
	if record.SitemapAllowed {
		issues = append(issues, Issue{Code: "batch_public_manifest_sitemap_allowed", Message: record.UniqueIntentID})
	}
	if record.PublicationAllowed {
		issues = append(issues, Issue{Code: "batch_public_manifest_publication_allowed", Message: record.UniqueIntentID})
	}
	if record.PublicPath != "" {
		issues = append(issues, Issue{Code: "batch_public_manifest_has_public_path", Message: record.PublicPath})
	}
	if record.CheckedAt == "" {
		issues = append(issues, Issue{Code: "batch_public_manifest_without_checked_at", Message: record.UniqueIntentID})
	}
	return Report{Issues: issues}
}

func LoadRecords(root string) ([]Entry, Report) {
	projectRoot, err := findProjectRoot(root)
	if err != nil {
		return nil, Report{Issues: []Issue{{Code: "project_root_not_found", Message: err.Error()}}}
	}
	path := filepath.Join(projectRoot, "data", "editorial", "batch_public_manifest_gates.jsonl")
	file, err := os.Open(path)
	if err != nil {
		return nil, Report{Issues: []Issue{{Code: "batch_public_manifest_missing", Message: err.Error()}}}
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
			issues = append(issues, Issue{Code: "batch_public_manifest_invalid_json", Message: fmt.Sprintf("line=%d %s", lineNumber, err.Error())})
			continue
		}
		entries = append(entries, Entry{Line: lineNumber, Record: record})
	}
	if err := scanner.Err(); err != nil {
		issues = append(issues, Issue{Code: "batch_public_manifest_scan_failed", Message: err.Error()})
	}
	return entries, Report{Issues: issues}
}

func compareSource(record Record, source SourceResolution) []Issue {
	issues := make([]Issue, 0)
	if record.SourceResolutionID != source.ResolutionID {
		issues = append(issues, Issue{Code: "batch_public_manifest_source_resolution_mismatch", Message: record.UniqueIntentID})
	}
	if record.PrepublicationID != source.PrepublicationID {
		issues = append(issues, Issue{Code: "batch_public_manifest_prepublication_mismatch", Message: record.UniqueIntentID})
	}
	if record.ReviewID != source.ReviewID {
		issues = append(issues, Issue{Code: "batch_public_manifest_review_mismatch", Message: record.UniqueIntentID})
	}
	if record.BatchID != source.BatchID {
		issues = append(issues, Issue{Code: "batch_public_manifest_batch_mismatch", Message: record.UniqueIntentID})
	}
	if record.SourceMatrixID != source.SourceMatrixID {
		issues = append(issues, Issue{Code: "batch_public_manifest_source_matrix_mismatch", Message: record.UniqueIntentID})
	}
	if record.Term != source.Term {
		issues = append(issues, Issue{Code: "batch_public_manifest_term_mismatch", Message: record.UniqueIntentID})
	}
	if record.CandidatePath != source.CandidatePath {
		issues = append(issues, Issue{Code: "batch_public_manifest_candidate_path_mismatch", Message: record.UniqueIntentID})
	}
	if record.CandidateCanonicalURL != source.CandidateCanonicalURL {
		issues = append(issues, Issue{Code: "batch_public_manifest_canonical_source_mismatch", Message: record.UniqueIntentID})
	}
	if record.CandidateRobots != source.CandidateRobots {
		issues = append(issues, Issue{Code: "batch_public_manifest_robots_source_mismatch", Message: record.UniqueIntentID})
	}
	if record.SourceSpecificityStatus != source.SourceSpecificityStatus {
		issues = append(issues, Issue{Code: "batch_public_manifest_source_status_mismatch", Message: record.UniqueIntentID})
	}
	if !sameStringSet(record.SelectedSourceURLs, source.SelectedSourceURLs) {
		issues = append(issues, Issue{Code: "batch_public_manifest_source_urls_mismatch", Message: record.UniqueIntentID})
	}
	if source.SourceSpecificityStatus == batchsourcespecificity.BlockedStatus {
		if record.BlockingReason != source.BlockingReason {
			issues = append(issues, Issue{Code: "batch_public_manifest_blocking_reason_mismatch", Message: record.UniqueIntentID})
		}
		if record.NeededSourceDetail != source.NeededSourceDetail {
			issues = append(issues, Issue{Code: "batch_public_manifest_needed_source_detail_mismatch", Message: record.UniqueIntentID})
		}
	}
	return issues
}

func sameStringSet(left []string, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	leftCopy := append([]string{}, left...)
	rightCopy := append([]string{}, right...)
	sort.Strings(leftCopy)
	sort.Strings(rightCopy)
	for index := range leftCopy {
		if leftCopy[index] != rightCopy[index] {
			return false
		}
	}
	return true
}

func convertSourceIssues(report batchsourcespecificity.Report) []Issue {
	issues := make([]Issue, 0, len(report.Issues))
	for _, issue := range report.Issues {
		issues = append(issues, Issue{Code: "batch_public_manifest_source_" + issue.Code, Message: issue.Message})
	}
	return issues
}

func convertPrepublicationIssues(report batchprepublication.Report) []Issue {
	issues := make([]Issue, 0, len(report.Issues))
	for _, issue := range report.Issues {
		issues = append(issues, Issue{Code: "batch_public_manifest_prepublication_" + issue.Code, Message: issue.Message})
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

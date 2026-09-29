package batchfinaldrafts

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"portaljuridico/internal/batchpublicmanifest"
	"portaljuridico/internal/content"
	"portaljuridico/internal/humanscore"
	"portaljuridico/internal/quality"
	"portaljuridico/internal/router"
	"portaljuridico/internal/seo"
)

const DraftStatus = "batch_final_authorial_draft_blocked"

type Record struct {
	DraftID                  string   `json:"draft_id"`
	ManifestGateID           string   `json:"manifest_gate_id"`
	UniqueIntentID           string   `json:"unique_intent_id"`
	BatchID                  string   `json:"batch_id"`
	SourceMatrixID           string   `json:"source_matrix_id"`
	Term                     string   `json:"term"`
	CandidatePath            string   `json:"candidate_path"`
	CandidateCanonicalURL    string   `json:"candidate_canonical_url"`
	CandidateTitle           string   `json:"candidate_title"`
	CandidateMetaDescription string   `json:"candidate_meta_description"`
	SelectedSourceURLs       []string `json:"selected_source_urls"`
	DraftStatus              string   `json:"draft_status"`
	Language                 string   `json:"language"`
	Opening                  string   `json:"opening"`
	SourceUse                string   `json:"source_use"`
	DocumentGuidance         string   `json:"document_guidance"`
	DigitalTriage            string   `json:"digital_triage"`
	CTAContextMessage        string   `json:"cta_context_message"`
	InformationalNotice      string   `json:"informational_notice"`
	HumanScore               int      `json:"human_score"`
	AILikeScore              int      `json:"ai_like_score"`
	IndexPolicy              string   `json:"index_policy"`
	RenderAllowed            bool     `json:"render_allowed"`
	SitemapAllowed           bool     `json:"sitemap_allowed"`
	PublicationAllowed       bool     `json:"publication_allowed"`
	PublicPath               string   `json:"public_path"`
	CheckedAt                string   `json:"checked_at"`
}

func (r Record) FullText() string {
	parts := []string{
		r.Term,
		r.Opening,
		r.SourceUse,
		r.DocumentGuidance,
		r.DigitalTriage,
		r.CTAContextMessage,
		r.InformationalNotice,
	}
	return strings.Join(parts, " ")
}

func (r Record) HasContextualCTA() bool {
	message := strings.ToLower(r.CTAContextMessage)
	return strings.Contains(message, "origem:") && strings.Contains(message, "intent:") && strings.Contains(message, "documentos:")
}

func (r Record) HasInformationalNotice() bool {
	notice := strings.ToLower(r.InformationalNotice)
	return strings.Contains(notice, "não substitui consulta jurídica individual") || strings.Contains(notice, "nao substitui consulta juridica individual")
}

type Entry struct {
	Line   int
	Record Record
}

type EligibleManifest struct {
	ManifestGateID           string
	UniqueIntentID           string
	BatchID                  string
	SourceMatrixID           string
	Term                     string
	CandidatePath            string
	CandidateCanonicalURL    string
	CandidateTitle           string
	CandidateMetaDescription string
	SelectedSourceURLs       []string
}

type ManifestIndex struct {
	BaseURL          string
	EligibleByIntent map[string]EligibleManifest
	BlockedByIntent  map[string]bool
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
	if len(entries) != len(index.EligibleByIntent) {
		issues = append(issues, Issue{Code: "batch_final_draft_count_mismatch", Message: fmt.Sprintf("drafts=%d eligible=%d", len(entries), len(index.EligibleByIntent))})
	}

	seen := make(map[string]int)
	for _, entry := range entries {
		recordReport := ValidateRecordAgainstManifestIndex(entry.Record, index)
		for _, issue := range recordReport.Issues {
			issue.Message = fmt.Sprintf("line=%d %s", entry.Line, issue.Message)
			issues = append(issues, issue)
		}
		if previous := seen[entry.Record.UniqueIntentID]; previous > 0 {
			issues = append(issues, Issue{Code: "batch_final_draft_duplicate_intent", Message: fmt.Sprintf("line=%d previous_line=%d id=%s", entry.Line, previous, entry.Record.UniqueIntentID)})
		}
		seen[entry.Record.UniqueIntentID] = entry.Line
	}
	for intentID := range index.EligibleByIntent {
		if seen[intentID] == 0 {
			issues = append(issues, Issue{Code: "batch_final_draft_missing_eligible_manifest", Message: intentID})
		}
	}
	return Report{Issues: issues}
}

func BuildManifestIndex(root string) (ManifestIndex, Report) {
	manifestReport := batchpublicmanifest.Validate(root)
	issues := convertManifestIssues(manifestReport)
	manifestEntries, manifestLoadReport := batchpublicmanifest.LoadRecords(root)
	issues = append(issues, convertManifestIssues(manifestLoadReport)...)
	repo, err := content.LoadRepository(root)
	if err != nil {
		issues = append(issues, Issue{Code: "batch_final_draft_site_config_unavailable", Message: err.Error()})
	}
	baseURL := "https://wikijuridica.com.br"
	if repo.BaseURL != "" {
		baseURL = repo.BaseURL
	}

	index := ManifestIndex{
		BaseURL:          strings.TrimRight(baseURL, "/"),
		EligibleByIntent: make(map[string]EligibleManifest),
		BlockedByIntent:  make(map[string]bool),
	}
	if index.BaseURL == "" {
		index.BaseURL = "https://wikijuridica.com.br"
	}
	for _, entry := range manifestEntries {
		record := entry.Record
		if record.ManifestGateStatus == batchpublicmanifest.SEOReviewPendingStatus {
			index.EligibleByIntent[record.UniqueIntentID] = EligibleManifest{
				ManifestGateID:           record.ManifestGateID,
				UniqueIntentID:           record.UniqueIntentID,
				BatchID:                  record.BatchID,
				SourceMatrixID:           record.SourceMatrixID,
				Term:                     record.Term,
				CandidatePath:            record.CandidatePath,
				CandidateCanonicalURL:    record.CandidateCanonicalURL,
				CandidateTitle:           record.CandidateTitle,
				CandidateMetaDescription: record.CandidateMetaDescription,
				SelectedSourceURLs:       append([]string{}, record.SelectedSourceURLs...),
			}
		}
		if record.ManifestGateStatus == batchpublicmanifest.SourceBlockedStatus {
			index.BlockedByIntent[record.UniqueIntentID] = true
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
		{"batch_final_draft_invalid_id", record.DraftID},
		{"batch_final_draft_invalid_manifest_gate", record.ManifestGateID},
		{"batch_final_draft_invalid_intent", record.UniqueIntentID},
		{"batch_final_draft_invalid_batch", record.BatchID},
		{"batch_final_draft_invalid_source_matrix", record.SourceMatrixID},
	} {
		if id.value == "" || !idPattern.MatchString(id.value) {
			issues = append(issues, Issue{Code: id.code, Message: id.value})
		}
	}

	if index.BlockedByIntent[record.UniqueIntentID] {
		issues = append(issues, Issue{Code: "batch_final_draft_source_blocked", Message: record.UniqueIntentID})
	}
	eligible, ok := index.EligibleByIntent[record.UniqueIntentID]
	if !ok {
		issues = append(issues, Issue{Code: "batch_final_draft_missing_eligible_manifest", Message: record.UniqueIntentID})
	} else {
		issues = append(issues, compareManifest(record, eligible)...)
	}

	if strings.Contains(record.CandidatePath, "://") || !router.IsCleanPublicPath(record.CandidatePath) || strings.ContainsAny(record.CandidatePath, "?#") {
		issues = append(issues, Issue{Code: "batch_final_draft_candidate_path_not_clean", Message: record.CandidatePath})
	}
	expectedCanonical := strings.TrimRight(index.BaseURL, "/") + record.CandidatePath
	if !seo.IsAbsoluteHTTPSURL(record.CandidateCanonicalURL) {
		issues = append(issues, Issue{Code: "batch_final_draft_canonical_not_https", Message: record.CandidateCanonicalURL})
	} else if record.CandidateCanonicalURL != expectedCanonical {
		issues = append(issues, Issue{Code: "batch_final_draft_canonical_mismatch", Message: record.CandidateCanonicalURL})
	}
	titleLen := len([]rune(record.CandidateTitle))
	if titleLen < seo.TitleMinCharacters {
		issues = append(issues, Issue{Code: "batch_final_draft_title_too_short", Message: fmt.Sprintf("%s:%d", record.UniqueIntentID, titleLen)})
	}
	if titleLen > seo.TitleMaxCharacters {
		issues = append(issues, Issue{Code: "batch_final_draft_title_too_long", Message: fmt.Sprintf("%s:%d", record.UniqueIntentID, titleLen)})
	}
	if looksTruncatedFocus(record.CandidateTitle) {
		issues = append(issues, Issue{Code: "batch_final_draft_title_truncated_focus", Message: record.UniqueIntentID})
	}
	metaLen := len([]rune(record.CandidateMetaDescription))
	if metaLen < seo.MetaDescriptionMinCharacters {
		issues = append(issues, Issue{Code: "batch_final_draft_meta_too_short", Message: fmt.Sprintf("%s:%d", record.UniqueIntentID, metaLen)})
	}
	if metaLen > seo.MetaDescriptionMaxCharacters {
		issues = append(issues, Issue{Code: "batch_final_draft_meta_too_long", Message: fmt.Sprintf("%s:%d", record.UniqueIntentID, metaLen)})
	}
	if looksTruncatedFocus(record.CandidateMetaDescription) {
		issues = append(issues, Issue{Code: "batch_final_draft_meta_truncated_focus", Message: record.UniqueIntentID})
	}
	if record.DraftStatus != DraftStatus {
		issues = append(issues, Issue{Code: "batch_final_draft_invalid_status", Message: record.DraftStatus})
	}
	if record.Language != "pt-BR" {
		issues = append(issues, Issue{Code: "batch_final_draft_not_ptbr", Message: record.UniqueIntentID})
	}
	score := humanscore.ScoreText(record.FullText())
	if record.HumanScore < 85 || score.HumanScore < 85 || record.AILikeScore > 20 || score.AILikeScore > 20 || len(score.BlockingIssues) > 0 {
		issues = append(issues, Issue{Code: "batch_final_draft_text_failed_human_score", Message: fmt.Sprintf("%s record=%d/%d computed=%d/%d issues=%s", record.UniqueIntentID, record.HumanScore, record.AILikeScore, score.HumanScore, score.AILikeScore, strings.Join(score.Codes(), ","))})
	}
	analysis := quality.AnalyzeText(record.FullText())
	if !analysis.Passed() {
		issues = append(issues, Issue{Code: "batch_final_draft_text_failed_quality", Message: record.UniqueIntentID + ":" + strings.Join(analysis.Messages(), ",")})
	}
	if !record.HasContextualCTA() {
		issues = append(issues, Issue{Code: "batch_final_draft_cta_not_contextual", Message: record.UniqueIntentID})
	}
	if !record.HasInformationalNotice() {
		issues = append(issues, Issue{Code: "batch_final_draft_missing_notice", Message: record.UniqueIntentID})
	}
	if record.IndexPolicy != "noindex" {
		issues = append(issues, Issue{Code: "batch_final_draft_invalid_index_policy", Message: record.IndexPolicy})
	}
	if record.RenderAllowed {
		issues = append(issues, Issue{Code: "batch_final_draft_render_allowed", Message: record.UniqueIntentID})
	}
	if record.SitemapAllowed {
		issues = append(issues, Issue{Code: "batch_final_draft_sitemap_allowed", Message: record.UniqueIntentID})
	}
	if record.PublicationAllowed {
		issues = append(issues, Issue{Code: "batch_final_draft_publication_allowed", Message: record.UniqueIntentID})
	}
	if record.PublicPath != "" {
		issues = append(issues, Issue{Code: "batch_final_draft_has_public_path", Message: record.PublicPath})
	}
	if record.CheckedAt == "" {
		issues = append(issues, Issue{Code: "batch_final_draft_without_checked_at", Message: record.UniqueIntentID})
	}
	return Report{Issues: issues}
}

func LoadRecords(root string) ([]Entry, Report) {
	projectRoot, err := findProjectRoot(root)
	if err != nil {
		return nil, Report{Issues: []Issue{{Code: "project_root_not_found", Message: err.Error()}}}
	}
	path := filepath.Join(projectRoot, "data", "editorial", "batch_final_authorial_drafts.jsonl")
	file, err := os.Open(path)
	if err != nil {
		return nil, Report{Issues: []Issue{{Code: "batch_final_drafts_missing", Message: err.Error()}}}
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
			issues = append(issues, Issue{Code: "batch_final_draft_invalid_json", Message: fmt.Sprintf("line=%d %s", lineNumber, err.Error())})
			continue
		}
		entries = append(entries, Entry{Line: lineNumber, Record: record})
	}
	if err := scanner.Err(); err != nil {
		issues = append(issues, Issue{Code: "batch_final_draft_scan_failed", Message: err.Error()})
	}
	return entries, Report{Issues: issues}
}

func compareManifest(record Record, manifest EligibleManifest) []Issue {
	issues := make([]Issue, 0)
	if record.ManifestGateID != manifest.ManifestGateID {
		issues = append(issues, Issue{Code: "batch_final_draft_manifest_mismatch", Message: record.UniqueIntentID})
	}
	if record.BatchID != manifest.BatchID {
		issues = append(issues, Issue{Code: "batch_final_draft_batch_mismatch", Message: record.UniqueIntentID})
	}
	if record.SourceMatrixID != manifest.SourceMatrixID {
		issues = append(issues, Issue{Code: "batch_final_draft_source_matrix_mismatch", Message: record.UniqueIntentID})
	}
	if record.Term != manifest.Term {
		issues = append(issues, Issue{Code: "batch_final_draft_term_mismatch", Message: record.UniqueIntentID})
	}
	if record.CandidatePath != manifest.CandidatePath {
		issues = append(issues, Issue{Code: "batch_final_draft_candidate_path_mismatch", Message: record.UniqueIntentID})
	}
	if record.CandidateCanonicalURL != manifest.CandidateCanonicalURL {
		issues = append(issues, Issue{Code: "batch_final_draft_canonical_manifest_mismatch", Message: record.UniqueIntentID})
	}
	if record.CandidateTitle != manifest.CandidateTitle {
		issues = append(issues, Issue{Code: "batch_final_draft_title_manifest_mismatch", Message: record.UniqueIntentID})
	}
	if record.CandidateMetaDescription != manifest.CandidateMetaDescription {
		issues = append(issues, Issue{Code: "batch_final_draft_meta_manifest_mismatch", Message: record.UniqueIntentID})
	}
	if !sameStringSet(record.SelectedSourceURLs, manifest.SelectedSourceURLs) {
		issues = append(issues, Issue{Code: "batch_final_draft_source_urls_mismatch", Message: record.UniqueIntentID})
	}
	return issues
}

func looksTruncatedFocus(value string) bool {
	normalized := strings.ToLower(strings.TrimSpace(value))
	normalized = strings.TrimRight(normalized, ".:;- ")
	return strings.HasSuffix(normalized, "com foco em") || strings.HasSuffix(normalized, "com foco")
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

func convertManifestIssues(report batchpublicmanifest.Report) []Issue {
	issues := make([]Issue, 0, len(report.Issues))
	for _, issue := range report.Issues {
		issues = append(issues, Issue{Code: "batch_final_draft_manifest_" + issue.Code, Message: issue.Message})
	}
	return issues
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

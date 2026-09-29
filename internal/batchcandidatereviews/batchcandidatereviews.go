package batchcandidatereviews

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"portaljuridico/internal/batchcandidategates"
	"portaljuridico/internal/batchdrafts"
	"portaljuridico/internal/batchsourceaudit"
	"portaljuridico/internal/router"
)

type Record struct {
	ReviewID               string   `json:"review_id"`
	GateID                 string   `json:"gate_id"`
	BatchID                string   `json:"batch_id"`
	UniqueIntentID         string   `json:"unique_intent_id"`
	CandidatePath          string   `json:"candidate_path"`
	BaseURLMode            string   `json:"base_url_mode"`
	OfficialURLLocked      bool     `json:"official_url_locked"`
	ReviewStatus           string   `json:"review_status"`
	Language               string   `json:"language"`
	LegalArea              string   `json:"legal_area"`
	Term                   string   `json:"term"`
	SourceMatrixID         string   `json:"source_matrix_id"`
	ReviewerRole           string   `json:"reviewer_role"`
	LegalReviewNotes       []string `json:"legal_review_notes"`
	RequiredFixes          []string `json:"required_fixes"`
	CTADraft               string   `json:"cta_draft"`
	CTAContextMessage      string   `json:"cta_context_message"`
	CTAStatus              string   `json:"cta_status"`
	PublicationBlockReason string   `json:"publication_block_reason"`
	RenderAllowed          bool     `json:"render_allowed"`
	SitemapAllowed         bool     `json:"sitemap_allowed"`
	PublicationAllowed     bool     `json:"publication_allowed"`
	PublicPath             string   `json:"public_path"`
	CheckedAt              string   `json:"checked_at"`
}

func (r Record) CTAContextMessageContains(token string) bool {
	return strings.Contains(r.CTAContextMessage, token)
}

type Entry struct {
	Line   int
	Record Record
}

type SelectedCandidate struct {
	GateID         string
	BatchID        string
	UniqueIntentID string
	CandidatePath  string
	SourceMatrixID string
	Draft          batchdrafts.Record
}

type CandidateIndex struct {
	BaseURLMode       string
	OfficialURLLocked bool
	SelectedByIntent  map[string]SelectedCandidate
	AuditedMatrixIDs  map[string]bool
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
	index, indexReport := BuildCandidateIndex(root)
	issues := append([]Issue{}, indexReport.Issues...)
	if len(entries) != len(index.SelectedByIntent) {
		issues = append(issues, Issue{Code: "batch_candidate_review_count_mismatch", Message: fmt.Sprintf("reviews=%d selected=%d", len(entries), len(index.SelectedByIntent))})
	}

	seen := make(map[string]int)
	for _, entry := range entries {
		recordReport := ValidateRecordAgainstCandidateIndex(entry.Record, index)
		for _, issue := range recordReport.Issues {
			issue.Message = fmt.Sprintf("line=%d %s", entry.Line, issue.Message)
			issues = append(issues, issue)
		}
		if previous := seen[entry.Record.UniqueIntentID]; previous > 0 {
			issues = append(issues, Issue{Code: "batch_candidate_review_duplicate_intent", Message: fmt.Sprintf("line=%d previous_line=%d id=%s", entry.Line, previous, entry.Record.UniqueIntentID)})
		}
		seen[entry.Record.UniqueIntentID] = entry.Line
	}
	for intentID := range index.SelectedByIntent {
		if seen[intentID] == 0 {
			issues = append(issues, Issue{Code: "batch_candidate_review_missing_selected_intent", Message: intentID})
		}
	}
	return Report{Issues: issues}
}

func BuildCandidateIndex(root string) (CandidateIndex, Report) {
	archiveIndex, archiveReport := batchcandidategates.BuildArchiveIndex(root)
	issues := convertGateIssues(archiveReport)

	gateEntries, gatesReport := batchcandidategates.LoadRecords(root)
	issues = append(issues, convertGateIssues(gatesReport)...)
	auditEntries, auditReport := batchsourceaudit.LoadRecords(root)
	issues = append(issues, convertSourceAuditIssues(auditReport)...)

	index := CandidateIndex{
		BaseURLMode:       archiveIndex.BaseURLMode,
		OfficialURLLocked: archiveIndex.OfficialURLLocked,
		SelectedByIntent:  make(map[string]SelectedCandidate),
		AuditedMatrixIDs:  make(map[string]bool),
	}
	for _, auditEntry := range auditEntries {
		if !batchsourceaudit.ValidateRecord(auditEntry.Record).Passed() {
			continue
		}
		for _, matrixID := range auditEntry.Record.MatrixIDs {
			index.AuditedMatrixIDs[matrixID] = true
		}
	}
	for _, gateEntry := range gateEntries {
		if !batchcandidategates.ValidateRecordAgainstArchive(gateEntry.Record, archiveIndex).Passed() {
			continue
		}
		for _, intentID := range gateEntry.Record.SelectedUniqueIntentIDs {
			draft := archiveIndex.ByIntent[intentID]
			index.SelectedByIntent[intentID] = SelectedCandidate{
				GateID:         gateEntry.Record.GateID,
				BatchID:        gateEntry.Record.BatchID,
				UniqueIntentID: intentID,
				CandidatePath:  gateEntry.Record.CandidatePathPrefix + intentID + "/",
				SourceMatrixID: draft.SourceMatrixID,
				Draft:          draft,
			}
		}
	}
	return index, Report{Issues: issues}
}

func ValidateRecordAgainstCandidateIndex(record Record, index CandidateIndex) Report {
	issues := make([]Issue, 0)
	if record.ReviewID == "" || !idPattern.MatchString(record.ReviewID) {
		issues = append(issues, Issue{Code: "batch_candidate_review_invalid_id", Message: record.ReviewID})
	}
	if record.GateID == "" || !idPattern.MatchString(record.GateID) {
		issues = append(issues, Issue{Code: "batch_candidate_review_invalid_gate_id", Message: record.GateID})
	}
	if record.BatchID == "" || !idPattern.MatchString(record.BatchID) {
		issues = append(issues, Issue{Code: "batch_candidate_review_invalid_batch_id", Message: record.BatchID})
	}
	if record.UniqueIntentID == "" || !idPattern.MatchString(record.UniqueIntentID) {
		issues = append(issues, Issue{Code: "batch_candidate_review_invalid_intent", Message: record.UniqueIntentID})
	}
	selected, ok := index.SelectedByIntent[record.UniqueIntentID]
	if !ok {
		issues = append(issues, Issue{Code: "batch_candidate_review_unknown_selected_intent", Message: record.UniqueIntentID})
	} else {
		if record.GateID != selected.GateID {
			issues = append(issues, Issue{Code: "batch_candidate_review_gate_mismatch", Message: fmt.Sprintf("record=%s selected=%s", record.GateID, selected.GateID)})
		}
		if record.BatchID != selected.BatchID {
			issues = append(issues, Issue{Code: "batch_candidate_review_batch_mismatch", Message: fmt.Sprintf("record=%s selected=%s", record.BatchID, selected.BatchID)})
		}
		if record.CandidatePath != selected.CandidatePath {
			issues = append(issues, Issue{Code: "batch_candidate_review_candidate_path_mismatch", Message: fmt.Sprintf("record=%s selected=%s", record.CandidatePath, selected.CandidatePath)})
		}
		if record.SourceMatrixID != selected.SourceMatrixID {
			issues = append(issues, Issue{Code: "batch_candidate_review_source_matrix_mismatch", Message: fmt.Sprintf("record=%s selected=%s", record.SourceMatrixID, selected.SourceMatrixID)})
		}
		if record.LegalArea != selected.Draft.LegalArea || record.Term != selected.Draft.Term {
			issues = append(issues, Issue{Code: "batch_candidate_review_draft_identity_mismatch", Message: record.UniqueIntentID})
		}
	}
	if strings.Contains(record.CandidatePath, "://") {
		issues = append(issues, Issue{Code: "batch_candidate_review_path_contains_domain", Message: record.CandidatePath})
	} else if !router.IsCleanPublicPath(record.CandidatePath) {
		issues = append(issues, Issue{Code: "batch_candidate_review_candidate_path_not_clean", Message: record.CandidatePath})
	}
	if index.BaseURLMode != "" && record.BaseURLMode != index.BaseURLMode {
		issues = append(issues, Issue{Code: "batch_candidate_review_url_config_mismatch", Message: fmt.Sprintf("mode record=%s site=%s", record.BaseURLMode, index.BaseURLMode)})
	}
	if record.OfficialURLLocked != index.OfficialURLLocked {
		issues = append(issues, Issue{Code: "batch_candidate_review_url_config_mismatch", Message: fmt.Sprintf("locked record=%t site=%t", record.OfficialURLLocked, index.OfficialURLLocked)})
	}
	if record.ReviewStatus != "batch_candidate_review_blocked" {
		issues = append(issues, Issue{Code: "batch_candidate_review_invalid_status", Message: record.ReviewStatus})
	}
	if record.Language != "pt-BR" {
		issues = append(issues, Issue{Code: "batch_candidate_review_not_ptbr", Message: record.UniqueIntentID})
	}
	if record.ReviewerRole != "juridico_editorial_lab" {
		issues = append(issues, Issue{Code: "batch_candidate_review_invalid_reviewer_role", Message: record.ReviewerRole})
	}
	if record.SourceMatrixID == "" || !idPattern.MatchString(record.SourceMatrixID) {
		issues = append(issues, Issue{Code: "batch_candidate_review_invalid_source_matrix", Message: record.SourceMatrixID})
	} else if !index.AuditedMatrixIDs[record.SourceMatrixID] {
		issues = append(issues, Issue{Code: "batch_candidate_review_source_matrix_not_audited", Message: record.SourceMatrixID})
	}
	if len(record.LegalReviewNotes) < 3 {
		issues = append(issues, Issue{Code: "batch_candidate_review_too_few_notes", Message: record.UniqueIntentID})
	}
	for _, note := range record.LegalReviewNotes {
		if len(strings.Fields(note)) < 7 {
			issues = append(issues, Issue{Code: "batch_candidate_review_thin_note", Message: record.UniqueIntentID + ":" + note})
		}
	}
	if len(record.RequiredFixes) == 0 {
		issues = append(issues, Issue{Code: "batch_candidate_review_missing_required_fixes", Message: record.UniqueIntentID})
	}
	if record.CTAStatus != "draft_contextual_not_public" {
		issues = append(issues, Issue{Code: "batch_candidate_review_invalid_cta_status", Message: record.CTAStatus})
	}
	if len(strings.Fields(record.CTADraft)) < 18 || !containsWhatsApp(record.CTADraft) {
		issues = append(issues, Issue{Code: "batch_candidate_review_thin_cta", Message: record.UniqueIntentID})
	}
	if hasPromiseCTA(record.CTADraft) {
		issues = append(issues, Issue{Code: "batch_candidate_review_promise_cta", Message: record.UniqueIntentID})
	}
	if len(strings.Fields(record.CTAContextMessage)) < 16 || !ctaContextHasOrigin(record) {
		issues = append(issues, Issue{Code: "batch_candidate_review_cta_context_without_origin", Message: record.UniqueIntentID})
	}
	if strings.TrimSpace(record.PublicationBlockReason) == "" {
		issues = append(issues, Issue{Code: "batch_candidate_review_missing_block_reason", Message: record.UniqueIntentID})
	}
	if record.RenderAllowed {
		issues = append(issues, Issue{Code: "batch_candidate_review_render_allowed", Message: record.UniqueIntentID})
	}
	if record.SitemapAllowed {
		issues = append(issues, Issue{Code: "batch_candidate_review_sitemap_allowed", Message: record.UniqueIntentID})
	}
	if record.PublicationAllowed {
		issues = append(issues, Issue{Code: "batch_candidate_review_publication_allowed", Message: record.UniqueIntentID})
	}
	if record.PublicPath != "" {
		issues = append(issues, Issue{Code: "batch_candidate_review_has_public_path", Message: record.PublicPath})
	}
	if record.CheckedAt == "" {
		issues = append(issues, Issue{Code: "batch_candidate_review_without_checked_at", Message: record.UniqueIntentID})
	}
	return Report{Issues: issues}
}

func LoadRecords(root string) ([]Entry, Report) {
	projectRoot, err := findProjectRoot(root)
	if err != nil {
		return nil, Report{Issues: []Issue{{Code: "project_root_not_found", Message: err.Error()}}}
	}
	path := filepath.Join(projectRoot, "data", "editorial", "batch_candidate_reviews.jsonl")
	file, err := os.Open(path)
	if err != nil {
		return nil, Report{Issues: []Issue{{Code: "batch_candidate_reviews_missing", Message: err.Error()}}}
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
			issues = append(issues, Issue{Code: "batch_candidate_review_invalid_json", Message: fmt.Sprintf("line=%d %s", lineNumber, err.Error())})
			continue
		}
		entries = append(entries, Entry{Line: lineNumber, Record: record})
	}
	if err := scanner.Err(); err != nil {
		issues = append(issues, Issue{Code: "batch_candidate_review_scan_failed", Message: err.Error()})
	}
	return entries, Report{Issues: issues}
}

func containsWhatsApp(value string) bool {
	return strings.Contains(strings.ToLower(value), "whatsapp")
}

func hasPromiseCTA(value string) bool {
	normalized := strings.ToLower(value)
	for _, promise := range []string{
		"garantimos",
		"garantia",
		"liminar garantida",
		"causa ganha",
		"resultado garantido",
		"resolver o processo",
		"em 24 horas",
	} {
		if strings.Contains(normalized, promise) {
			return true
		}
	}
	return false
}

func ctaContextHasOrigin(record Record) bool {
	for _, token := range []string{
		"Origem: " + record.CandidatePath,
		"Gate: " + record.GateID,
		"Intent: " + record.UniqueIntentID,
	} {
		if !strings.Contains(record.CTAContextMessage, token) {
			return false
		}
	}
	if record.Term != "" && !strings.Contains(record.CTAContextMessage, record.Term) {
		return false
	}
	return true
}

func convertGateIssues(report batchcandidategates.Report) []Issue {
	issues := make([]Issue, 0, len(report.Issues))
	for _, issue := range report.Issues {
		issues = append(issues, Issue{Code: "batch_candidate_review_gate_" + issue.Code, Message: issue.Message})
	}
	return issues
}

func convertSourceAuditIssues(report batchsourceaudit.Report) []Issue {
	issues := make([]Issue, 0, len(report.Issues))
	for _, issue := range report.Issues {
		issues = append(issues, Issue{Code: "batch_candidate_review_source_audit_" + issue.Code, Message: issue.Message})
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

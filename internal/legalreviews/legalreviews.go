package legalreviews

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"portaljuridico/internal/authorialdrafts"
	"portaljuridico/internal/prepublication"
	"portaljuridico/internal/sourceresolutions"
)

type Record struct {
	TermID               string   `json:"term_id"`
	Term                 string   `json:"term"`
	ReviewID             string   `json:"review_id"`
	ReviewStatus         string   `json:"review_status"`
	Language             string   `json:"language"`
	AuthorialDraftID     string   `json:"authorial_draft_id"`
	SourceResolutionID   string   `json:"source_resolution_id"`
	PrepublicationGateID string   `json:"prepublication_gate_id"`
	ReviewerRole         string   `json:"reviewer_role"`
	LegalReviewNotes     []string `json:"legal_review_notes"`
	RequiredFixes        []string `json:"required_fixes"`
	CTADraft             string   `json:"cta_draft"`
	CTAContextMessage    string   `json:"cta_context_message"`
	CTAOriginPath        string   `json:"cta_origin_path"`
	CTAOriginGateID      string   `json:"cta_origin_gate_id"`
	CTAStatus            string   `json:"cta_status"`
	RenderAllowed        bool     `json:"render_allowed"`
	SitemapAllowed       bool     `json:"sitemap_allowed"`
	PublicationAllowed   bool     `json:"publication_allowed"`
	PublicPath           string   `json:"public_path"`
	CheckedAt            string   `json:"checked_at"`
}

func (r Record) CTAContextMessageContains(token string) bool {
	return strings.Contains(r.CTAContextMessage, token)
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

var termIDPattern = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)

func Validate(root string) Report {
	entries, report := LoadRecords(root)
	if !report.Passed() {
		return report
	}
	issues := make([]Issue, 0)
	if len(entries) == 0 {
		issues = append(issues, Issue{Code: "legal_reviews_empty", Message: "data/editorial/legal_reviews.jsonl"})
	}

	authorialDrafts := validAuthorialDrafts(root)
	sourceResolutions := validSourceResolutions(root)
	prepublicationGates := validPrepublicationGates(root)
	seen := make(map[string]int)
	for _, entry := range entries {
		recordReport := ValidateRecord(entry.Record)
		for _, issue := range recordReport.Issues {
			issue.Message = fmt.Sprintf("line=%d %s", entry.Line, issue.Message)
			issues = append(issues, issue)
		}
		if !authorialDrafts[entry.Record.AuthorialDraftID] {
			issues = append(issues, Issue{Code: "legal_review_missing_authorial_draft", Message: fmt.Sprintf("line=%d id=%s", entry.Line, entry.Record.AuthorialDraftID)})
		}
		if !sourceResolutions[entry.Record.SourceResolutionID] {
			issues = append(issues, Issue{Code: "legal_review_missing_source_resolution", Message: fmt.Sprintf("line=%d id=%s", entry.Line, entry.Record.SourceResolutionID)})
		}
		if !prepublicationGates[entry.Record.PrepublicationGateID] {
			issues = append(issues, Issue{Code: "legal_review_missing_prepublication_gate", Message: fmt.Sprintf("line=%d id=%s", entry.Line, entry.Record.PrepublicationGateID)})
		}
		if previousLine := seen[entry.Record.ReviewID]; previousLine > 0 {
			issues = append(issues, Issue{Code: "legal_review_duplicate_id", Message: fmt.Sprintf("line=%d previous_line=%d id=%s", entry.Line, previousLine, entry.Record.ReviewID)})
		}
		seen[entry.Record.ReviewID] = entry.Line
	}
	return Report{Issues: issues}
}

func LoadRecords(root string) ([]Entry, Report) {
	projectRoot, err := findProjectRoot(root)
	if err != nil {
		return nil, Report{Issues: []Issue{{Code: "project_root_not_found", Message: err.Error()}}}
	}
	path := filepath.Join(projectRoot, "data", "editorial", "legal_reviews.jsonl")
	file, err := os.Open(path)
	if err != nil {
		return nil, Report{Issues: []Issue{{Code: "legal_reviews_missing", Message: err.Error()}}}
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
			issues = append(issues, Issue{Code: "legal_review_invalid_json", Message: fmt.Sprintf("line=%d %s", lineNumber, err.Error())})
			continue
		}
		entries = append(entries, Entry{Line: lineNumber, Record: record})
	}
	if err := scanner.Err(); err != nil {
		issues = append(issues, Issue{Code: "legal_review_scan_failed", Message: err.Error()})
	}
	return entries, Report{Issues: issues}
}

func ValidateRecord(record Record) Report {
	issues := make([]Issue, 0)
	if record.TermID == "" || !termIDPattern.MatchString(record.TermID) || record.Term == "" {
		issues = append(issues, Issue{Code: "legal_review_invalid_term", Message: record.TermID})
	}
	if record.ReviewID == "" || !termIDPattern.MatchString(record.ReviewID) {
		issues = append(issues, Issue{Code: "legal_review_invalid_id", Message: record.ReviewID})
	}
	if record.ReviewStatus != "legal_editorial_review_blocked" {
		issues = append(issues, Issue{Code: "legal_review_invalid_status", Message: record.ReviewStatus})
	}
	if record.Language != "pt-BR" {
		issues = append(issues, Issue{Code: "legal_review_not_ptbr", Message: record.TermID})
	}
	for _, id := range []struct {
		code  string
		value string
	}{
		{"legal_review_invalid_authorial_draft_id", record.AuthorialDraftID},
		{"legal_review_invalid_source_resolution_id", record.SourceResolutionID},
		{"legal_review_invalid_prepublication_gate_id", record.PrepublicationGateID},
	} {
		if id.value == "" || !termIDPattern.MatchString(id.value) {
			issues = append(issues, Issue{Code: id.code, Message: id.value})
		}
	}
	if record.ReviewerRole != "juridico_editorial_lab" {
		issues = append(issues, Issue{Code: "legal_review_invalid_reviewer_role", Message: record.ReviewerRole})
	}
	if len(record.LegalReviewNotes) < 3 {
		issues = append(issues, Issue{Code: "legal_review_too_few_notes", Message: record.TermID})
	}
	for _, note := range record.LegalReviewNotes {
		if len(strings.Fields(note)) < 6 {
			issues = append(issues, Issue{Code: "legal_review_thin_note", Message: record.TermID + ":" + note})
		}
	}
	if len(record.RequiredFixes) == 0 {
		issues = append(issues, Issue{Code: "legal_review_missing_required_fixes", Message: record.TermID})
	}
	if record.CTAStatus != "draft_contextual_not_public" {
		issues = append(issues, Issue{Code: "legal_review_invalid_cta_status", Message: record.CTAStatus})
	}
	if len(strings.Fields(record.CTADraft)) < 18 || !containsWhatsApp(record.CTADraft) {
		issues = append(issues, Issue{Code: "legal_review_thin_cta", Message: record.TermID})
	}
	if hasPromiseCTA(record.CTADraft) {
		issues = append(issues, Issue{Code: "legal_review_promise_cta", Message: record.TermID})
	}
	if record.CTAOriginPath == "" || record.CTAOriginGateID == "" {
		issues = append(issues, Issue{Code: "legal_review_missing_cta_origin", Message: record.TermID})
	}
	if record.CTAOriginGateID != "" && !termIDPattern.MatchString(record.CTAOriginGateID) {
		issues = append(issues, Issue{Code: "legal_review_invalid_cta_origin_gate", Message: record.CTAOriginGateID})
	}
	if len(strings.Fields(record.CTAContextMessage)) < 14 || !ctaContextHasOrigin(record) {
		issues = append(issues, Issue{Code: "legal_review_cta_context_without_origin", Message: record.TermID})
	}
	if record.RenderAllowed {
		issues = append(issues, Issue{Code: "legal_review_render_allowed", Message: record.TermID})
	}
	if record.SitemapAllowed {
		issues = append(issues, Issue{Code: "legal_review_sitemap_allowed", Message: record.TermID})
	}
	if record.PublicationAllowed {
		issues = append(issues, Issue{Code: "legal_review_publication_allowed", Message: record.TermID})
	}
	if record.PublicPath != "" {
		issues = append(issues, Issue{Code: "legal_review_has_public_path", Message: record.PublicPath})
	}
	if record.CheckedAt == "" {
		issues = append(issues, Issue{Code: "legal_review_without_checked_at", Message: record.TermID})
	}
	return Report{Issues: issues}
}

func validAuthorialDrafts(root string) map[string]bool {
	entries, report := authorialdrafts.LoadRecords(root)
	if !report.Passed() {
		return map[string]bool{}
	}
	valid := make(map[string]bool)
	for _, entry := range entries {
		if authorialdrafts.ValidateRecord(entry.Record).Passed() {
			valid[entry.Record.TermID] = true
		}
	}
	return valid
}

func validSourceResolutions(root string) map[string]bool {
	entries, report := sourceresolutions.LoadRecords(root)
	if !report.Passed() {
		return map[string]bool{}
	}
	valid := make(map[string]bool)
	for _, entry := range entries {
		if sourceresolutions.ValidateRecord(entry.Record).Passed() {
			valid[entry.Record.ResolutionID] = true
		}
	}
	return valid
}

func validPrepublicationGates(root string) map[string]bool {
	entries, report := prepublication.LoadRecords(root)
	if !report.Passed() {
		return map[string]bool{}
	}
	valid := make(map[string]bool)
	for _, entry := range entries {
		if prepublication.ValidateRecord(entry.Record).Passed() {
			valid[entry.Record.GateID] = true
		}
	}
	return valid
}

func containsWhatsApp(value string) bool {
	return strings.Contains(strings.ToLower(value), "whatsapp")
}

func hasPromiseCTA(value string) bool {
	normalized := strings.ToLower(value)
	promises := []string{
		"garantimos",
		"garantia",
		"liminar garantida",
		"causa ganha",
		"resultado garantido",
		"resultado certo",
		"ganho garantido",
		"vitoria garantida",
		"vitória garantida",
		"resolver seu plano",
		"em 24 horas",
	}
	for _, promise := range promises {
		if strings.Contains(normalized, promise) {
			return true
		}
	}
	return false
}

func ctaContextHasOrigin(record Record) bool {
	if !strings.Contains(record.CTAContextMessage, "Origem:") {
		return false
	}
	if record.CTAOriginPath != "" && !strings.Contains(record.CTAContextMessage, record.CTAOriginPath) {
		return false
	}
	if record.CTAOriginGateID != "" && !strings.Contains(record.CTAContextMessage, record.CTAOriginGateID) {
		return false
	}
	if record.Term != "" && !strings.Contains(record.CTAContextMessage, record.Term) {
		return false
	}
	return true
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

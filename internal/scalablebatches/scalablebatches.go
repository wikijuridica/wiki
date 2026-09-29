package scalablebatches

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type Record struct {
	BatchID                 string   `json:"batch_id"`
	LegalArea               string   `json:"legal_area"`
	BatchStatus             string   `json:"batch_status"`
	GenerationStrategy      string   `json:"generation_strategy"`
	ValidationMode          string   `json:"validation_mode"`
	ReviewStrategy          string   `json:"review_strategy"`
	PlannedPageCount        int      `json:"planned_page_count"`
	DigitalOnly             bool     `json:"digital_only"`
	WhatsAppContextRequired bool     `json:"whatsapp_context_required"`
	AutoRewriteOnFail       bool     `json:"auto_rewrite_on_fail"`
	MinimumHumanScore       int      `json:"minimum_human_score"`
	MaximumAILikeScore      int      `json:"maximum_ai_like_score"`
	MaximumSimilarity       float64  `json:"maximum_similarity"`
	IntentDimensions        []string `json:"intent_dimensions"`
	SourceFamilies          []string `json:"source_families"`
	SampleValidationRules   []string `json:"sample_validation_rules"`
	CTAContextTemplate      string   `json:"cta_context_template"`
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
	if len(entries) == 0 {
		issues = append(issues, Issue{Code: "batches_empty", Message: "data/editorial/scalable_content_batches.jsonl"})
	}
	seen := make(map[string]int)
	areas := make(map[string]bool)
	for _, entry := range entries {
		recordReport := ValidateRecord(entry.Record)
		for _, issue := range recordReport.Issues {
			issue.Message = fmt.Sprintf("line=%d %s", entry.Line, issue.Message)
			issues = append(issues, issue)
		}
		if previous := seen[entry.Record.BatchID]; previous > 0 {
			issues = append(issues, Issue{Code: "batch_duplicate_id", Message: fmt.Sprintf("line=%d previous_line=%d id=%s", entry.Line, previous, entry.Record.BatchID)})
		}
		seen[entry.Record.BatchID] = entry.Line
		if entry.Record.LegalArea != "" {
			areas[entry.Record.LegalArea] = true
		}
	}
	if len(entries) < 6 {
		issues = append(issues, Issue{Code: "batch_too_few_legal_families", Message: fmt.Sprintf("batches=%d", len(entries))})
	}
	if len(areas) < 6 {
		issues = append(issues, Issue{Code: "batch_low_area_diversity", Message: fmt.Sprintf("areas=%d", len(areas))})
	}
	if TotalPlannedPages(entries) < 1000000 {
		issues = append(issues, Issue{Code: "batch_total_pages_too_low", Message: fmt.Sprintf("planned=%d", TotalPlannedPages(entries))})
	}
	return Report{Issues: issues}
}

func LoadRecords(root string) ([]Entry, Report) {
	projectRoot, err := findProjectRoot(root)
	if err != nil {
		return nil, Report{Issues: []Issue{{Code: "project_root_not_found", Message: err.Error()}}}
	}
	path := filepath.Join(projectRoot, "data", "editorial", "scalable_content_batches.jsonl")
	file, err := os.Open(path)
	if err != nil {
		return nil, Report{Issues: []Issue{{Code: "batches_missing", Message: err.Error()}}}
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
			issues = append(issues, Issue{Code: "batch_invalid_json", Message: fmt.Sprintf("line=%d %s", lineNumber, err.Error())})
			continue
		}
		entries = append(entries, Entry{Line: lineNumber, Record: record})
	}
	if err := scanner.Err(); err != nil {
		issues = append(issues, Issue{Code: "batch_scan_failed", Message: err.Error()})
	}
	return entries, Report{Issues: issues}
}

func ValidateRecord(record Record) Report {
	issues := make([]Issue, 0)
	if record.BatchID == "" || record.LegalArea == "" {
		issues = append(issues, Issue{Code: "batch_missing_identity", Message: record.BatchID})
	}
	if record.BatchStatus != "mass_generation_blocked" {
		issues = append(issues, Issue{Code: "batch_invalid_status", Message: record.BatchStatus})
	}
	if record.GenerationStrategy != "unique_intent_mass_batch" || containsAny(record.GenerationStrategy, []string{"keyword", "permutation", "template"}) {
		issues = append(issues, Issue{Code: "batch_template_strategy", Message: record.GenerationStrategy})
	}
	if record.ValidationMode != "mass_batch_before_publication" {
		issues = append(issues, Issue{Code: "batch_single_script_validation", Message: record.ValidationMode})
	}
	if record.ReviewStrategy == "manual_review_page_by_page" || record.ReviewStrategy == "" {
		issues = append(issues, Issue{Code: "batch_manual_review_bottleneck", Message: record.ReviewStrategy})
	}
	if record.PlannedPageCount < 10000 {
		issues = append(issues, Issue{Code: "batch_planned_pages_too_low", Message: fmt.Sprintf("%s=%d", record.BatchID, record.PlannedPageCount)})
	}
	if !record.DigitalOnly {
		issues = append(issues, Issue{Code: "batch_not_digital_only", Message: record.BatchID})
	}
	if !record.WhatsAppContextRequired {
		issues = append(issues, Issue{Code: "batch_missing_whatsapp_context", Message: record.BatchID})
	}
	if !record.AutoRewriteOnFail {
		issues = append(issues, Issue{Code: "batch_rewrite_disabled", Message: record.BatchID})
	}
	if record.MinimumHumanScore < 85 {
		issues = append(issues, Issue{Code: "batch_human_score_too_low", Message: fmt.Sprintf("%s=%d", record.BatchID, record.MinimumHumanScore)})
	}
	if record.MaximumAILikeScore > 20 {
		issues = append(issues, Issue{Code: "batch_ai_score_too_high", Message: fmt.Sprintf("%s=%d", record.BatchID, record.MaximumAILikeScore)})
	}
	if record.MaximumSimilarity <= 0 || record.MaximumSimilarity > 0.64 {
		issues = append(issues, Issue{Code: "batch_similarity_too_high", Message: fmt.Sprintf("%s=%.2f", record.BatchID, record.MaximumSimilarity)})
	}
	if len(record.IntentDimensions) < 6 {
		issues = append(issues, Issue{Code: "batch_too_few_dimensions", Message: record.BatchID})
	}
	if len(record.SourceFamilies) < 2 || !allOfficialSources(record.SourceFamilies) {
		issues = append(issues, Issue{Code: "batch_too_few_sources", Message: record.BatchID})
	}
	if len(record.SampleValidationRules) < 6 {
		issues = append(issues, Issue{Code: "batch_too_few_validation_rules", Message: record.BatchID})
	}
	if !strings.Contains(record.CTAContextTemplate, "{unique_intent_id}") || !strings.Contains(record.CTAContextTemplate, "{source_family}") {
		issues = append(issues, Issue{Code: "batch_cta_context_template_weak", Message: record.BatchID})
	}
	if record.RenderAllowed {
		issues = append(issues, Issue{Code: "batch_render_allowed", Message: record.BatchID})
	}
	if record.SitemapAllowed {
		issues = append(issues, Issue{Code: "batch_sitemap_allowed", Message: record.BatchID})
	}
	if record.PublicationAllowed {
		issues = append(issues, Issue{Code: "batch_publication_allowed", Message: record.BatchID})
	}
	if record.PublicPath != "" {
		issues = append(issues, Issue{Code: "batch_has_public_path", Message: record.PublicPath})
	}
	if record.CheckedAt == "" {
		issues = append(issues, Issue{Code: "batch_without_checked_at", Message: record.BatchID})
	}
	return Report{Issues: issues}
}

func TotalPlannedPages(entries []Entry) int {
	total := 0
	for _, entry := range entries {
		total += entry.Record.PlannedPageCount
	}
	return total
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

func containsAny(value string, needles []string) bool {
	lower := strings.ToLower(value)
	for _, needle := range needles {
		if strings.Contains(lower, needle) {
			return true
		}
	}
	return false
}

func allOfficialSources(sources []string) bool {
	for _, source := range sources {
		if !isOfficialSource(source) {
			return false
		}
	}
	return true
}

func isOfficialSource(source string) bool {
	prefixes := []string{
		"https://www.gov.br/",
		"https://www.planalto.gov.br/",
		"https://www.cnj.jus.br/",
		"https://www.bcb.gov.br/",
		"https://www.tst.jus.br/",
		"https://www.stj.jus.br/",
		"https://www.stf.jus.br/",
		"https://www.ans.gov.br/",
	}
	for _, prefix := range prefixes {
		if strings.HasPrefix(source, prefix) {
			return true
		}
	}
	return false
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

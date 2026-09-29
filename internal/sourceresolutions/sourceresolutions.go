package sourceresolutions

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

type OfficialSource struct {
	URL          string `json:"url"`
	SourceName   string `json:"source_name"`
	SourceType   string `json:"source_type"`
	UseInContent string `json:"use_in_content"`
	CheckedAt    string `json:"checked_at"`
}

type Record struct {
	TermID                    string           `json:"term_id"`
	Term                      string           `json:"term"`
	ResolutionID              string           `json:"resolution_id"`
	ResolutionStatus          string           `json:"resolution_status"`
	Language                  string           `json:"language"`
	SourceBriefID             string           `json:"source_brief_id"`
	SourceResolutionScope     string           `json:"source_resolution_scope"`
	OfficialSources           []OfficialSource `json:"official_sources"`
	SourceSummary             string           `json:"source_summary"`
	SpecificityScore          int              `json:"specificity_score"`
	RemainingPublicationGates []string         `json:"remaining_publication_gates"`
	IndexPolicy               string           `json:"index_policy"`
	PublicationAllowed        bool             `json:"publication_allowed"`
	PublicPath                string           `json:"public_path"`
	CheckedAt                 string           `json:"checked_at"`
}

func (r Record) HasSourceType(sourceType string) bool {
	for _, source := range r.OfficialSources {
		if source.SourceType == sourceType {
			return true
		}
	}
	return false
}

func (r Record) SourceTypes() []string {
	seen := make(map[string]bool)
	types := make([]string, 0)
	for _, source := range r.OfficialSources {
		if source.SourceType == "" || seen[source.SourceType] {
			continue
		}
		seen[source.SourceType] = true
		types = append(types, source.SourceType)
	}
	sort.Strings(types)
	return types
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
		issues = append(issues, Issue{Code: "source_resolutions_empty", Message: "data/editorial/source_resolutions.jsonl"})
	}
	seen := make(map[string]int)
	for _, entry := range entries {
		recordReport := ValidateRecord(entry.Record)
		for _, issue := range recordReport.Issues {
			issue.Message = fmt.Sprintf("line=%d %s", entry.Line, issue.Message)
			issues = append(issues, issue)
		}
		key := strings.TrimSpace(entry.Record.ResolutionID)
		if key != "" {
			if previousLine := seen[key]; previousLine > 0 {
				issues = append(issues, Issue{Code: "source_resolution_duplicate_id", Message: fmt.Sprintf("line=%d previous_line=%d id=%s", entry.Line, previousLine, key)})
			}
			seen[key] = entry.Line
		}
	}
	return Report{Issues: issues}
}

func LoadRecords(root string) ([]Entry, Report) {
	projectRoot, err := findProjectRoot(root)
	if err != nil {
		return nil, Report{Issues: []Issue{{Code: "project_root_not_found", Message: err.Error()}}}
	}
	path := filepath.Join(projectRoot, "data", "editorial", "source_resolutions.jsonl")
	file, err := os.Open(path)
	if err != nil {
		return nil, Report{Issues: []Issue{{Code: "source_resolutions_missing", Message: err.Error()}}}
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
			issues = append(issues, Issue{Code: "source_resolution_invalid_json", Message: fmt.Sprintf("line=%d %s", lineNumber, err.Error())})
			continue
		}
		entries = append(entries, Entry{Line: lineNumber, Record: record})
	}
	if err := scanner.Err(); err != nil {
		issues = append(issues, Issue{Code: "source_resolution_scan_failed", Message: err.Error()})
	}
	return entries, Report{Issues: issues}
}

func ValidateRecord(record Record) Report {
	issues := make([]Issue, 0)
	if record.TermID == "" || !termIDPattern.MatchString(record.TermID) || record.Term == "" {
		issues = append(issues, Issue{Code: "source_resolution_invalid_term", Message: record.TermID})
	}
	if record.ResolutionID == "" || !termIDPattern.MatchString(record.ResolutionID) {
		issues = append(issues, Issue{Code: "source_resolution_invalid_id", Message: record.ResolutionID})
	}
	if record.ResolutionStatus != "source_resolved_for_prepublication" {
		issues = append(issues, Issue{Code: "source_resolution_invalid_status", Message: record.ResolutionStatus})
	}
	if record.Language != "pt-BR" {
		issues = append(issues, Issue{Code: "source_resolution_not_ptbr", Message: record.TermID})
	}
	if record.SourceBriefID == "" || !termIDPattern.MatchString(record.SourceBriefID) {
		issues = append(issues, Issue{Code: "source_resolution_missing_source_brief", Message: record.TermID})
	}
	if len(record.OfficialSources) < 4 {
		issues = append(issues, Issue{Code: "source_resolution_too_few_sources", Message: record.TermID})
	}
	if !record.HasSourceType("primary_law") {
		issues = append(issues, Issue{Code: "source_resolution_missing_primary_law", Message: record.TermID})
	}
	if !record.HasSourceType("coverage_rule") {
		issues = append(issues, Issue{Code: "source_resolution_missing_coverage_rule", Message: record.TermID})
	}
	for _, source := range record.OfficialSources {
		issues = append(issues, validateOfficialSource(record.TermID, source)...)
	}
	if len(strings.Fields(record.SourceSummary)) < 18 {
		issues = append(issues, Issue{Code: "source_resolution_thin_summary", Message: record.TermID})
	}
	if record.SpecificityScore < 80 {
		issues = append(issues, Issue{Code: "source_resolution_low_specificity", Message: fmt.Sprintf("%s:%d", record.TermID, record.SpecificityScore)})
	}
	if len(record.RemainingPublicationGates) == 0 {
		issues = append(issues, Issue{Code: "source_resolution_missing_remaining_gates", Message: record.TermID})
	}
	if record.IndexPolicy != "noindex" {
		issues = append(issues, Issue{Code: "source_resolution_not_noindex", Message: record.IndexPolicy})
	}
	if record.PublicationAllowed {
		issues = append(issues, Issue{Code: "source_resolution_publication_allowed", Message: record.TermID})
	}
	if record.PublicPath != "" {
		issues = append(issues, Issue{Code: "source_resolution_has_public_path", Message: record.PublicPath})
	}
	if record.CheckedAt == "" {
		issues = append(issues, Issue{Code: "source_resolution_without_checked_at", Message: record.TermID})
	}
	return Report{Issues: issues}
}

func validateOfficialSource(termID string, source OfficialSource) []Issue {
	issues := make([]Issue, 0)
	if source.URL == "" || !isOfficialURL(source.URL) {
		issues = append(issues, Issue{Code: "source_resolution_untrusted_url", Message: termID + ":" + source.URL})
	}
	if isBroadSource(source) {
		issues = append(issues, Issue{Code: "source_resolution_broad_source", Message: termID + ":" + source.URL})
	}
	if source.SourceName == "" || source.SourceType == "" || len(strings.Fields(source.UseInContent)) < 5 {
		issues = append(issues, Issue{Code: "source_resolution_incomplete_source", Message: termID + ":" + source.URL})
	}
	if source.CheckedAt == "" {
		issues = append(issues, Issue{Code: "source_resolution_source_without_checked_at", Message: termID + ":" + source.URL})
	}
	return issues
}

func isBroadSource(source OfficialSource) bool {
	if source.URL == "https://www.gov.br/" || source.URL == "https://www.gov.br" {
		return true
	}
	broadTypes := []string{"institutional_home", "news_only", "generic_portal"}
	for _, sourceType := range broadTypes {
		if source.SourceType == sourceType {
			return true
		}
	}
	return false
}

func isOfficialURL(value string) bool {
	prefixes := []string{
		"https://www.gov.br/ans/",
		"https://planalto.gov.br/",
		"https://www.planalto.gov.br/",
	}
	for _, prefix := range prefixes {
		if strings.HasPrefix(value, prefix) {
			return true
		}
	}
	return false
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

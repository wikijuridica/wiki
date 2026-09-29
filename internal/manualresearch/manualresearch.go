package manualresearch

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

type Source struct {
	SourceID      string `json:"source_id"`
	URL           string `json:"url"`
	AuthorityRole string `json:"authority_role"`
}

type Record struct {
	TermID             string   `json:"term_id"`
	Term               string   `json:"term"`
	Language           string   `json:"language"`
	ResearchMethod     string   `json:"research_method"`
	PracticeArea       string   `json:"practice_area"`
	DigitalServiceMode string   `json:"digital_service_mode"`
	HiringIntent       string   `json:"hiring_intent"`
	TrendsRole         string   `json:"trends_role"`
	TrendURLs          []string `json:"trend_urls"`
	OfficialSources    []Source `json:"official_sources"`
	ContentUse         string   `json:"content_use"`
	PublicationAllowed bool     `json:"publication_allowed"`
	PublicPath         string   `json:"public_path"`
	CheckedAt          string   `json:"checked_at"`
	ResearchNotes      string   `json:"research_notes"`
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
	seen := make(map[string]bool)
	for _, entry := range entries {
		report := ValidateRecord(entry.Record)
		for _, issue := range report.Issues {
			issue.Message = fmt.Sprintf("line=%d %s", entry.Line, issue.Message)
			issues = append(issues, issue)
		}
		if seen[entry.Record.TermID] {
			issues = append(issues, Issue{Code: "duplicate_manual_research_term", Message: entry.Record.TermID})
		}
		seen[entry.Record.TermID] = true
	}
	return Report{Issues: issues}
}

func LoadRecords(root string) ([]Entry, Report) {
	projectRoot, err := findProjectRoot(root)
	if err != nil {
		return nil, Report{Issues: []Issue{{Code: "project_root_not_found", Message: err.Error()}}}
	}
	path := filepath.Join(projectRoot, "data", "research", "high_intent_terms.jsonl")
	file, err := os.Open(path)
	if err != nil {
		return nil, Report{Issues: []Issue{{Code: "manual_research_missing", Message: err.Error()}}}
	}
	defer file.Close()

	issues := make([]Issue, 0)
	entries := make([]Entry, 0)
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 0, 4096), 32768)
	lineNumber := 0
	for scanner.Scan() {
		lineNumber++
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var record Record
		if err := json.Unmarshal([]byte(line), &record); err != nil {
			issues = append(issues, Issue{Code: "manual_research_invalid_json", Message: fmt.Sprintf("line=%d %s", lineNumber, err.Error())})
			continue
		}
		entries = append(entries, Entry{Line: lineNumber, Record: record})
	}
	if err := scanner.Err(); err != nil {
		issues = append(issues, Issue{Code: "manual_research_scan_failed", Message: err.Error()})
	}
	return entries, Report{Issues: issues}
}

func ValidateRecord(record Record) Report {
	issues := make([]Issue, 0)
	if record.TermID == "" || !termIDPattern.MatchString(record.TermID) {
		issues = append(issues, Issue{Code: "invalid_manual_research_term_id", Message: record.TermID})
	}
	if record.Term == "" || record.Language != "pt-BR" {
		issues = append(issues, Issue{Code: "manual_research_not_ptbr", Message: record.TermID})
	}
	if record.ResearchMethod != "manual_web_research" {
		issues = append(issues, Issue{Code: "not_manual_web_research", Message: record.TermID + ":" + record.ResearchMethod})
	}
	if record.PracticeArea == "" {
		issues = append(issues, Issue{Code: "missing_practice_area", Message: record.TermID})
	}
	if record.DigitalServiceMode != "digital_only" {
		issues = append(issues, Issue{Code: "not_digital_only", Message: record.TermID})
	}
	if record.HiringIntent != "high" {
		issues = append(issues, Issue{Code: "not_high_hiring_intent", Message: record.TermID})
	}
	if record.TrendsRole != "orientation_only" {
		issues = append(issues, Issue{Code: "trend_not_orientation_only", Message: record.TermID + ":" + record.TrendsRole})
	}
	if len(record.TrendURLs) == 0 {
		issues = append(issues, Issue{Code: "missing_trend_orientation", Message: record.TermID})
	}
	for _, url := range record.TrendURLs {
		if !strings.HasPrefix(url, "https://trends.google.com.br/trends/explore") && !strings.HasPrefix(url, "https://trends.google.com/trends/explore") {
			issues = append(issues, Issue{Code: "unsafe_trend_url", Message: url})
		}
	}
	if len(record.OfficialSources) == 0 {
		issues = append(issues, Issue{Code: "missing_official_authority_sources", Message: record.TermID})
	}
	for _, source := range record.OfficialSources {
		if source.SourceID == "" || source.AuthorityRole == "" || !isOfficialURL(source.URL) {
			issues = append(issues, Issue{Code: "invalid_official_authority_source", Message: record.TermID + ":" + source.URL})
		}
	}
	if record.ContentUse != "brief_only" {
		issues = append(issues, Issue{Code: "invalid_content_use", Message: record.TermID + ":" + record.ContentUse})
	}
	if record.PublicationAllowed {
		issues = append(issues, Issue{Code: "research_publication_allowed", Message: record.TermID})
	}
	if record.PublicPath != "" {
		issues = append(issues, Issue{Code: "research_has_public_path", Message: record.PublicPath})
	}
	if record.CheckedAt == "" {
		issues = append(issues, Issue{Code: "manual_research_without_checked_at", Message: record.TermID})
	}
	if len(strings.Fields(record.ResearchNotes)) < 8 {
		issues = append(issues, Issue{Code: "thin_research_notes", Message: record.TermID})
	}
	return Report{Issues: issues}
}

func isOfficialURL(value string) bool {
	prefixes := []string{
		"https://www.gov.br/",
		"https://www.cnj.jus.br/",
		"https://www.planalto.gov.br/",
		"https://www.bcb.gov.br/",
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

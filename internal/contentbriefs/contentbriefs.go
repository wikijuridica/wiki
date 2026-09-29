package contentbriefs

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

type Section struct {
	Heading string `json:"heading"`
	Purpose string `json:"purpose"`
}

type Record struct {
	TermID             string    `json:"term_id"`
	Term               string    `json:"term"`
	BriefStatus        string    `json:"brief_status"`
	Language           string    `json:"language"`
	UniqueAngle        string    `json:"unique_angle"`
	ReaderProblem      string    `json:"reader_problem"`
	DigitalCTAReason   string    `json:"digital_cta_reason"`
	OfficialSourceURLs []string  `json:"official_source_urls"`
	Sections           []Section `json:"sections"`
	PublicationAllowed bool      `json:"publication_allowed"`
	PublicPath         string    `json:"public_path"`
	CheckedAt          string    `json:"checked_at"`
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
	angles := make(map[string]bool)
	for _, entry := range entries {
		report := ValidateRecord(entry.Record)
		for _, issue := range report.Issues {
			issue.Message = fmt.Sprintf("line=%d %s", entry.Line, issue.Message)
			issues = append(issues, issue)
		}
		angleKey := strings.ToLower(strings.TrimSpace(entry.Record.UniqueAngle))
		if angleKey != "" && angles[angleKey] {
			issues = append(issues, Issue{Code: "duplicate_brief_angle", Message: entry.Record.UniqueAngle})
		}
		angles[angleKey] = true
	}
	return Report{Issues: issues}
}

func LoadRecords(root string) ([]Entry, Report) {
	projectRoot, err := findProjectRoot(root)
	if err != nil {
		return nil, Report{Issues: []Issue{{Code: "project_root_not_found", Message: err.Error()}}}
	}
	path := filepath.Join(projectRoot, "data", "editorial", "content_briefs.jsonl")
	file, err := os.Open(path)
	if err != nil {
		return nil, Report{Issues: []Issue{{Code: "content_briefs_missing", Message: err.Error()}}}
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
			issues = append(issues, Issue{Code: "content_brief_invalid_json", Message: fmt.Sprintf("line=%d %s", lineNumber, err.Error())})
			continue
		}
		entries = append(entries, Entry{Line: lineNumber, Record: record})
	}
	if err := scanner.Err(); err != nil {
		issues = append(issues, Issue{Code: "content_brief_scan_failed", Message: err.Error()})
	}
	return entries, Report{Issues: issues}
}

func ValidateRecord(record Record) Report {
	issues := make([]Issue, 0)
	if record.TermID == "" || !termIDPattern.MatchString(record.TermID) || record.Term == "" {
		issues = append(issues, Issue{Code: "invalid_brief_term", Message: record.TermID})
	}
	if record.BriefStatus != "research_brief" {
		issues = append(issues, Issue{Code: "invalid_brief_status", Message: record.TermID + ":" + record.BriefStatus})
	}
	if record.Language != "pt-BR" {
		issues = append(issues, Issue{Code: "brief_not_ptbr", Message: record.TermID})
	}
	if isGenericAngle(record.UniqueAngle) {
		issues = append(issues, Issue{Code: "brief_generic_angle", Message: record.TermID})
	}
	if len(strings.Fields(record.ReaderProblem)) < 8 {
		issues = append(issues, Issue{Code: "brief_thin_reader_problem", Message: record.TermID})
	}
	if len(strings.Fields(record.DigitalCTAReason)) < 8 {
		issues = append(issues, Issue{Code: "brief_thin_cta_reason", Message: record.TermID})
	}
	if len(record.OfficialSourceURLs) == 0 {
		issues = append(issues, Issue{Code: "brief_without_official_source", Message: record.TermID})
	}
	for _, url := range record.OfficialSourceURLs {
		if !isOfficialURL(url) {
			issues = append(issues, Issue{Code: "brief_untrusted_source_url", Message: record.TermID + ":" + url})
		}
	}
	if len(record.Sections) < 5 {
		issues = append(issues, Issue{Code: "brief_too_few_sections", Message: record.TermID})
	}
	for _, section := range record.Sections {
		if isGenericHeading(section.Heading) {
			issues = append(issues, Issue{Code: "brief_generic_heading", Message: record.TermID + ":" + section.Heading})
		}
		if len(strings.Fields(section.Purpose)) < 3 {
			issues = append(issues, Issue{Code: "brief_thin_section_purpose", Message: record.TermID + ":" + section.Heading})
		}
	}
	if record.PublicationAllowed {
		issues = append(issues, Issue{Code: "brief_publication_allowed", Message: record.TermID})
	}
	if record.PublicPath != "" {
		issues = append(issues, Issue{Code: "brief_has_public_path", Message: record.PublicPath})
	}
	if record.CheckedAt == "" {
		issues = append(issues, Issue{Code: "brief_without_checked_at", Message: record.TermID})
	}
	return Report{Issues: issues}
}

func isGenericAngle(value string) bool {
	normalized := strings.ToLower(strings.TrimSpace(value))
	return len(strings.Fields(normalized)) < 6 || normalized == "explicar o tema" || strings.Contains(normalized, "conteudo informativo")
}

func isGenericHeading(value string) bool {
	normalized := strings.ToLower(strings.TrimSpace(value))
	generic := []string{"o que é", "quando procurar advogado", "documentos necessários", "como funciona", "quanto custa"}
	for _, item := range generic {
		if normalized == item || strings.Contains(normalized, item) {
			return true
		}
	}
	return false
}

func isOfficialURL(value string) bool {
	prefixes := []string{
		"https://www.gov.br/",
		"https://www.cnj.jus.br/",
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

package prepublication

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"portaljuridico/internal/content"
	"portaljuridico/internal/seo"
	"portaljuridico/internal/sourceblockers"
	"portaljuridico/internal/sourceresolutions"
)

type Record struct {
	TermID                      string   `json:"term_id"`
	Term                        string   `json:"term"`
	GateID                      string   `json:"gate_id"`
	GateStatus                  string   `json:"gate_status"`
	Language                    string   `json:"language"`
	SourceResolutionID          string   `json:"source_resolution_id"`
	SourceBlockerReconciliation string   `json:"source_blocker_reconciliation"`
	CandidatePath               string   `json:"candidate_path"`
	CandidateCanonicalURL       string   `json:"candidate_canonical_url"`
	CandidateRobots             string   `json:"candidate_robots"`
	CandidateTitle              string   `json:"candidate_title"`
	CandidateMetaDescription    string   `json:"candidate_meta_description"`
	RenderAllowed               bool     `json:"render_allowed"`
	SitemapAllowed              bool     `json:"sitemap_allowed"`
	PublicationAllowed          bool     `json:"publication_allowed"`
	PublicPath                  string   `json:"public_path"`
	RemainingGates              []string `json:"remaining_gates"`
	CheckedAt                   string   `json:"checked_at"`
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

var (
	termIDPattern         = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)
	cleanCandidatePath    = regexp.MustCompile(`^/[a-z0-9]+(-[a-z0-9]+)*/[a-z0-9]+(-[a-z0-9]+)*/$`)
	defaultProjectBaseURL = "https://wikijuridica.com.br"
)

func Validate(root string) Report {
	entries, report := LoadRecords(root)
	if !report.Passed() {
		return report
	}
	issues := make([]Issue, 0)
	if len(entries) == 0 {
		issues = append(issues, Issue{Code: "prepublication_gates_empty", Message: "data/editorial/prepublication_gates.jsonl"})
	}
	repo, repoErr := content.LoadRepository(root)
	baseURL := defaultProjectBaseURL
	if repoErr != nil {
		issues = append(issues, Issue{Code: "prepublication_base_url_unavailable", Message: repoErr.Error()})
	} else {
		baseURL = repo.BaseURL
	}

	resolutions := validSourceResolutionIDs(root)
	blockers := activeSourceBlockers(root)
	seen := make(map[string]int)
	for _, entry := range entries {
		recordReport := ValidateRecordWithBaseURL(entry.Record, baseURL)
		for _, issue := range recordReport.Issues {
			issue.Message = fmt.Sprintf("line=%d %s", entry.Line, issue.Message)
			issues = append(issues, issue)
		}
		if _, ok := resolutions[entry.Record.SourceResolutionID]; !ok {
			issues = append(issues, Issue{Code: "prepublication_missing_source_resolution", Message: fmt.Sprintf("line=%d id=%s", entry.Line, entry.Record.SourceResolutionID)})
		}
		if !blockers[entry.Record.TermID] {
			issues = append(issues, Issue{Code: "prepublication_missing_active_source_blocker", Message: fmt.Sprintf("line=%d term_id=%s", entry.Line, entry.Record.TermID)})
		}
		if previousLine := seen[entry.Record.GateID]; previousLine > 0 {
			issues = append(issues, Issue{Code: "prepublication_duplicate_gate_id", Message: fmt.Sprintf("line=%d previous_line=%d id=%s", entry.Line, previousLine, entry.Record.GateID)})
		}
		seen[entry.Record.GateID] = entry.Line
	}
	return Report{Issues: issues}
}

func LoadRecords(root string) ([]Entry, Report) {
	projectRoot, err := findProjectRoot(root)
	if err != nil {
		return nil, Report{Issues: []Issue{{Code: "project_root_not_found", Message: err.Error()}}}
	}
	path := filepath.Join(projectRoot, "data", "editorial", "prepublication_gates.jsonl")
	file, err := os.Open(path)
	if err != nil {
		return nil, Report{Issues: []Issue{{Code: "prepublication_gates_missing", Message: err.Error()}}}
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
			issues = append(issues, Issue{Code: "prepublication_gate_invalid_json", Message: fmt.Sprintf("line=%d %s", lineNumber, err.Error())})
			continue
		}
		entries = append(entries, Entry{Line: lineNumber, Record: record})
	}
	if err := scanner.Err(); err != nil {
		issues = append(issues, Issue{Code: "prepublication_gate_scan_failed", Message: err.Error()})
	}
	return entries, Report{Issues: issues}
}

func ValidateRecord(record Record) Report {
	return ValidateRecordWithBaseURL(record, defaultProjectBaseURL)
}

func ValidateRecordWithBaseURL(record Record, baseURL string) Report {
	issues := make([]Issue, 0)
	baseURL = strings.TrimRight(baseURL, "/")
	if !seo.IsAbsoluteHTTPSURL(baseURL) {
		issues = append(issues, Issue{Code: "prepublication_base_url_not_https", Message: baseURL})
	}
	if record.TermID == "" || !termIDPattern.MatchString(record.TermID) || record.Term == "" {
		issues = append(issues, Issue{Code: "prepublication_invalid_term", Message: record.TermID})
	}
	if record.GateID == "" || !termIDPattern.MatchString(record.GateID) {
		issues = append(issues, Issue{Code: "prepublication_invalid_gate_id", Message: record.GateID})
	}
	if record.GateStatus != "blocked_prepublication" {
		issues = append(issues, Issue{Code: "prepublication_invalid_status", Message: record.GateStatus})
	}
	if record.Language != "pt-BR" {
		issues = append(issues, Issue{Code: "prepublication_not_ptbr", Message: record.TermID})
	}
	if record.SourceResolutionID == "" || !termIDPattern.MatchString(record.SourceResolutionID) {
		issues = append(issues, Issue{Code: "prepublication_invalid_source_resolution", Message: record.SourceResolutionID})
	}
	if record.SourceBlockerReconciliation != "source_resolved_blocker_still_active" {
		issues = append(issues, Issue{Code: "prepublication_bad_source_reconciliation", Message: record.SourceBlockerReconciliation})
	}
	if !cleanCandidatePath.MatchString(record.CandidatePath) || strings.ContainsAny(record.CandidatePath, "?#") {
		issues = append(issues, Issue{Code: "prepublication_candidate_path_not_clean", Message: record.CandidatePath})
	}
	expectedCanonical := baseURL + record.CandidatePath
	if !seo.IsAbsoluteHTTPSURL(record.CandidateCanonicalURL) {
		issues = append(issues, Issue{Code: "prepublication_canonical_not_https", Message: record.CandidateCanonicalURL})
	} else if record.CandidateCanonicalURL != expectedCanonical {
		issues = append(issues, Issue{Code: "prepublication_canonical_mismatch", Message: record.CandidateCanonicalURL})
	}
	if record.CandidateRobots != "noindex,follow" {
		issues = append(issues, Issue{Code: "prepublication_not_noindex", Message: record.CandidateRobots})
	}
	titleLen := len([]rune(record.CandidateTitle))
	if titleLen < seo.TitleMinCharacters {
		issues = append(issues, Issue{Code: "prepublication_title_too_short", Message: fmt.Sprintf("%s:%d", record.TermID, titleLen)})
	}
	if titleLen > seo.TitleMaxCharacters {
		issues = append(issues, Issue{Code: "prepublication_title_too_long", Message: fmt.Sprintf("%s:%d", record.TermID, titleLen)})
	}
	metaLen := len([]rune(record.CandidateMetaDescription))
	if metaLen < seo.MetaDescriptionMinCharacters {
		issues = append(issues, Issue{Code: "prepublication_meta_too_short", Message: fmt.Sprintf("%s:%d", record.TermID, metaLen)})
	}
	if metaLen > seo.MetaDescriptionMaxCharacters {
		issues = append(issues, Issue{Code: "prepublication_meta_too_long", Message: fmt.Sprintf("%s:%d", record.TermID, metaLen)})
	}
	if record.RenderAllowed {
		issues = append(issues, Issue{Code: "prepublication_render_allowed", Message: record.TermID})
	}
	if record.SitemapAllowed {
		issues = append(issues, Issue{Code: "prepublication_sitemap_allowed", Message: record.TermID})
	}
	if record.PublicationAllowed {
		issues = append(issues, Issue{Code: "prepublication_publication_allowed", Message: record.TermID})
	}
	if record.PublicPath != "" {
		issues = append(issues, Issue{Code: "prepublication_has_public_path", Message: record.PublicPath})
	}
	if len(record.RemainingGates) == 0 {
		issues = append(issues, Issue{Code: "prepublication_missing_remaining_gates", Message: record.TermID})
	}
	if record.CheckedAt == "" {
		issues = append(issues, Issue{Code: "prepublication_without_checked_at", Message: record.TermID})
	}
	return Report{Issues: issues}
}

func validSourceResolutionIDs(root string) map[string]bool {
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

func activeSourceBlockers(root string) map[string]bool {
	entries, report := sourceblockers.LoadRecords(root)
	if !report.Passed() {
		return map[string]bool{}
	}
	active := make(map[string]bool)
	for _, entry := range entries {
		if sourceblockers.ValidateRecord(entry.Record).Passed() {
			active[entry.Record.TermID] = true
		}
	}
	return active
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

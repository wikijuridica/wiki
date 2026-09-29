package terms

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"unicode"
)

type Seed struct {
	TermID             string `json:"term_id"`
	Term               string `json:"term"`
	Language           string `json:"language"`
	QualityState       string `json:"quality_state"`
	SourceID           string `json:"source_id"`
	SourceURL          string `json:"source_url"`
	CheckedAt          string `json:"checked_at"`
	IntentHint         string `json:"intent_hint"`
	CandidateID        string `json:"candidate_id,omitempty"`
	DemandEvidenceType string `json:"demand_evidence_type,omitempty"`
	DemandEvidenceURL  string `json:"demand_evidence_url,omitempty"`
	DemandQueryGroup   string `json:"demand_query_group,omitempty"`
	OnlineServiceMode  string `json:"online_service_mode,omitempty"`
	WhatsAppCTAIntent  string `json:"whatsapp_cta_intent,omitempty"`
	Notes              string `json:"notes"`
}

type Issue struct {
	Code    string
	Message string
}

type Report struct {
	Issues []Issue
}

var termIDPattern = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)

func ValidateSeeds(root string) Report {
	seeds, report := LoadSeeds(root)
	if !report.Passed() {
		return report
	}
	issues := make([]Issue, 0)
	seen := make(map[string]bool)
	for _, entry := range seeds {
		report := ValidateSeed(entry.Seed)
		for _, issue := range report.Issues {
			issue.Message = fmt.Sprintf("line=%d %s", entry.Line, issue.Message)
			issues = append(issues, issue)
		}
		if seen[entry.Seed.TermID] {
			issues = append(issues, Issue{Code: "duplicate_term_id", Message: entry.Seed.TermID})
		}
		seen[entry.Seed.TermID] = true
	}
	return Report{Issues: issues}
}

type SeedEntry struct {
	Line int
	Seed Seed
}

func LoadSeeds(root string) ([]SeedEntry, Report) {
	projectRoot, err := findProjectRoot(root)
	if err != nil {
		return nil, Report{Issues: []Issue{{Code: "project_root_not_found", Message: err.Error()}}}
	}
	path := filepath.Join(projectRoot, "data", "terms", "legal_terms.jsonl")
	file, err := os.Open(path)
	if err != nil {
		return nil, Report{Issues: []Issue{{Code: "term_seed_file_missing", Message: err.Error()}}}
	}
	defer file.Close()

	issues := make([]Issue, 0)
	seeds := make([]SeedEntry, 0)
	scanner := bufio.NewScanner(file)
	lineNumber := 0
	for scanner.Scan() {
		lineNumber++
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var seed Seed
		if err := json.Unmarshal([]byte(line), &seed); err != nil {
			issues = append(issues, Issue{Code: "term_seed_invalid_json", Message: fmt.Sprintf("line=%d %s", lineNumber, err.Error())})
			continue
		}
		seeds = append(seeds, SeedEntry{Line: lineNumber, Seed: seed})
	}
	if err := scanner.Err(); err != nil {
		issues = append(issues, Issue{Code: "term_seed_scan_failed", Message: err.Error()})
	}
	return seeds, Report{Issues: issues}
}

func ValidateSeed(seed Seed) Report {
	issues := make([]Issue, 0)
	if seed.TermID == "" || !termIDPattern.MatchString(seed.TermID) {
		issues = append(issues, Issue{Code: "invalid_term_id", Message: seed.TermID})
	}
	if !looksLikePortugueseTerm(seed.Term) {
		issues = append(issues, Issue{Code: "invalid_ptbr_term", Message: seed.Term})
	}
	if seed.Language != "pt-BR" {
		issues = append(issues, Issue{Code: "term_seed_not_ptbr", Message: seed.Language})
	}
	if seed.QualityState != "draft_only" {
		issues = append(issues, Issue{Code: "term_seed_not_draft_only", Message: seed.QualityState})
	}
	if seed.SourceID == "" || seed.SourceURL == "" {
		issues = append(issues, Issue{Code: "term_seed_without_source", Message: seed.TermID})
	}
	if seed.CheckedAt == "" {
		issues = append(issues, Issue{Code: "term_seed_without_checked_at", Message: seed.TermID})
	}
	if len(strings.Fields(seed.IntentHint)) < 6 {
		issues = append(issues, Issue{Code: "term_seed_without_intent_hint", Message: seed.TermID})
	}
	if seed.CandidateID != "" || seed.DemandEvidenceURL != "" || seed.OnlineServiceMode != "" || seed.WhatsAppCTAIntent != "" {
		if seed.CandidateID == "" {
			issues = append(issues, Issue{Code: "promoted_seed_without_candidate_id", Message: seed.TermID})
		}
		if seed.DemandEvidenceType != "google_trends_directional" || !isSafeDemandEvidenceURL(seed.DemandEvidenceURL) {
			issues = append(issues, Issue{Code: "promoted_seed_without_safe_demand_evidence", Message: seed.TermID})
		}
		if seed.OnlineServiceMode != "digital_only" {
			issues = append(issues, Issue{Code: "promoted_seed_not_digital_only", Message: seed.TermID})
		}
		if seed.WhatsAppCTAIntent != "high" {
			issues = append(issues, Issue{Code: "promoted_seed_without_high_whatsapp_intent", Message: seed.TermID})
		}
	}
	return Report{Issues: issues}
}

func isSafeDemandEvidenceURL(value string) bool {
	return strings.HasPrefix(value, "https://trends.google.com.br/trends/explore") ||
		strings.HasPrefix(value, "https://trends.google.com/trends/explore")
}

func looksLikePortugueseTerm(value string) bool {
	value = strings.TrimSpace(value)
	if value == "" || len(value) > 120 {
		return false
	}
	letters := 0
	for _, r := range value {
		if unicode.IsLetter(r) {
			letters++
			continue
		}
		if unicode.IsSpace(r) || r == '-' || r == '/' {
			continue
		}
		return false
	}
	return letters >= 3
}

func (r Report) Passed() bool {
	return len(r.Issues) == 0
}

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

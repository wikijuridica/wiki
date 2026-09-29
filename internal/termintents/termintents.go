package termintents

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

type Candidate struct {
	TermID               string `json:"term_id"`
	Term                 string `json:"term"`
	Language             string `json:"language"`
	CandidateStatus      string `json:"candidate_status"`
	QualityState         string `json:"quality_state"`
	PracticeArea         string `json:"practice_area"`
	DemandEvidenceType   string `json:"demand_evidence_type"`
	DemandEvidenceURL    string `json:"demand_evidence_url"`
	DemandQueryGroup     string `json:"demand_query_group"`
	OfficialSourceID     string `json:"official_source_id"`
	OfficialSourceURL    string `json:"official_source_url"`
	OnlineServiceMode    string `json:"online_service_mode"`
	WhatsAppCTAIntent    string `json:"whatsapp_cta_intent"`
	IntentReason         string `json:"intent_reason"`
	DigitalServiceReason string `json:"digital_service_reason"`
	RiskLevel            string `json:"risk_level"`
	CheckedAt            string `json:"checked_at"`
	PublicationAllowed   bool   `json:"publication_allowed"`
	PublicPath           string `json:"public_path"`
	Notes                string `json:"notes"`
}

type Entry struct {
	Line      int
	Candidate Candidate
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
	candidates, report := LoadCandidates(root)
	if !report.Passed() {
		return report
	}
	issues := make([]Issue, 0)
	seen := make(map[string]bool)
	for _, entry := range candidates {
		report := ValidateCandidate(entry.Candidate)
		for _, issue := range report.Issues {
			issue.Message = fmt.Sprintf("line=%d %s", entry.Line, issue.Message)
			issues = append(issues, issue)
		}
		if seen[entry.Candidate.TermID] {
			issues = append(issues, Issue{Code: "duplicate_term_candidate", Message: entry.Candidate.TermID})
		}
		seen[entry.Candidate.TermID] = true
	}
	return Report{Issues: issues}
}

func LoadCandidates(root string) ([]Entry, Report) {
	projectRoot, err := findProjectRoot(root)
	if err != nil {
		return nil, Report{Issues: []Issue{{Code: "project_root_not_found", Message: err.Error()}}}
	}
	path := filepath.Join(projectRoot, "data", "terms", "intent_candidates.jsonl")
	file, err := os.Open(path)
	if err != nil {
		return nil, Report{Issues: []Issue{{Code: "term_intent_file_missing", Message: err.Error()}}}
	}
	defer file.Close()

	issues := make([]Issue, 0)
	candidates := make([]Entry, 0)
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 0, 4096), 16384)
	lineNumber := 0
	for scanner.Scan() {
		lineNumber++
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var candidate Candidate
		if err := json.Unmarshal([]byte(line), &candidate); err != nil {
			issues = append(issues, Issue{Code: "term_intent_invalid_json", Message: fmt.Sprintf("line=%d %s", lineNumber, err.Error())})
			continue
		}
		candidates = append(candidates, Entry{Line: lineNumber, Candidate: candidate})
	}
	if err := scanner.Err(); err != nil {
		issues = append(issues, Issue{Code: "term_intent_scan_failed", Message: err.Error()})
	}
	return candidates, Report{Issues: issues}
}

func ValidateCandidate(candidate Candidate) Report {
	issues := make([]Issue, 0)
	if candidate.TermID == "" || !termIDPattern.MatchString(candidate.TermID) {
		issues = append(issues, Issue{Code: "invalid_term_id", Message: candidate.TermID})
	}
	if !looksLikePortugueseTerm(candidate.Term) {
		issues = append(issues, Issue{Code: "invalid_ptbr_term", Message: candidate.Term})
	}
	if candidate.Language != "pt-BR" {
		issues = append(issues, Issue{Code: "candidate_not_ptbr", Message: candidate.Language})
	}
	if candidate.CandidateStatus != "research_candidate" {
		issues = append(issues, Issue{Code: "candidate_status_not_research", Message: candidate.CandidateStatus})
	}
	if candidate.QualityState != "candidate_only" {
		issues = append(issues, Issue{Code: "candidate_quality_not_candidate_only", Message: candidate.QualityState})
	}
	if candidate.PracticeArea == "" {
		issues = append(issues, Issue{Code: "missing_practice_area", Message: candidate.TermID})
	}
	if candidate.DemandEvidenceType == "" || candidate.DemandEvidenceURL == "" {
		issues = append(issues, Issue{Code: "missing_demand_evidence", Message: candidate.TermID})
	} else if candidate.DemandEvidenceType != "google_trends_directional" || !isSafeDemandURL(candidate.DemandEvidenceURL) {
		issues = append(issues, Issue{Code: "unsafe_demand_evidence", Message: candidate.DemandEvidenceURL})
	}
	if candidate.DemandQueryGroup == "" {
		issues = append(issues, Issue{Code: "missing_demand_query_group", Message: candidate.TermID})
	}
	if candidate.OfficialSourceID == "" || candidate.OfficialSourceURL == "" {
		issues = append(issues, Issue{Code: "missing_official_source", Message: candidate.TermID})
	} else if !isOfficialSourceURL(candidate.OfficialSourceURL) {
		issues = append(issues, Issue{Code: "unsafe_official_source", Message: candidate.OfficialSourceURL})
	}
	if candidate.OnlineServiceMode != "digital_only" {
		issues = append(issues, Issue{Code: "not_digital_only", Message: candidate.TermID + ":" + candidate.OnlineServiceMode})
	}
	if candidate.WhatsAppCTAIntent != "high" {
		issues = append(issues, Issue{Code: "whatsapp_intent_not_high", Message: candidate.TermID + ":" + candidate.WhatsAppCTAIntent})
	}
	if len(strings.Fields(candidate.IntentReason)) < 8 {
		issues = append(issues, Issue{Code: "thin_intent_reason", Message: candidate.TermID})
	}
	if len(strings.Fields(candidate.DigitalServiceReason)) < 8 {
		issues = append(issues, Issue{Code: "thin_digital_service_reason", Message: candidate.TermID})
	}
	if candidate.RiskLevel != "medium" && candidate.RiskLevel != "high" {
		issues = append(issues, Issue{Code: "invalid_risk_level", Message: candidate.TermID + ":" + candidate.RiskLevel})
	}
	if candidate.CheckedAt == "" {
		issues = append(issues, Issue{Code: "candidate_without_checked_at", Message: candidate.TermID})
	}
	if candidate.PublicationAllowed {
		issues = append(issues, Issue{Code: "candidate_publication_allowed", Message: candidate.TermID})
	}
	if candidate.PublicPath != "" {
		issues = append(issues, Issue{Code: "candidate_has_public_path", Message: candidate.TermID + ":" + candidate.PublicPath})
	}
	return Report{Issues: issues}
}

func isSafeDemandURL(value string) bool {
	return strings.HasPrefix(value, "https://trends.google.com.br/trends/explore") ||
		strings.HasPrefix(value, "https://trends.google.com/trends/explore")
}

func isOfficialSourceURL(value string) bool {
	prefixes := []string{
		"https://www.gov.br/",
		"https://www.cnj.jus.br/",
		"https://datajud-wiki.cnj.jus.br/",
		"https://dadosabertos.camara.leg.br/",
		"https://dadosabertos.web.stj.jus.br/",
		"https://legis.senado.leg.br/",
		"https://www.planalto.gov.br/",
	}
	for _, prefix := range prefixes {
		if strings.HasPrefix(value, prefix) {
			return true
		}
	}
	return false
}

func looksLikePortugueseTerm(value string) bool {
	value = strings.TrimSpace(value)
	if value == "" || len(value) > 140 {
		return false
	}
	letters := 0
	for _, r := range value {
		if unicode.IsLetter(r) {
			letters++
			continue
		}
		if unicode.IsDigit(r) || unicode.IsSpace(r) || r == '-' || r == '/' {
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

package provenance

import (
	"fmt"
	"regexp"

	"portaljuridico/internal/sources"
)

var sha256Pattern = regexp.MustCompile(`^[a-f0-9]{64}$`)

type PayloadRecord struct {
	SourceID       string
	OfficialURL    string
	RetrievedAt    string
	PayloadSHA256  string
	RobotsSnapshot string
	TermsSnapshot  string
	Fields         []string
	UsePurpose     string
}

type Issue struct {
	Code    string
	Message string
}

type Report struct {
	Issues []Issue
}

func ValidatePayloadRecord(record PayloadRecord, registry sources.Registry) Report {
	issues := make([]Issue, 0)
	source, ok := registry.ByID(record.SourceID)
	if record.SourceID == "" {
		issues = append(issues, Issue{Code: "missing_source_id", Message: "registro sem fonte"})
	} else if !ok {
		issues = append(issues, Issue{Code: "unknown_source_id", Message: record.SourceID})
	}
	if record.OfficialURL == "" {
		issues = append(issues, Issue{Code: "missing_official_url", Message: "registro sem URL oficial"})
	}
	if record.RetrievedAt == "" {
		issues = append(issues, Issue{Code: "missing_retrieved_at", Message: "registro sem data de acesso"})
	}
	if record.PayloadSHA256 == "" {
		issues = append(issues, Issue{Code: "missing_payload_hash", Message: "registro sem hash SHA-256"})
	} else if !sha256Pattern.MatchString(record.PayloadSHA256) {
		issues = append(issues, Issue{Code: "invalid_payload_hash", Message: "hash SHA-256 invalido"})
	}
	if record.RobotsSnapshot == "" {
		issues = append(issues, Issue{Code: "missing_robots_snapshot", Message: "registro sem snapshot robots"})
	}
	if record.TermsSnapshot == "" {
		issues = append(issues, Issue{Code: "missing_terms_snapshot", Message: "registro sem snapshot termos"})
	}
	if len(record.Fields) == 0 {
		issues = append(issues, Issue{Code: "missing_fields", Message: "registro sem campos"})
	}
	if record.UsePurpose == "" {
		issues = append(issues, Issue{Code: "missing_use_purpose", Message: "registro sem finalidade de uso"})
	}
	if ok && (source.RobotsURL == "" || source.TermsURL == "" || source.ProvenanceStrategy == "") {
		issues = append(issues, Issue{Code: "source_missing_audit_contract", Message: record.SourceID})
	}
	return Report{Issues: issues}
}

func (r Report) Passed() bool {
	return len(r.Issues) == 0
}

func (r Report) Codes() []string {
	codes := make([]string, 0, len(r.Issues))
	for _, issue := range r.Issues {
		codes = append(codes, issue.Code)
	}
	return codes
}

func (r Report) HasIssue(code string) bool {
	for _, issue := range r.Issues {
		if issue.Code == code {
			return true
		}
	}
	return false
}

func (r Report) Messages() []string {
	messages := make([]string, 0, len(r.Issues))
	for _, issue := range r.Issues {
		messages = append(messages, fmt.Sprintf("%s: %s", issue.Code, issue.Message))
	}
	return messages
}

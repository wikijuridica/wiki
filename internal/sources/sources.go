package sources

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type Source struct {
	SourceID           string `json:"source_id"`
	Name               string `json:"name"`
	Organization       string `json:"organization"`
	BaseURL            string `json:"base_url"`
	ResearchURL        string `json:"research_url"`
	DataType           string `json:"data_type"`
	DocumentationPath  string `json:"documentation_path"`
	AccessMode         string `json:"access_mode"`
	IngestionEnabled   bool   `json:"ingestion_enabled"`
	AuditStatus        string `json:"audit_status"`
	RobotsURL          string `json:"robots_url"`
	RobotsStatus       string `json:"robots_status"`
	RobotsCheckedAt    string `json:"robots_checked_at"`
	TermsURL           string `json:"terms_url"`
	TermsStatus        string `json:"terms_status"`
	TermsCheckedAt     string `json:"terms_checked_at"`
	AuditDecision      string `json:"audit_decision"`
	AuditNote          string `json:"audit_note"`
	ProvenanceStrategy string `json:"provenance_strategy"`
	PrivacyRisk        string `json:"privacy_risk"`
	NoSignupRequired   bool   `json:"no_signup_required"`
}

type Registry struct {
	Sources []Source `json:"sources"`
	Root    string   `json:"-"`
}

func (r Registry) RegistryPath() string {
	return filepath.Join(r.Root, "content", "source_registry.json")
}

type Issue struct {
	SourceID string
	Code     string
	Message  string
}

type Report struct {
	Issues []Issue
}

func LoadRegistry(root string) (Registry, error) {
	projectRoot, err := findProjectRoot(root)
	if err != nil {
		return Registry{}, err
	}
	data, err := os.ReadFile(filepath.Join(projectRoot, "content", "source_registry.json"))
	if err != nil {
		return Registry{}, err
	}
	var registry Registry
	if err := json.Unmarshal(data, &registry); err != nil {
		return Registry{}, err
	}
	registry.Root = projectRoot
	return registry, nil
}

func (r Registry) ByID(sourceID string) (Source, bool) {
	for _, source := range r.Sources {
		if source.SourceID == sourceID {
			return source, true
		}
	}
	return Source{}, false
}

func (r Registry) AnyIngestionEnabled() bool {
	for _, source := range r.Sources {
		if source.IngestionEnabled {
			return true
		}
	}
	return false
}

func (r Registry) ValidateForP0() Report {
	issues := make([]Issue, 0)
	seen := make(map[string]bool)
	for _, source := range r.Sources {
		if source.SourceID == "" {
			issues = append(issues, Issue{Code: "missing_source_id", Message: "fonte sem identificador"})
			continue
		}
		if seen[source.SourceID] {
			issues = append(issues, Issue{SourceID: source.SourceID, Code: "duplicate_source_id", Message: "fonte duplicada"})
		}
		seen[source.SourceID] = true
		if source.Name == "" || source.Organization == "" || source.BaseURL == "" || source.ResearchURL == "" || source.DataType == "" {
			issues = append(issues, Issue{SourceID: source.SourceID, Code: "incomplete_source", Message: "fonte sem metadados minimos"})
		}
		if source.DocumentationPath == "" {
			issues = append(issues, Issue{SourceID: source.SourceID, Code: "missing_documentation_path", Message: "fonte sem documento"})
		} else if _, err := os.Stat(filepath.Join(r.Root, source.DocumentationPath)); err != nil {
			issues = append(issues, Issue{SourceID: source.SourceID, Code: "missing_documentation_file", Message: source.DocumentationPath})
		}
		if source.IngestionEnabled {
			issues = append(issues, Issue{SourceID: source.SourceID, Code: "ingestion_enabled_during_p0", Message: "ingestao deve ficar bloqueada no P0"})
		}
		if source.RobotsURL == "" || source.RobotsStatus == "" {
			issues = append(issues, Issue{SourceID: source.SourceID, Code: "missing_robots_audit", Message: "fonte sem auditoria robots"})
		}
		if source.RobotsCheckedAt == "" {
			issues = append(issues, Issue{SourceID: source.SourceID, Code: "missing_robots_checked_at", Message: "fonte sem data de auditoria robots"})
		}
		if isPendingAudit(source.RobotsStatus) {
			issues = append(issues, Issue{SourceID: source.SourceID, Code: "robots_audit_still_pending", Message: "robots pendente nao pode ser mascarado como entrega"})
		}
		if source.TermsURL == "" || source.TermsStatus == "" {
			issues = append(issues, Issue{SourceID: source.SourceID, Code: "missing_terms_audit", Message: "fonte sem auditoria de termos"})
		}
		if source.TermsCheckedAt == "" {
			issues = append(issues, Issue{SourceID: source.SourceID, Code: "missing_terms_checked_at", Message: "fonte sem data de auditoria de termos"})
		}
		if isPendingAudit(source.TermsStatus) {
			issues = append(issues, Issue{SourceID: source.SourceID, Code: "terms_audit_still_pending", Message: "termos pendentes nao podem ser mascarados como entrega"})
		}
		if source.AuditDecision == "" || source.AuditNote == "" {
			issues = append(issues, Issue{SourceID: source.SourceID, Code: "missing_audit_decision", Message: "fonte sem decisao e nota de auditoria"})
		}
		if source.ProvenanceStrategy == "" {
			issues = append(issues, Issue{SourceID: source.SourceID, Code: "missing_provenance_strategy", Message: "fonte sem estrategia de proveniencia"})
		}
		if source.PrivacyRisk == "" {
			issues = append(issues, Issue{SourceID: source.SourceID, Code: "missing_privacy_risk", Message: "fonte sem risco de privacidade"})
		}
	}
	return Report{Issues: issues}
}

func isPendingAudit(value string) bool {
	return value == "" || strings.Contains(value, "pendente")
}

func (s Source) ApprovedForIndexableLegalContent() bool {
	return s.AuditStatus == "approved" && !s.IngestionEnabled
}

func (r Report) Passed() bool {
	return len(r.Issues) == 0
}

func (r Report) Messages() []string {
	messages := make([]string, 0, len(r.Issues))
	for _, issue := range r.Issues {
		messages = append(messages, fmt.Sprintf("%s: %s: %s", issue.SourceID, issue.Code, issue.Message))
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

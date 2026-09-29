package cta

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"portaljuridico/internal/content"
	"portaljuridico/internal/editorial"
	"portaljuridico/internal/sources"
)

type Policy struct {
	Enabled                      bool     `json:"enabled"`
	Channel                      string   `json:"channel"`
	PhonePlaceholder             string   `json:"phone_placeholder"`
	Intent                       string   `json:"intent"`
	RequiresContextMessage       bool     `json:"requires_context_message"`
	ContextMessageTemplate       string   `json:"context_message_template"`
	ContextOriginParameters      []string `json:"context_origin_parameters"`
	RequiresApprovedLegalContent bool     `json:"requires_approved_legal_content"`
	RequiresSourceProvenance     bool     `json:"requires_source_provenance"`
	RequiresEditorialReview      bool     `json:"requires_editorial_review"`
	AllowedPageTypes             []string `json:"allowed_page_types"`
	BlockedStatuses              []string `json:"blocked_statuses"`
}

func LoadPolicy(root string) (Policy, error) {
	projectRoot, err := findProjectRoot(root)
	if err != nil {
		return Policy{}, err
	}
	data, err := os.ReadFile(filepath.Join(projectRoot, "content", "cta_policy.json"))
	if err != nil {
		return Policy{}, err
	}
	var policy Policy
	if err := json.Unmarshal(data, &policy); err != nil {
		return Policy{}, err
	}
	return policy, nil
}

func CanRender(policy Policy, page content.Page, registry sources.Registry) bool {
	if !policy.Enabled || policy.Channel != "whatsapp" {
		return false
	}
	if !editorial.IsIndexable(page) {
		return false
	}
	if !allowedPageType(policy, page.PageType) {
		return false
	}
	if blockedStatus(policy, page.Status) {
		return false
	}
	if policy.RequiresEditorialReview && (page.Reviewer == "" || page.ReviewedAt == "") {
		return false
	}
	if policy.RequiresSourceProvenance && len(page.SourceProvenance) == 0 {
		return false
	}
	if policy.RequiresApprovedLegalContent && page.IsLegalContent() {
		for _, provenance := range page.SourceProvenance {
			source, ok := registry.ByID(provenance.SourceID)
			if !ok || source.AuditStatus != "approved" || source.IngestionEnabled {
				return false
			}
		}
	}
	if policy.RequiresContextMessage && ContextMessage(policy, page) == "" {
		return false
	}
	return true
}

func ContextMessage(policy Policy, page content.Page) string {
	template := strings.TrimSpace(policy.ContextMessageTemplate)
	if template == "" {
		return ""
	}
	message := strings.NewReplacer(
		"{path}", page.Path,
		"{unique_intent_id}", page.UniqueIntentID,
		"{title}", page.Title,
		"{intent}", policy.Intent,
	).Replace(template)
	if strings.Contains(message, "{") || strings.Contains(message, "}") {
		return ""
	}
	if !strings.Contains(message, page.Path) || !strings.Contains(message, page.UniqueIntentID) || !strings.Contains(message, page.Title) {
		return ""
	}
	return message
}

func allowedPageType(policy Policy, pageType string) bool {
	for _, allowed := range policy.AllowedPageTypes {
		if allowed == pageType {
			return true
		}
	}
	return false
}

func blockedStatus(policy Policy, status string) bool {
	for _, blocked := range policy.BlockedStatuses {
		if blocked == status {
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

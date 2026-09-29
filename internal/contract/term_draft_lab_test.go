package contract_test

import (
	"strings"
	"testing"

	"portaljuridico/internal/draftlab"
	"portaljuridico/internal/quality"
	"portaljuridico/internal/terms"
)

func TestTermSeedDraftLabCreatesNaturalNoindexDraftOnly(t *testing.T) {
	seed := terms.Seed{
		TermID:       "responsabilidade-civil",
		Term:         "responsabilidade civil",
		Language:     "pt-BR",
		QualityState: "draft_only",
		SourceID:     "lexml",
		SourceURL:    "https://www.lexml.gov.br/",
		CheckedAt:    "2026-06-09",
		IntentHint:   "explicar o conceito juridico em linguagem informativa antes de qualquer publicacao",
	}

	draft, err := draftlab.Build(seed)
	if err != nil {
		t.Fatal(err)
	}
	if draft.Status != "draft" || draft.IndexPolicy != "noindex" {
		t.Fatalf("draft state = %s/%s, want draft/noindex", draft.Status, draft.IndexPolicy)
	}
	if draft.PublicPath != "" {
		t.Fatalf("draft has public path %q, want no public path", draft.PublicPath)
	}
	for _, token := range []string{"responsabilidade civil", "LexML", "não substitui consulta jurídica individual"} {
		if !strings.Contains(draft.Text, token) {
			t.Fatalf("draft text missing %q:\n%s", token, draft.Text)
		}
	}
	analysis := quality.AnalyzeText(draft.Text)
	if !analysis.Passed() {
		t.Fatalf("draft quality failed: %v", analysis.Messages())
	}
}

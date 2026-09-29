package contract_test

import (
	"testing"

	"portaljuridico/internal/terms"
)

func TestTermSeedsStayDraftOnlyWithSourceAndQualityState(t *testing.T) {
	report := terms.ValidateSeeds(".")

	if !report.Passed() {
		t.Fatalf("term seeds failed P0 contract: %v", report.Messages())
	}
}

func TestTermSeedRejectsPublishedOrUnsourcedSeed(t *testing.T) {
	seed := terms.Seed{
		TermID:       "responsabilidade-civil",
		Term:         "responsabilidade civil",
		Language:     "pt-BR",
		QualityState: "published",
		SourceID:     "",
		SourceURL:    "",
		CheckedAt:    "",
		IntentHint:   "explicar o tema juridico em linguagem informativa",
	}

	report := terms.ValidateSeed(seed)
	if report.Passed() {
		t.Fatal("ValidateSeed passed, want draft/source failures")
	}
	for _, code := range []string{"term_seed_not_draft_only", "term_seed_without_source", "term_seed_without_checked_at"} {
		if !report.HasIssue(code) {
			t.Fatalf("missing issue %q in %v", code, report.Codes())
		}
	}
}

package contract_test

import (
	"path/filepath"
	"strings"
	"testing"

	"portaljuridico/internal/draftlab"
	"portaljuridico/internal/reviewqueue"
	"portaljuridico/internal/terms"
)

func TestReviewQueueKeepsDraftNeedsReviewAndNotPublishable(t *testing.T) {
	root := copyStorageFixture(t)
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

	inserted, err := reviewqueue.EnqueueDraft(root, draft, reviewqueue.Metadata{
		Author:    "Redacao tecnica",
		CreatedAt: "2026-06-09",
		Reason:    "Rascunho de laboratorio pronto para revisao editorial, ainda sem publicacao.",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !inserted {
		t.Fatal("first enqueue should insert review record")
	}

	report := reviewqueue.Validate(root)
	if !report.Passed() {
		t.Fatalf("review queue validation failed: %v", report.Messages())
	}
	data := readFile(t, filepath.Join(root, "data", "editorial", "review_queue.jsonl"))
	for _, token := range []string{`"review_status":"needs_review"`, `"publication_allowed":false`, `"public_path":""`, `"author":"Redacao tecnica"`, `"event":"queued_for_review"`} {
		if !strings.Contains(data, token) {
			t.Fatalf("review queue missing %q in %s", token, data)
		}
	}
}

func TestReviewQueueRejectsPublishableRecord(t *testing.T) {
	record := reviewqueue.Record{
		TermID:             "responsabilidade-civil",
		DraftStatus:        "draft",
		IndexPolicy:        "noindex",
		ReviewStatus:       "needs_review",
		PublicationAllowed: true,
		PublicPath:         "/wiki/responsabilidade-civil/",
		SourceID:           "lexml",
		Author:             "Redacao tecnica",
		History: []reviewqueue.Event{
			{Event: "queued_for_review", At: "2026-06-09", Actor: "codex", Note: "teste"},
		},
	}

	report := reviewqueue.ValidateRecord(record)
	if report.Passed() {
		t.Fatal("ValidateRecord passed, want publish/public path failures")
	}
	for _, code := range []string{"review_queue_publication_allowed", "review_queue_has_public_path"} {
		if !report.HasIssue(code) {
			t.Fatalf("missing issue %q in %v", code, report.Codes())
		}
	}
}

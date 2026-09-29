package contract_test

import (
	"path/filepath"
	"strings"
	"testing"

	"portaljuridico/internal/approvals"
	"portaljuridico/internal/reviewqueue"
)

func TestEditorialApprovalKeepsPublicationBlocked(t *testing.T) {
	root := copyStorageFixture(t)
	queued := reviewqueue.Record{
		TermID:             "responsabilidade-civil",
		Term:               "responsabilidade civil",
		DraftStatus:        "draft",
		IndexPolicy:        "noindex",
		ReviewStatus:       "needs_review",
		PublicationAllowed: false,
		PublicPath:         "",
		SourceID:           "lexml",
		SourceURL:          "https://www.lexml.gov.br/",
		Author:             "Redacao tecnica",
		Reviewer:           "pending",
		Reason:             "Rascunho aguardando revisao.",
		History: []reviewqueue.Event{
			{Event: "queued_for_review", At: "2026-06-09", Actor: "codex", Note: "fila"},
		},
	}

	inserted, err := approvals.Approve(root, queued, approvals.Metadata{
		Reviewer:   "Revisao juridica",
		ApprovedAt: "2026-06-09",
		Reason:     "Aprovacao editorial de laboratorio; publicacao ainda bloqueada.",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !inserted {
		t.Fatal("first approval should insert approval record")
	}

	report := approvals.Validate(root)
	if !report.Passed() {
		t.Fatalf("approval validation failed: %v", report.Messages())
	}
	data := readFile(t, filepath.Join(root, "data", "editorial", "approved_drafts.jsonl"))
	for _, token := range []string{`"editorial_status":"approved"`, `"publication_allowed":false`, `"public_path":""`, `"reviewer":"Revisao juridica"`, `"event":"approved_editorial_only"`} {
		if !strings.Contains(data, token) {
			t.Fatalf("approval store missing %q in %s", token, data)
		}
	}
}

func TestEditorialApprovalRejectsPublicPathOrPublicationAllowed(t *testing.T) {
	record := approvals.Record{
		TermID:             "responsabilidade-civil",
		Term:               "responsabilidade civil",
		EditorialStatus:    "approved",
		IndexPolicy:        "noindex",
		PublicationAllowed: true,
		PublicPath:         "/wiki/responsabilidade-civil/",
		SourceID:           "lexml",
		SourceURL:          "https://www.lexml.gov.br/",
		Author:             "Redacao tecnica",
		Reviewer:           "Revisao juridica",
		ApprovedAt:         "2026-06-09",
		Reason:             "teste",
		History: []approvals.Event{
			{Event: "approved_editorial_only", At: "2026-06-09", Actor: "Revisao juridica", Note: "teste"},
		},
	}

	report := approvals.ValidateRecord(record)
	if report.Passed() {
		t.Fatal("ValidateRecord passed, want publication failures")
	}
	for _, code := range []string{"approval_publication_allowed", "approval_has_public_path"} {
		if !report.HasIssue(code) {
			t.Fatalf("missing issue %q in %v", code, report.Codes())
		}
	}
}

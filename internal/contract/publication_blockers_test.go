package contract_test

import (
	"path/filepath"
	"strings"
	"testing"

	"portaljuridico/internal/approvals"
	"portaljuridico/internal/publicationblockers"
)

func TestPublicationBlockerManifestKeepsApprovedDraftOutOfPublicRoutes(t *testing.T) {
	root := copyStorageFixture(t)
	approval := approvals.Record{
		TermID:             "responsabilidade-civil",
		Term:               "responsabilidade civil",
		EditorialStatus:    "approved",
		IndexPolicy:        "noindex",
		PublicationAllowed: false,
		PublicPath:         "",
		SourceID:           "lexml",
		SourceURL:          "https://www.lexml.gov.br/",
		Author:             "Redacao tecnica",
		Reviewer:           "Revisao juridica",
		ApprovedAt:         "2026-06-09",
		Reason:             "Aprovacao editorial de laboratorio; publicacao bloqueada.",
		History: []approvals.Event{
			{Event: "approved_editorial_only", At: "2026-06-09", Actor: "Revisao juridica", Note: "teste"},
		},
	}

	inserted, err := publicationblockers.AppendFromApproval(root, approval, "2026-06-09")
	if err != nil {
		t.Fatal(err)
	}
	if !inserted {
		t.Fatal("first blocker insert should write a manifest record")
	}

	report := publicationblockers.Validate(root)
	if !report.Passed() {
		t.Fatalf("publication blocker validation failed: %v", report.Messages())
	}
	data := readFile(t, filepath.Join(root, "data", "editorial", "publication_blockers.jsonl"))
	for _, token := range []string{`"publication_status":"blocked"`, `"publication_allowed":false`, `"public_path":""`, `"specific_source_research"`, `"legal_reviewer_approval"`} {
		if !strings.Contains(data, token) {
			t.Fatalf("publication blocker missing %q in %s", token, data)
		}
	}
}

func TestPublicationBlockerRejectsEmptyMissingRequirements(t *testing.T) {
	record := publicationblockers.Record{
		TermID:              "responsabilidade-civil",
		PublicationStatus:   "blocked",
		PublicationAllowed:  false,
		PublicPath:          "",
		MissingRequirements: []string{},
		CheckedAt:           "2026-06-09",
	}
	report := publicationblockers.ValidateRecord(record)
	if report.Passed() {
		t.Fatal("ValidateRecord passed, want missing requirements failure")
	}
	if !report.HasIssue("publication_blocker_without_requirements") {
		t.Fatalf("missing publication_blocker_without_requirements in %v", report.Codes())
	}
}

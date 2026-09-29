package contract_test

import (
	"testing"

	"portaljuridico/internal/batchcandidatereviews"
	"portaljuridico/internal/batchprepublication"
)

func TestBatchPrepublicationGatesCoverReviewedCandidatesWithoutPublishing(t *testing.T) {
	report := batchprepublication.Validate(".")
	if !report.Passed() {
		t.Fatalf("batch prepublication gates failed contract: %v", report.Messages())
	}

	records, loadReport := batchprepublication.LoadRecords(".")
	if !loadReport.Passed() {
		t.Fatalf("could not load batch prepublication gates: %v", loadReport.Messages())
	}
	reviewIndex, indexReport := batchcandidatereviews.BuildCandidateIndex(".")
	if !indexReport.Passed() {
		t.Fatalf("could not build selected candidate index: %v", indexReport.Messages())
	}
	if len(records) != len(reviewIndex.SelectedByIntent) {
		t.Fatalf("prepublication gates=%d selected=%d, want one blocked gate for each selected batch candidate", len(records), len(reviewIndex.SelectedByIntent))
	}
	if len(records) < 168 {
		t.Fatalf("prepublication gates=%d, want at least 168 expanded paid-passed candidates", len(records))
	}

	for _, entry := range records {
		record := entry.Record
		if record.CandidateRobots != "noindex,follow" {
			t.Fatalf("line=%d robots=%q, want noindex,follow", entry.Line, record.CandidateRobots)
		}
		if record.RenderAllowed || record.SitemapAllowed || record.PublicationAllowed || record.PublicPath != "" {
			t.Fatalf("line=%d gate escaped blocked contract: render=%t sitemap=%t publication=%t path=%q", entry.Line, record.RenderAllowed, record.SitemapAllowed, record.PublicationAllowed, record.PublicPath)
		}
		if !record.CanonicalUsesOfficialBase("https://wikijuridica.com.br") {
			t.Fatalf("line=%d canonical=%q does not use official base and candidate path %q", entry.Line, record.CandidateCanonicalURL, record.CandidatePath)
		}
		if record.SourceSpecificityStatus != "matrix_audited_final_source_pending" {
			t.Fatalf("line=%d source status=%q, want matrix_audited_final_source_pending", entry.Line, record.SourceSpecificityStatus)
		}
	}
}

func TestBatchPrepublicationGateRejectsPublicOrStaleCanonical(t *testing.T) {
	index := batchprepublication.ReviewIndex{
		BaseURL: "https://wikijuridica.com.br",
		ReviewedByIntent: map[string]batchprepublication.ReviewedCandidate{
			"familia-divorcio-consensual-filhos-bens": {
				ReviewID:       "review-familia-divorcio-consensual-filhos-bens",
				GateID:         "candidate-familia-digital",
				BatchID:        "batch-familia-digital",
				UniqueIntentID: "familia-divorcio-consensual-filhos-bens",
				CandidatePath:  "/temas/familia-divorcio-consensual-filhos-bens/",
				SourceMatrixID: "familia-divorcio-consensual-filhos-bens",
				Term:           "divórcio consensual online com filhos e bens",
			},
		},
	}

	record := batchprepublication.Record{
		PrepublicationID:         "bad-gate",
		ReviewID:                 "review-familia-divorcio-consensual-filhos-bens",
		GateID:                   "candidate-familia-digital",
		BatchID:                  "batch-familia-digital",
		UniqueIntentID:           "familia-divorcio-consensual-filhos-bens",
		SourceMatrixID:           "fonte-nao-auditada",
		Term:                     "divórcio consensual online com filhos e bens",
		CandidatePath:            "/temas/familia-divorcio-consensual-filhos-bens/?utm=1",
		CandidateCanonicalURL:    "https://portal-juridico.example/temas/familia-divorcio-consensual-filhos-bens/",
		CandidateRobots:          "index,follow",
		CandidateTitle:           "Divórcio",
		CandidateMetaDescription: "Meta curta.",
		SourceSpecificityStatus:  "",
		RemainingGates:           []string{},
		RenderAllowed:            true,
		SitemapAllowed:           true,
		PublicationAllowed:       true,
		PublicPath:               "/temas/familia-divorcio-consensual-filhos-bens/",
		CheckedAt:                "2026-06-09",
	}

	report := batchprepublication.ValidateRecordAgainstReviewIndex(record, index)
	if report.Passed() {
		t.Fatal("ValidateRecordAgainstReviewIndex passed, want blocked prepublication failures")
	}
	for _, code := range []string{
		"batch_prepublication_candidate_path_not_clean",
		"batch_prepublication_canonical_mismatch",
		"batch_prepublication_not_noindex",
		"batch_prepublication_title_too_short",
		"batch_prepublication_meta_too_short",
		"batch_prepublication_source_matrix_mismatch",
		"batch_prepublication_source_specificity_not_pending",
		"batch_prepublication_missing_remaining_gates",
		"batch_prepublication_render_allowed",
		"batch_prepublication_sitemap_allowed",
		"batch_prepublication_publication_allowed",
		"batch_prepublication_has_public_path",
	} {
		if !report.HasIssue(code) {
			t.Fatalf("missing issue %q in %v", code, report.Codes())
		}
	}
}

package contract_test

import (
	"testing"

	"portaljuridico/internal/batchfinaldrafts"
	"portaljuridico/internal/batchpublicmanifest"
)

func TestBatchPublicManifestGatesCoverSourceSpecificityWithoutPublishing(t *testing.T) {
	report := batchpublicmanifest.Validate(".")
	if !report.Passed() {
		t.Fatalf("batch public manifest gates failed contract: %v", report.Messages())
	}

	records, loadReport := batchpublicmanifest.LoadRecords(".")
	if !loadReport.Passed() {
		t.Fatalf("could not load batch public manifest gates: %v", loadReport.Messages())
	}
	index, indexReport := batchpublicmanifest.BuildManifestIndex(".")
	if !indexReport.Passed() {
		t.Fatalf("could not build public manifest index: %v", indexReport.Messages())
	}
	if len(records) != len(index.SourceByIntent) {
		t.Fatalf("public manifest gates=%d source_resolutions=%d, want one blocked gate for each source-specificity resolution", len(records), len(index.SourceByIntent))
	}
	if len(records) < 168 {
		t.Fatalf("public manifest gates=%d, want at least 168 expanded paid-passed candidates", len(records))
	}

	finalDrafts, finalLoadReport := batchfinaldrafts.LoadRecords(".")
	if !finalLoadReport.Passed() {
		t.Fatalf("could not load final drafts for manifest lock count: %v", finalLoadReport.Messages())
	}
	sourceLocked := 0
	sourceBlocked := 0
	for _, entry := range records {
		record := entry.Record
		if record.CandidateRobots != "noindex,follow" || record.IndexPolicy != "noindex" {
			t.Fatalf("line=%d robots=%q index_policy=%q", entry.Line, record.CandidateRobots, record.IndexPolicy)
		}
		if record.UsePolicy != "reference_only_no_scraping_no_ingestion" {
			t.Fatalf("line=%d use_policy=%q", entry.Line, record.UsePolicy)
		}
		if record.ManifestAllowed || record.RenderAllowed || record.SitemapAllowed || record.PublicationAllowed || record.PublicPath != "" {
			t.Fatalf("line=%d gate escaped blocked contract: manifest=%t render=%t sitemap=%t publication=%t path=%q", entry.Line, record.ManifestAllowed, record.RenderAllowed, record.SitemapAllowed, record.PublicationAllowed, record.PublicPath)
		}
		switch record.ManifestGateStatus {
		case batchpublicmanifest.SEOReviewPendingStatus:
			sourceLocked++
			if record.SourceSpecificityStatus != "final_source_locked_reference_only" || !record.SEOReviewRequired || !record.ContentDraftRequired {
				t.Fatalf("line=%d SEO pending gate must require locked source, SEO review and content draft", entry.Line)
			}
		case batchpublicmanifest.SourceBlockedStatus:
			sourceBlocked++
			if record.SourceSpecificityStatus != "final_source_blocked_needs_specific_url" || record.BlockingReason == "" || record.NeededSourceDetail == "" {
				t.Fatalf("line=%d source-blocked gate must keep source blocker details", entry.Line)
			}
		default:
			t.Fatalf("line=%d invalid manifest status=%q", entry.Line, record.ManifestGateStatus)
		}
	}
	if sourceLocked != len(finalDrafts) {
		t.Fatalf("source_locked=%d final_drafts=%d, want locked manifest only for eligible final drafts", sourceLocked, len(finalDrafts))
	}
	if sourceLocked+sourceBlocked != len(records) {
		t.Fatalf("source_locked=%d source_blocked=%d records=%d, manifest statuses must cover every record", sourceLocked, sourceBlocked, len(records))
	}
}

func TestBatchPublicManifestGateRejectsPublicOrSourceBlockedManifest(t *testing.T) {
	index := batchpublicmanifest.ManifestIndex{
		BaseURL: "https://wikijuridica.com.br",
		SourceByIntent: map[string]batchpublicmanifest.SourceResolution{
			"familia-divorcio-consensual-filhos-bens": {
				ResolutionID:            "source-specificity-familia-divorcio-consensual-filhos-bens",
				PrepublicationID:        "prepub-familia-divorcio-consensual-filhos-bens",
				ReviewID:                "review-familia-divorcio-consensual-filhos-bens",
				BatchID:                 "batch-familia-digital",
				UniqueIntentID:          "familia-divorcio-consensual-filhos-bens",
				SourceMatrixID:          "familia-divorcio-consensual-filhos-bens",
				Term:                    "divórcio consensual online com filhos e bens",
				CandidatePath:           "/temas/familia-divorcio-consensual-filhos-bens/",
				CandidateCanonicalURL:   "https://wikijuridica.com.br/temas/familia-divorcio-consensual-filhos-bens/",
				CandidateRobots:         "noindex,follow",
				SourceSpecificityStatus: "final_source_blocked_needs_specific_url",
				SelectedSourceURLs:      []string{"https://www.planalto.gov.br/ccivil_03/leis/2002/l10406compilada.htm"},
				BlockingReason:          "fonte ampla",
				NeededSourceDetail:      "fonte especifica",
			},
		},
		PrepublicationByIntent: map[string]batchpublicmanifest.PrepublicationSEO{
			"familia-divorcio-consensual-filhos-bens": {
				CandidateTitle:           "Divórcio consensual online com filhos e bens",
				CandidateMetaDescription: "Saiba quais documentos e pontos de consenso precisam ser organizados antes da análise jurídica digital.",
			},
		},
	}

	record := batchpublicmanifest.Record{
		ManifestGateID:           "bad-public-manifest",
		SourceResolutionID:       "source-specificity-familia-divorcio-consensual-filhos-bens",
		PrepublicationID:         "prepub-familia-divorcio-consensual-filhos-bens",
		ReviewID:                 "review-familia-divorcio-consensual-filhos-bens",
		BatchID:                  "batch-familia-digital",
		UniqueIntentID:           "familia-divorcio-consensual-filhos-bens",
		SourceMatrixID:           "familia-divorcio-consensual-filhos-bens",
		Term:                     "divórcio consensual online com filhos e bens",
		CandidatePath:            "/temas/familia-divorcio-consensual-filhos-bens/?utm=1",
		CandidateCanonicalURL:    "https://portal-juridico.example/temas/familia-divorcio-consensual-filhos-bens/",
		CandidateRobots:          "index,follow",
		CandidateTitle:           "Divórcio",
		CandidateMetaDescription: "Meta curta.",
		SourceSpecificityStatus:  "final_source_blocked_needs_specific_url",
		ManifestGateStatus:       batchpublicmanifest.SEOReviewPendingStatus,
		SelectedSourceURLs:       []string{"https://www.planalto.gov.br/ccivil_03/leis/2002/l10406compilada.htm"},
		IndexPolicy:              "index",
		UsePolicy:                "copy_or_scrape",
		SEOReviewRequired:        true,
		ContentDraftRequired:     true,
		ManifestAllowed:          true,
		RenderAllowed:            true,
		SitemapAllowed:           true,
		PublicationAllowed:       true,
		PublicPath:               "/temas/familia-divorcio-consensual-filhos-bens/",
		CheckedAt:                "2026-06-09",
	}

	report := batchpublicmanifest.ValidateRecordAgainstManifestIndex(record, index)
	if report.Passed() {
		t.Fatal("ValidateRecordAgainstManifestIndex passed, want blocked manifest failures")
	}
	for _, code := range []string{
		"batch_public_manifest_source_not_locked_for_seo",
		"batch_public_manifest_candidate_path_not_clean",
		"batch_public_manifest_canonical_mismatch",
		"batch_public_manifest_not_noindex",
		"batch_public_manifest_title_too_short",
		"batch_public_manifest_meta_too_short",
		"batch_public_manifest_invalid_index_policy",
		"batch_public_manifest_invalid_use_policy",
		"batch_public_manifest_manifest_allowed",
		"batch_public_manifest_render_allowed",
		"batch_public_manifest_sitemap_allowed",
		"batch_public_manifest_publication_allowed",
		"batch_public_manifest_has_public_path",
	} {
		if !report.HasIssue(code) {
			t.Fatalf("missing issue %q in %v", code, report.Codes())
		}
	}
}

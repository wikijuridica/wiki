package contract_test

import (
	"testing"

	"portaljuridico/internal/sourceresolutions"
)

func TestSourceSpecificityResolutionsStartWithOfficialSpecificSources(t *testing.T) {
	report := sourceresolutions.Validate(".")
	if !report.Passed() {
		t.Fatalf("source resolutions failed contract: %v", report.Messages())
	}

	records, loadReport := sourceresolutions.LoadRecords(".")
	if !loadReport.Passed() {
		t.Fatalf("could not load source resolutions: %v", loadReport.Messages())
	}

	var found *sourceresolutions.Record
	for _, entry := range records {
		if entry.Record.TermID == "negativa-cobertura-plano-saude" {
			record := entry.Record
			found = &record
			break
		}
	}
	if found == nil {
		t.Fatal("missing source resolution for negativa-cobertura-plano-saude")
	}
	if found.PublicationAllowed || found.PublicPath != "" || found.IndexPolicy != "noindex" {
		t.Fatalf("source resolution escaped pre-publication gate: publication=%t path=%q index=%q", found.PublicationAllowed, found.PublicPath, found.IndexPolicy)
	}
	if !found.HasSourceType("primary_law") {
		t.Fatalf("resolution missing primary_law source: %v", found.SourceTypes())
	}
	if !found.HasSourceType("coverage_rule") {
		t.Fatalf("resolution missing coverage_rule source: %v", found.SourceTypes())
	}
	if len(found.OfficialSources) < 4 {
		t.Fatalf("got %d official sources, want at least 4 specific official sources", len(found.OfficialSources))
	}
}

func TestSourceSpecificityResolutionRejectsBroadOrPublishingRecord(t *testing.T) {
	record := sourceresolutions.Record{
		TermID:           "negativa-cobertura-plano-saude",
		Term:             "negativa de cobertura do plano de saúde",
		ResolutionID:     "negativa-cobertura-plano-saude",
		ResolutionStatus: "source_resolved",
		Language:         "pt-BR",
		SourceBriefID:    "negativa-cobertura-plano-saude",
		OfficialSources: []sourceresolutions.OfficialSource{
			{
				URL:          "https://www.gov.br/",
				SourceType:   "institutional_home",
				UseInContent: "fonte ampla",
				CheckedAt:    "2026-06-09",
			},
		},
		SpecificityScore:   20,
		IndexPolicy:        "index",
		PublicationAllowed: true,
		PublicPath:         "/temas/negativa-cobertura-plano-saude/",
		CheckedAt:          "2026-06-09",
	}

	report := sourceresolutions.ValidateRecord(record)
	if report.Passed() {
		t.Fatal("ValidateRecord passed, want broad source and publication failures")
	}
	for _, code := range []string{
		"source_resolution_too_few_sources",
		"source_resolution_missing_primary_law",
		"source_resolution_missing_coverage_rule",
		"source_resolution_low_specificity",
		"source_resolution_publication_allowed",
		"source_resolution_has_public_path",
		"source_resolution_not_noindex",
		"source_resolution_broad_source",
	} {
		if !report.HasIssue(code) {
			t.Fatalf("missing issue %q in %v", code, report.Codes())
		}
	}
}

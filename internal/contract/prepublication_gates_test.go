package contract_test

import (
	"testing"

	"portaljuridico/internal/prepublication"
)

func TestPrepublicationGateReconcilesResolvedSourceWithoutPublishing(t *testing.T) {
	report := prepublication.Validate(".")
	if !report.Passed() {
		t.Fatalf("prepublication gates failed contract: %v", report.Messages())
	}

	records, loadReport := prepublication.LoadRecords(".")
	if !loadReport.Passed() {
		t.Fatalf("could not load prepublication gates: %v", loadReport.Messages())
	}

	var found *prepublication.Record
	for _, entry := range records {
		if entry.Record.TermID == "negativa-cobertura-plano-saude" {
			record := entry.Record
			found = &record
			break
		}
	}
	if found == nil {
		t.Fatal("missing prepublication gate for negativa-cobertura-plano-saude")
	}
	if found.PublicationAllowed || found.RenderAllowed || found.SitemapAllowed || found.PublicPath != "" {
		t.Fatalf("gate escaped blocked prepublication contract: publication=%t render=%t sitemap=%t public_path=%q", found.PublicationAllowed, found.RenderAllowed, found.SitemapAllowed, found.PublicPath)
	}
	if found.SourceBlockerReconciliation != "source_resolved_blocker_still_active" {
		t.Fatalf("source blocker reconciliation=%q, want source_resolved_blocker_still_active", found.SourceBlockerReconciliation)
	}
}

func TestPrepublicationGateRejectsIndexableOrWeakSearchContract(t *testing.T) {
	record := prepublication.Record{
		TermID:                      "negativa-cobertura-plano-saude",
		Term:                        "negativa de cobertura do plano de saúde",
		GateID:                      "negativa-cobertura-plano-saude",
		GateStatus:                  "blocked_prepublication",
		Language:                    "pt-BR",
		SourceResolutionID:          "missing-resolution",
		SourceBlockerReconciliation: "source_resolved",
		CandidatePath:               "/temas/negativa-cobertura-plano-saude/?utm=1",
		CandidateCanonicalURL:       "http://portal-juridico.example/temas/negativa-cobertura-plano-saude/",
		CandidateRobots:             "index,follow",
		CandidateTitle:              "Negativa",
		CandidateMetaDescription:    "Meta curta.",
		RenderAllowed:               true,
		SitemapAllowed:              true,
		PublicationAllowed:          true,
		PublicPath:                  "/temas/negativa-cobertura-plano-saude/",
		RemainingGates:              []string{},
		CheckedAt:                   "2026-06-09",
	}

	report := prepublication.ValidateRecord(record)
	if report.Passed() {
		t.Fatal("ValidateRecord passed, want prepublication failures")
	}
	for _, code := range []string{
		"prepublication_candidate_path_not_clean",
		"prepublication_canonical_not_https",
		"prepublication_not_noindex",
		"prepublication_title_too_short",
		"prepublication_meta_too_short",
		"prepublication_render_allowed",
		"prepublication_sitemap_allowed",
		"prepublication_publication_allowed",
		"prepublication_has_public_path",
		"prepublication_bad_source_reconciliation",
		"prepublication_missing_remaining_gates",
	} {
		if !report.HasIssue(code) {
			t.Fatalf("missing issue %q in %v", code, report.Codes())
		}
	}
}

func TestPrepublicationGateAcceptsConfigurableProjectBaseURL(t *testing.T) {
	record := prepublication.Record{
		TermID:                      "negativa-cobertura-plano-saude",
		Term:                        "negativa de cobertura do plano de saúde",
		GateID:                      "negativa-cobertura-plano-saude",
		GateStatus:                  "blocked_prepublication",
		Language:                    "pt-BR",
		SourceResolutionID:          "negativa-cobertura-plano-saude",
		SourceBlockerReconciliation: "source_resolved_blocker_still_active",
		CandidatePath:               "/temas/negativa-cobertura-plano-saude/",
		CandidateCanonicalURL:       "https://juridico-lab.test/temas/negativa-cobertura-plano-saude/",
		CandidateRobots:             "noindex,follow",
		CandidateTitle:              "Negativa de cobertura do plano: prova e urgência",
		CandidateMetaDescription:    "Entenda quais documentos separar após negativa do plano de saúde e quando buscar triagem jurídica online com fonte oficial.",
		RemainingGates:              []string{"definir URL oficial antes de publicar"},
		CheckedAt:                   "2026-06-09",
	}

	report := prepublication.ValidateRecordWithBaseURL(record, "https://juridico-lab.test")
	if !report.Passed() {
		t.Fatalf("configurable base URL should pass blocked prepublication gate, got %v", report.Messages())
	}

	record.CandidateCanonicalURL = "https://portal-juridico.example/temas/negativa-cobertura-plano-saude/"
	report = prepublication.ValidateRecordWithBaseURL(record, "https://juridico-lab.test")
	if !report.HasIssue("prepublication_canonical_mismatch") {
		t.Fatalf("missing canonical mismatch for stale hardcoded base, got %v", report.Codes())
	}
}

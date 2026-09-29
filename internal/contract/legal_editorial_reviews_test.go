package contract_test

import (
	"testing"

	"portaljuridico/internal/legalreviews"
)

func TestLegalEditorialReviewBlocksPublicationButPreparesContextualCTA(t *testing.T) {
	report := legalreviews.Validate(".")
	if !report.Passed() {
		t.Fatalf("legal editorial reviews failed contract: %v", report.Messages())
	}

	records, loadReport := legalreviews.LoadRecords(".")
	if !loadReport.Passed() {
		t.Fatalf("could not load legal editorial reviews: %v", loadReport.Messages())
	}

	var found *legalreviews.Record
	for _, entry := range records {
		if entry.Record.TermID == "negativa-cobertura-plano-saude" {
			record := entry.Record
			found = &record
			break
		}
	}
	if found == nil {
		t.Fatal("missing legal editorial review for negativa-cobertura-plano-saude")
	}
	if found.PublicationAllowed || found.RenderAllowed || found.SitemapAllowed || found.PublicPath != "" {
		t.Fatalf("review escaped blocked contract: publication=%t render=%t sitemap=%t path=%q", found.PublicationAllowed, found.RenderAllowed, found.SitemapAllowed, found.PublicPath)
	}
	if found.CTAStatus != "draft_contextual_not_public" {
		t.Fatalf("cta status=%q, want draft_contextual_not_public", found.CTAStatus)
	}
	if found.CTAOriginPath != "/temas/negativa-cobertura-plano-saude/" {
		t.Fatalf("cta origin path=%q, want candidate page path", found.CTAOriginPath)
	}
	if found.CTAOriginGateID != "negativa-cobertura-plano-saude" {
		t.Fatalf("cta origin gate=%q, want prepublication gate id", found.CTAOriginGateID)
	}
	for _, token := range []string{"Origem:", "/temas/negativa-cobertura-plano-saude/", "Gate: negativa-cobertura-plano-saude", "negativa de cobertura do plano de saúde"} {
		if !found.CTAContextMessageContains(token) {
			t.Fatalf("cta context message missing %q: %s", token, found.CTAContextMessage)
		}
	}
}

func TestLegalEditorialReviewRejectsPromiseCTAOrPublication(t *testing.T) {
	record := legalreviews.Record{
		TermID:               "negativa-cobertura-plano-saude",
		Term:                 "negativa de cobertura do plano de saúde",
		ReviewID:             "negativa-cobertura-plano-saude",
		ReviewStatus:         "legal_editorial_review_blocked",
		Language:             "pt-BR",
		AuthorialDraftID:     "negativa-cobertura-plano-saude",
		SourceResolutionID:   "negativa-cobertura-plano-saude",
		PrepublicationGateID: "negativa-cobertura-plano-saude",
		ReviewerRole:         "juridico_editorial_lab",
		LegalReviewNotes:     []string{"nota curta"},
		RequiredFixes:        []string{},
		CTADraft:             "Garantimos liminar pelo WhatsApp em 24 horas para resolver seu plano de saúde.",
		CTAContextMessage:    "Quero atendimento.",
		CTAStatus:            "approved_public",
		RenderAllowed:        true,
		SitemapAllowed:       true,
		PublicationAllowed:   true,
		PublicPath:           "/temas/negativa-cobertura-plano-saude/",
		CheckedAt:            "2026-06-09",
	}

	report := legalreviews.ValidateRecord(record)
	if report.Passed() {
		t.Fatal("ValidateRecord passed, want legal review and CTA failures")
	}
	for _, code := range []string{
		"legal_review_too_few_notes",
		"legal_review_missing_required_fixes",
		"legal_review_promise_cta",
		"legal_review_missing_cta_origin",
		"legal_review_cta_context_without_origin",
		"legal_review_invalid_cta_status",
		"legal_review_render_allowed",
		"legal_review_sitemap_allowed",
		"legal_review_publication_allowed",
		"legal_review_has_public_path",
	} {
		if !report.HasIssue(code) {
			t.Fatalf("missing issue %q in %v", code, report.Codes())
		}
	}
}

func TestLegalEditorialReviewAllowsDigitalServiceLanguageButRejectsOutcomePromise(t *testing.T) {
	record := validDigitalServiceReview()
	report := legalreviews.ValidateRecord(record)
	if report.HasIssue("legal_review_promise_cta") {
		t.Fatalf("digital service wording must not be treated as promise: %v", report.Messages())
	}
	if !report.Passed() {
		t.Fatalf("valid digital service review failed: %v", report.Messages())
	}

	record.CTADraft = "WhatsApp para atendimento jurídico online com resultado certo depois da análise digital dos documentos."
	report = legalreviews.ValidateRecord(record)
	if !report.HasIssue("legal_review_promise_cta") {
		t.Fatalf("outcome promise was not blocked: %v", report.Codes())
	}
}

func validDigitalServiceReview() legalreviews.Record {
	return legalreviews.Record{
		TermID:               "negativa-cobertura-plano-saude",
		Term:                 "negativa de cobertura do plano de saúde",
		ReviewID:             "negativa-cobertura-plano-saude",
		ReviewStatus:         "legal_editorial_review_blocked",
		Language:             "pt-BR",
		AuthorialDraftID:     "negativa-cobertura-plano-saude",
		SourceResolutionID:   "negativa-cobertura-plano-saude",
		PrepublicationGateID: "negativa-cobertura-plano-saude",
		ReviewerRole:         "juridico_editorial_lab",
		LegalReviewNotes: []string{
			"A revisão mantém conteúdo informativo sem promessa de resultado.",
			"O CTA pede documentos para triagem jurídica digital responsável.",
			"A linguagem remota descreve modo de atendimento, não êxito.",
		},
		RequiredFixes:     []string{"manter aviso informativo e revisar fonte antes de publicar"},
		CTADraft:          "WhatsApp para triagem jurídica 100% digital, com envio remoto de documentos e atendimento sem sair de casa para avaliar o contexto antes de contratar.",
		CTAContextMessage: "Origem: /temas/negativa-cobertura-plano-saude/ | Gate: negativa-cobertura-plano-saude | Tema: negativa de cobertura do plano de saúde | Documentos: contrato, negativa e laudos.",
		CTAOriginPath:     "/temas/negativa-cobertura-plano-saude/",
		CTAOriginGateID:   "negativa-cobertura-plano-saude",
		CTAStatus:         "draft_contextual_not_public",
		CheckedAt:         "2026-06-09",
	}
}

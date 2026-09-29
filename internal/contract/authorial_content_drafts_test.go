package contract_test

import (
	"testing"

	"portaljuridico/internal/authorialdrafts"
)

func TestAuthorialContentDraftsStartFromBriefsWithoutTemplateReuse(t *testing.T) {
	report := authorialdrafts.Validate(".")
	if !report.Passed() {
		t.Fatalf("authorial drafts failed contract: %v", report.Messages())
	}

	records, loadReport := authorialdrafts.LoadRecords(".")
	if !loadReport.Passed() {
		t.Fatalf("could not load authorial drafts: %v", loadReport.Messages())
	}
	if len(records) < 6 {
		t.Fatalf("got %d authorial drafts, want at least 6 initial drafts", len(records))
	}

	sourceBriefs := make(map[string]bool)
	angles := make(map[string]bool)
	for _, entry := range records {
		draft := entry.Record
		if draft.PublicationAllowed || draft.PublicPath != "" {
			t.Fatalf("draft %q escaped lab contract: publication=%t public_path=%q", draft.TermID, draft.PublicationAllowed, draft.PublicPath)
		}
		if draft.SourceBriefID == "" {
			t.Fatalf("draft %q missing source brief id", draft.TermID)
		}
		if sourceBriefs[draft.SourceBriefID] {
			t.Fatalf("source brief %q reused by multiple authorial drafts", draft.SourceBriefID)
		}
		sourceBriefs[draft.SourceBriefID] = true
		if angles[draft.UniqueAngle] {
			t.Fatalf("duplicated authorial angle %q", draft.UniqueAngle)
		}
		angles[draft.UniqueAngle] = true
	}
}

func TestAuthorialContentDraftRejectsMechanicalTemplateOrPublication(t *testing.T) {
	draft := authorialdrafts.Record{
		TermID:        "divorcio-online",
		Term:          "divórcio online",
		DraftStatus:   "authorial_draft",
		Language:      "pt-BR",
		SourceBriefID: "divorcio-online",
		UniqueAngle:   "conteudo informativo sobre o tema",
		ReaderProblem: "pessoa quer resolver uma questao juridica online",
		Opening:       "Este conteudo explica o tema de forma geral para quem deseja entender a questao juridica e procurar ajuda online.",
		BodySections: []authorialdrafts.Section{
			{Heading: "O que é", Text: "Texto curto e generico."},
			{Heading: "Documentos necessários", Text: "Texto curto e generico."},
			{Heading: "Quando procurar advogado", Text: "Texto curto e generico."},
		},
		DigitalCTAContext:  "chamar no WhatsApp",
		OfficialSourceURLs: []string{"https://www.cnj.jus.br/e-notariado-completa-tres-anos-com-mais-de-15-milhao-de-atos-online/"},
		PublicationAllowed: true,
		PublicPath:         "/temas/divorcio-online/",
		CheckedAt:          "2026-06-09",
	}

	report := authorialdrafts.ValidateRecord(draft)
	if report.Passed() {
		t.Fatal("ValidateRecord passed, want template/publication failures")
	}
	for _, code := range []string{
		"authorial_draft_publication_allowed",
		"authorial_draft_has_public_path",
		"authorial_draft_generic_angle",
		"authorial_draft_thin_opening",
		"authorial_draft_too_few_sections",
		"authorial_draft_generic_heading",
		"authorial_draft_thin_section_text",
		"authorial_draft_thin_cta_context",
		"authorial_draft_ptbr_spelling_failed",
	} {
		if !report.HasIssue(code) {
			t.Fatalf("missing issue %q in %v", code, report.Codes())
		}
	}
}

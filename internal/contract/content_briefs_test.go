package contract_test

import (
	"testing"

	"portaljuridico/internal/contentbriefs"
)

func TestContentBriefsStartContentWithoutTemplateReuseOrPublication(t *testing.T) {
	report := contentbriefs.Validate(".")
	if !report.Passed() {
		t.Fatalf("content briefs failed contract: %v", report.Messages())
	}

	records, loadReport := contentbriefs.LoadRecords(".")
	if !loadReport.Passed() {
		t.Fatalf("could not load content briefs: %v", loadReport.Messages())
	}
	if len(records) < 6 {
		t.Fatalf("got %d content briefs, want at least 6 initial briefs", len(records))
	}

	angles := make(map[string]bool)
	for _, entry := range records {
		brief := entry.Record
		if brief.PublicationAllowed || brief.PublicPath != "" {
			t.Fatalf("brief %q escaped lab contract: publication=%t public_path=%q", brief.TermID, brief.PublicationAllowed, brief.PublicPath)
		}
		if angles[brief.UniqueAngle] {
			t.Fatalf("duplicated brief angle %q", brief.UniqueAngle)
		}
		angles[brief.UniqueAngle] = true
	}
}

func TestContentBriefRejectsGenericTemplateSections(t *testing.T) {
	brief := contentbriefs.Record{
		TermID:             "divorcio-online",
		Term:               "divórcio online",
		BriefStatus:        "research_brief",
		Language:           "pt-BR",
		UniqueAngle:        "explicar o tema",
		ReaderProblem:      "pessoa procura informacao juridica",
		DigitalCTAReason:   "pode chamar no WhatsApp",
		OfficialSourceURLs: []string{"https://www.cnj.jus.br/e-notariado-completa-tres-anos-com-mais-de-15-milhao-de-atos-online/"},
		Sections: []contentbriefs.Section{
			{Heading: "O que é", Purpose: "explicar"},
			{Heading: "Quando procurar advogado", Purpose: "converter"},
			{Heading: "Documentos necessários", Purpose: "listar"},
		},
		PublicationAllowed: true,
		PublicPath:         "/temas/divorcio-online/",
		CheckedAt:          "2026-06-09",
	}

	report := contentbriefs.ValidateRecord(brief)
	if report.Passed() {
		t.Fatal("ValidateRecord passed, want template/publication failures")
	}
	for _, code := range []string{"brief_publication_allowed", "brief_has_public_path", "brief_too_few_sections", "brief_generic_angle", "brief_generic_heading"} {
		if !report.HasIssue(code) {
			t.Fatalf("missing issue %q in %v", code, report.Codes())
		}
	}
}

package contract_test

import (
	"strings"
	"testing"

	"portaljuridico/internal/content"
	"portaljuridico/internal/cta"
	"portaljuridico/internal/sources"
)

func TestWhatsAppCTARendersOnlyForApprovedLegalPageWithApprovedSource(t *testing.T) {
	policy, err := cta.LoadPolicy(".")
	if err != nil {
		t.Fatal(err)
	}
	registry, err := sources.LoadRegistry(".")
	if err != nil {
		t.Fatal(err)
	}

	noindexPage := approvedLegalPage()
	noindexPage.Status = "noindex"
	noindexPage.IndexPolicy = "noindex"
	if cta.CanRender(policy, noindexPage, registry) {
		t.Fatal("CTA rendered for noindex page")
	}

	preliminarySourcePage := approvedLegalPage()
	if cta.CanRender(policy, preliminarySourcePage, registry) {
		t.Fatal("CTA rendered for legal page with preliminary source")
	}

	approvedRegistry := registry
	for index, source := range approvedRegistry.Sources {
		if source.SourceID == "planalto" {
			approvedRegistry.Sources[index].AuditStatus = "approved"
			approvedRegistry.Sources[index].IngestionEnabled = false
		}
	}
	if !cta.CanRender(policy, approvedLegalPage(), approvedRegistry) {
		t.Fatal("CTA did not render for approved legal page with approved source")
	}
	message := cta.ContextMessage(policy, approvedLegalPage())
	for _, token := range []string{"/wiki/direito-civil/advogado-teste/", "wiki:advogado-teste", "Advogado teste"} {
		if !strings.Contains(message, token) {
			t.Fatalf("context message missing %q: %s", token, message)
		}
	}
}

func approvedLegalPage() content.Page {
	return content.Page{
		Path:            "/wiki/direito-civil/advogado-teste/",
		PageType:        "wiki",
		Status:          "published",
		IndexPolicy:     "index",
		UniqueIntentID:  "wiki:advogado-teste",
		CanonicalURL:    "https://portal-juridico.example/wiki/direito-civil/advogado-teste/",
		Title:           "Advogado teste",
		MetaDescription: "Pagina juridica aprovada para testar CTA de WhatsApp.",
		Heading:         "Advogado teste",
		Summary:         "Pagina informativa com fonte, revisao e intencao comercial legitima.",
		BodySections: []content.Section{
			{Title: "Informacao", Body: "Texto juridico informativo suficiente para teste de CTA. A pagina possui fonte, revisor, aviso e finalidade. O CTA so pode aparecer quando a fonte estiver aprovada, a pagina estiver publicada, indexavel, revisada e for tipo permitido pela politica. Isso impede uso comercial como atalho para conteudo raso ou mecanico."},
		},
		InternalLinks:      []string{"/", "/fontes/planalto/"},
		PublicationDate:    "2026-06-09",
		ReviewedAt:         "2026-06-09",
		Author:             "Redacao tecnica",
		Reviewer:           "Revisao juridica",
		SourceProvenance:   []content.SourceProvenance{{SourceID: "planalto", SourceName: "Planalto", SourceURL: "https://www.planalto.gov.br/", CheckedAt: "2026-06-09", LicenseNote: "Fonte aprovada apenas no fixture de teste."}},
		LegalNotice:        "Conteudo informativo; nao substitui consulta juridica individual.",
		PublicationPurpose: "Validar CTA gated.",
	}
}

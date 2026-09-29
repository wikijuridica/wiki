package contract_test

import (
	"testing"

	"portaljuridico/internal/content"
	"portaljuridico/internal/quality"
	"portaljuridico/internal/sources"
)

func TestIndexableLegalPageCannotUsePreliminarySource(t *testing.T) {
	registry, err := sources.LoadRegistry(".")
	if err != nil {
		t.Fatal(err)
	}

	page := content.Page{
		Path:            "/wiki/direito-civil/teste-fonte-preliminar/",
		PageType:        "wiki",
		Status:          "published",
		IndexPolicy:     "index",
		UniqueIntentID:  "wiki:teste-fonte-preliminar",
		CanonicalURL:    "https://portal-juridico.example/wiki/direito-civil/teste-fonte-preliminar/",
		Title:           "Teste de fonte preliminar",
		MetaDescription: "Pagina juridica de teste para bloquear fonte sem auditoria completa.",
		Heading:         "Teste de fonte preliminar",
		Summary:         "Conteudo juridico informativo ficticio para validar bloqueio de fonte preliminar.",
		BodySections: []content.Section{
			{
				Title: "Finalidade informativa",
				Body:  "Este texto tem tamanho suficiente para passar pelos gates basicos, mas a fonte ainda nao foi aprovada. A regra P0 deve bloquear qualquer indexacao juridica quando a fonte estiver apenas preliminar. A pagina demonstra que fonte pesquisada nao equivale a fonte aprovada para publicacao. A arquitetura deve exigir auditoria de termos, robots, proveniencia, privacidade, revisao e estrategia de cache antes de liberar conteudo juridico ao indice.",
			},
			{
				Title: "Bloqueio esperado",
				Body:  "Mesmo com autor, revisor, datas, aviso informativo e links internos, o conteudo juridico nao pode ser publicado enquanto o registro da fonte indicar ingestao bloqueada ou auditoria preliminar. O teste protege contra crescimento mecanico e contra publicacao de paginas juridicas sem fonte plenamente auditada.",
			},
		},
		InternalLinks:      []string{"/", "/fontes/planalto/"},
		PublicationDate:    "2026-06-09",
		ReviewedAt:         "2026-06-09",
		Author:             "Redacao tecnica",
		Reviewer:           "Revisao juridica",
		SourceProvenance:   []content.SourceProvenance{{SourceID: "planalto", SourceName: "Planalto", SourceURL: "https://www.planalto.gov.br/", CheckedAt: "2026-06-09", LicenseNote: "Fonte preliminar em auditoria."}},
		LegalNotice:        "Conteudo informativo; nao substitui consulta juridica individual.",
		PublicationPurpose: "Validar bloqueio P0 de fonte preliminar.",
	}

	report := quality.ValidatePagesWithSources([]content.Page{page}, registry)
	if report.Passed() {
		t.Fatal("ValidatePagesWithSources passed, want source audit failure")
	}
	if !report.HasIssue("source_not_approved_for_indexable_legal_content") {
		t.Fatalf("missing source_not_approved_for_indexable_legal_content in %v", report.Codes())
	}
}

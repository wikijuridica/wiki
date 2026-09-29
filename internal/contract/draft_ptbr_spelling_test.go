package contract_test

import (
	"strings"
	"testing"

	"portaljuridico/internal/draftlab"
	"portaljuridico/internal/terms"
)

func TestDraftLabUsesNaturalPTBRDisplayTerms(t *testing.T) {
	seed := terms.Seed{
		TermID:       "divorcio-online",
		Term:         "divorcio online",
		Language:     "pt-BR",
		QualityState: "draft_only",
		SourceID:     "cnj",
		SourceURL:    "https://www.cnj.jus.br/e-notariado-completa-tres-anos-com-mais-de-15-milhao-de-atos-online/",
		CheckedAt:    "2026-06-09",
		IntentHint:   "priorizar conteudo informativo com demanda humana e contratacao juridica digital",
	}

	draft, err := draftlab.Build(seed)
	if err != nil {
		t.Fatal(err)
	}
	if draft.Term != "divórcio online" {
		t.Fatalf("draft term=%q, want natural PT-BR spelling", draft.Term)
	}
	for _, token := range []string{"Divórcio online", "No caso de divórcio online"} {
		if !strings.Contains(draft.Text, token) {
			t.Fatalf("draft text missing %q: %s", token, draft.Text)
		}
	}
}

package contract_test

import (
	"testing"

	"portaljuridico/internal/sourceblockers"
)

func TestPrioritizedTermsRequireSpecificSourceBlockersBeforeApproval(t *testing.T) {
	report := sourceblockers.Validate(".")
	if !report.Passed() {
		t.Fatalf("source blockers failed contract: %v", report.Messages())
	}

	for _, termID := range []string{
		"advogado-trabalhista-online",
		"negativa-cobertura-plano-saude",
		"divorcio-online",
		"auxilio-doenca-negado",
		"inventario-extrajudicial-online",
		"desconto-indevido-inss",
	} {
		if !sourceblockers.HasBlockerForTerm(".", termID) {
			t.Fatalf("missing source-specific blocker for %q", termID)
		}
	}
}

func TestSourceSpecificityBlockerRejectsApprovalOrPublicRoute(t *testing.T) {
	blocker := sourceblockers.Record{
		TermID:                "divorcio-online",
		Term:                  "divórcio online",
		SourceReviewStatus:    "needs_specific_source",
		ApprovalAllowed:       true,
		PublicationAllowed:    true,
		PublicPath:            "/temas/divorcio-online/",
		CurrentSourceID:       "cnj",
		CurrentSourceURL:      "https://www.cnj.jus.br/e-notariado-completa-tres-anos-com-mais-de-15-milhao-de-atos-online/",
		RequiredSourceTypes:   []string{"primary_legal_basis"},
		MissingRequirements:   []string{"primary_legal_basis"},
		SpecificityReason:     "fonte institucional ampla ainda nao basta para conteudo juridico publico",
		CheckedAt:             "2026-06-09",
		NextResearchDirection: "pesquisar norma especifica antes de aprovar",
	}

	report := sourceblockers.ValidateRecord(blocker)
	if report.Passed() {
		t.Fatal("ValidateRecord passed, want approval/publication failures")
	}
	for _, code := range []string{"source_blocker_approval_allowed", "source_blocker_publication_allowed", "source_blocker_has_public_path"} {
		if !report.HasIssue(code) {
			t.Fatalf("missing issue %q in %v", code, report.Codes())
		}
	}
}

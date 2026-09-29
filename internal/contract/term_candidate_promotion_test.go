package contract_test

import (
	"testing"

	"portaljuridico/internal/termpromotion"
	"portaljuridico/internal/terms"
)

func TestTermCandidatePromotionCreatesDemandBackedDraftSeeds(t *testing.T) {
	plan, err := termpromotion.PlanTopCandidates(".", 6)
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.Seeds) != 6 {
		t.Fatalf("got %d promoted seeds, want 6", len(plan.Seeds))
	}

	seenAreas := make(map[string]bool)
	for _, promoted := range plan.Seeds {
		report := terms.ValidateSeed(promoted.Seed)
		if !report.Passed() {
			t.Fatalf("promoted seed %q failed seed contract: %v", promoted.Seed.TermID, report.Messages())
		}
		if promoted.Seed.CandidateID == "" {
			t.Fatalf("promoted seed %q missing candidate id", promoted.Seed.TermID)
		}
		if promoted.Seed.DemandEvidenceURL == "" {
			t.Fatalf("promoted seed %q missing demand evidence", promoted.Seed.TermID)
		}
		if promoted.Seed.OnlineServiceMode != "digital_only" {
			t.Fatalf("promoted seed %q online mode=%q", promoted.Seed.TermID, promoted.Seed.OnlineServiceMode)
		}
		if promoted.Seed.WhatsAppCTAIntent != "high" {
			t.Fatalf("promoted seed %q whatsapp intent=%q", promoted.Seed.TermID, promoted.Seed.WhatsAppCTAIntent)
		}
		seenAreas[promoted.Candidate.PracticeArea] = true
	}

	if len(seenAreas) < 4 {
		t.Fatalf("promoted seeds cover %d areas, want at least 4 to avoid one-area tunnel vision", len(seenAreas))
	}
}

func TestPromotedTermSeedsArePersistedAndStillNonPublishing(t *testing.T) {
	report := termpromotion.ValidatePromotedSeeds(".", 6)
	if !report.Passed() {
		t.Fatalf("promoted seeds failed contract: %v", report.Messages())
	}
}

func TestTermCandidateScoringRefinesWeakOrPresentialSignals(t *testing.T) {
	strong := termpromotion.ScoreCandidateForTest(candidateForScoring("negativa-cobertura-plano-saude", "saude_consumidor", "high", "digital_only", "https://www.gov.br/ans/pt-br/canais_atendimento/canais-de-atendimento-ao-consumidor-1"))
	weak := termpromotion.ScoreCandidateForTest(candidateForScoring("consulta-presencial-forum", "familia", "high", "in_person_required", "https://example.com/sem-fonte-oficial"))

	if strong <= weak {
		t.Fatalf("strong digital official candidate scored %d, weak presential candidate scored %d", strong, weak)
	}
}

func candidateForScoring(termID string, area string, cta string, mode string, sourceURL string) termpromotion.ScoreCandidate {
	return termpromotion.ScoreCandidate{
		TermID:            termID,
		PracticeArea:      area,
		RiskLevel:         "high",
		WhatsAppCTAIntent: cta,
		OnlineServiceMode: mode,
		OfficialSourceURL: sourceURL,
	}
}

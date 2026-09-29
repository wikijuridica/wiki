package contract_test

import (
	"testing"

	"portaljuridico/internal/termintents"
)

func TestHighIntentTermCandidatesStayNonPublishingAndDigital(t *testing.T) {
	report := termintents.Validate(".")
	if !report.Passed() {
		t.Fatalf("term intent candidates failed P0 contract: %v", report.Messages())
	}

	candidates, loadReport := termintents.LoadCandidates(".")
	if !loadReport.Passed() {
		t.Fatalf("could not load term intent candidates: %v", loadReport.Messages())
	}
	if len(candidates) < 8 {
		t.Fatalf("got %d candidates, want at least 8 high-intent digital legal terms", len(candidates))
	}

	for _, entry := range candidates {
		candidate := entry.Candidate
		if candidate.PublicationAllowed {
			t.Fatalf("candidate %q allowed publication during research cycle", candidate.TermID)
		}
		if candidate.PublicPath != "" {
			t.Fatalf("candidate %q has public path %q", candidate.TermID, candidate.PublicPath)
		}
		if candidate.OnlineServiceMode != "digital_only" {
			t.Fatalf("candidate %q mode=%q, want digital_only", candidate.TermID, candidate.OnlineServiceMode)
		}
		if candidate.WhatsAppCTAIntent != "high" {
			t.Fatalf("candidate %q whatsapp intent=%q, want high", candidate.TermID, candidate.WhatsAppCTAIntent)
		}
	}
}

func TestHighIntentTermCandidateRejectsPresentialOrUnprovenDemand(t *testing.T) {
	candidate := termintents.Candidate{
		TermID:             "inventario-presencial",
		Term:               "inventario presencial",
		Language:           "pt-BR",
		CandidateStatus:    "research_candidate",
		PracticeArea:       "familia_sucessoes",
		DemandEvidenceType: "",
		DemandEvidenceURL:  "",
		OfficialSourceID:   "cnj",
		OfficialSourceURL:  "https://www.cnj.jus.br/",
		OnlineServiceMode:  "in_person_required",
		WhatsAppCTAIntent:  "low",
		CheckedAt:          "2026-06-09",
		IntentReason:       "termo depende de comparecimento presencial e nao prova demanda humana atual",
		PublicationAllowed: true,
		PublicPath:         "/temas/inventario-presencial/",
	}

	report := termintents.ValidateCandidate(candidate)
	if report.Passed() {
		t.Fatal("ValidateCandidate passed, want demand, digital-only and publication failures")
	}
	for _, code := range []string{"missing_demand_evidence", "not_digital_only", "whatsapp_intent_not_high", "candidate_publication_allowed", "candidate_has_public_path"} {
		if !report.HasIssue(code) {
			t.Fatalf("missing issue %q in %v", code, report.Codes())
		}
	}
}

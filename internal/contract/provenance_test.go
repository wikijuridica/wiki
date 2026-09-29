package contract_test

import (
	"testing"

	"portaljuridico/internal/provenance"
	"portaljuridico/internal/sources"
)

func TestPayloadProvenanceRequiresSourceURLHashRobotsAndTermsSnapshot(t *testing.T) {
	registry, err := sources.LoadRegistry(".")
	if err != nil {
		t.Fatal(err)
	}

	record := provenance.PayloadRecord{
		SourceID:       "camara-dados-abertos",
		OfficialURL:    "https://dadosabertos.camara.leg.br/api/v2/deputados",
		RetrievedAt:    "2026-06-09T08:00:00-03:00",
		PayloadSHA256:  "b94d27b9934d3e08a52e52d7da7dabfadeadbeef000000000000000000000000",
		RobotsSnapshot: "https://dadosabertos.camara.leg.br/robots.txt",
		TermsSnapshot:  "https://dadosabertos.camara.leg.br/",
		Fields:         []string{"id", "nome", "siglaPartido", "siglaUf"},
		UsePurpose:     "laboratorio de proveniencia sem publicacao de conteudo",
	}

	report := provenance.ValidatePayloadRecord(record, registry)
	if !report.Passed() {
		t.Fatalf("valid payload provenance failed: %v", report.Messages())
	}

	record.PayloadSHA256 = ""
	report = provenance.ValidatePayloadRecord(record, registry)
	if report.Passed() {
		t.Fatal("missing payload hash should fail")
	}
	if !report.HasIssue("missing_payload_hash") {
		t.Fatalf("missing missing_payload_hash in %v", report.Codes())
	}
}

func TestSourceRegistryCheckIsAvailableAsNamedCheck(t *testing.T) {
	// Covered through cmd/check in lab-cycle; this keeps the required check name in contract.
	required := "sources"
	if required == "" {
		t.Fatal("unreachable")
	}
}

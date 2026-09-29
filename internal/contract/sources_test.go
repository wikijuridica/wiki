package contract_test

import (
	"os"
	"path/filepath"
	"testing"

	"portaljuridico/internal/sources"
)

func TestInitialOfficialSourcesAreRegisteredDocumentedAndBlockedForIngestion(t *testing.T) {
	registry, err := sources.LoadRegistry(".")
	if err != nil {
		t.Fatal(err)
	}

	required := []string{
		"planalto",
		"camara-dados-abertos",
		"senado-dados-abertos",
		"lexml",
		"cnj-datajud",
		"stf",
		"stj",
		"cjf",
	}
	for _, sourceID := range required {
		source, ok := registry.ByID(sourceID)
		if !ok {
			t.Fatalf("missing official source %q", sourceID)
		}
		if source.IngestionEnabled {
			t.Fatalf("source %q has ingestion enabled before full audit", sourceID)
		}
		if source.DocumentationPath == "" {
			t.Fatalf("source %q missing documentation path", sourceID)
		}
		if _, err := os.Stat(filepath.Join(findRoot(t), source.DocumentationPath)); err != nil {
			t.Fatalf("source %q documentation missing: %v", sourceID, err)
		}
		if source.BaseURL == "" || source.ResearchURL == "" || source.DataType == "" {
			t.Fatalf("source %q incomplete registry entry: %+v", sourceID, source)
		}
	}
}

func TestOfficialSourceRegistryBlocksContentUntilResearchRobotsTermsAndProvenance(t *testing.T) {
	registry, err := sources.LoadRegistry(".")
	if err != nil {
		t.Fatal(err)
	}

	report := registry.ValidateForP0()
	if !report.Passed() {
		t.Fatalf("P0 source registry failed: %v", report.Messages())
	}
	if registry.AnyIngestionEnabled() {
		t.Fatal("no source ingestion can be enabled during P0 source-audit stage")
	}
}

func TestOfficialSourcesCarryRobotsTermsAndProvenanceAuditFields(t *testing.T) {
	registry, err := sources.LoadRegistry(".")
	if err != nil {
		t.Fatal(err)
	}

	for _, source := range registry.Sources {
		if source.RobotsURL == "" || source.RobotsStatus == "" {
			t.Fatalf("source %q missing robots audit fields: %+v", source.SourceID, source)
		}
		if source.RobotsCheckedAt == "" {
			t.Fatalf("source %q missing robots checked date", source.SourceID)
		}
		if source.TermsURL == "" || source.TermsStatus == "" {
			t.Fatalf("source %q missing terms audit fields: %+v", source.SourceID, source)
		}
		if source.TermsCheckedAt == "" {
			t.Fatalf("source %q missing terms checked date", source.SourceID)
		}
		if source.AuditDecision == "" || source.AuditNote == "" {
			t.Fatalf("source %q missing audit decision/note: %+v", source.SourceID, source)
		}
		if source.ProvenanceStrategy == "" {
			t.Fatalf("source %q missing provenance strategy", source.SourceID)
		}
		if source.PrivacyRisk == "" {
			t.Fatalf("source %q missing privacy risk", source.SourceID)
		}
		if source.NoSignupRequired && source.AccessMode == "" {
			t.Fatalf("source %q claims no signup but has no access mode", source.SourceID)
		}
	}
}

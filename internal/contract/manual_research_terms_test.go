package contract_test

import (
	"testing"

	"portaljuridico/internal/manualresearch"
)

func TestManualHighIntentResearchDatabaseIsHumanResearchedAndNonPublishing(t *testing.T) {
	report := manualresearch.Validate(".")
	if !report.Passed() {
		t.Fatalf("manual research database failed contract: %v", report.Messages())
	}

	records, loadReport := manualresearch.LoadRecords(".")
	if !loadReport.Passed() {
		t.Fatalf("could not load manual research database: %v", loadReport.Messages())
	}
	if len(records) < 20 {
		t.Fatalf("got %d researched terms, want at least 20 to avoid conservative tunnel vision", len(records))
	}

	areas := make(map[string]bool)
	for _, entry := range records {
		record := entry.Record
		if record.ResearchMethod != "manual_web_research" {
			t.Fatalf("term %q method=%q, want manual_web_research", record.TermID, record.ResearchMethod)
		}
		if record.TrendsRole != "orientation_only" {
			t.Fatalf("term %q trends_role=%q, want orientation_only", record.TermID, record.TrendsRole)
		}
		if record.PublicationAllowed || record.PublicPath != "" {
			t.Fatalf("term %q escaped research contract: publication=%t public_path=%q", record.TermID, record.PublicationAllowed, record.PublicPath)
		}
		areas[record.PracticeArea] = true
	}
	if len(areas) < 6 {
		t.Fatalf("researched terms cover %d areas, want at least 6 legal service areas", len(areas))
	}
}

func TestManualResearchRejectsTrendOnlyOrTemplateRecord(t *testing.T) {
	record := manualresearch.Record{
		TermID:             "advogado-generico",
		Term:               "advogado online",
		Language:           "pt-BR",
		ResearchMethod:     "script_generated",
		PracticeArea:       "generico",
		DigitalServiceMode: "digital_only",
		HiringIntent:       "high",
		TrendsRole:         "primary_source",
		TrendURLs:          []string{"https://trends.google.com.br/trends/explore?geo=BR&q=advogado%20online"},
		OfficialSources:    []manualresearch.Source{},
		ContentUse:         "publish_now",
		PublicationAllowed: true,
		PublicPath:         "/temas/advogado-online/",
		CheckedAt:          "2026-06-09",
		ResearchNotes:      "termo generico sem pesquisa editorial real",
	}

	report := manualresearch.ValidateRecord(record)
	if report.Passed() {
		t.Fatal("ValidateRecord passed, want weak research failures")
	}
	for _, code := range []string{"not_manual_web_research", "trend_not_orientation_only", "missing_official_authority_sources", "research_publication_allowed", "research_has_public_path", "invalid_content_use"} {
		if !report.HasIssue(code) {
			t.Fatalf("missing issue %q in %v", code, report.Codes())
		}
	}
}

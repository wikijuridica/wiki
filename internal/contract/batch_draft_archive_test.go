package contract_test

import (
	"testing"

	"portaljuridico/internal/batchdraftarchive"
)

func TestBatchDraftExpansionArchiveKeepsHundredsOfValidatedDraftsBlocked(t *testing.T) {
	report := batchdraftarchive.Validate(".")
	if !report.Passed() {
		t.Fatalf("batch draft expansion archive failed: %v", report.Messages())
	}

	entries, loadReport := batchdraftarchive.LoadRecords(".")
	if !loadReport.Passed() {
		t.Fatalf("could not load batch draft expansion archive: %v", loadReport.Messages())
	}
	if len(entries) < 600 {
		t.Fatalf("archive drafts=%d, want at least 600 preserved drafts", len(entries))
	}
}

package batchcandidatepromotion

import (
	"testing"

	"portaljuridico/internal/batchcandidategates"
)

func TestValidateTotalSelectedRejectsUnexpectedAdvanceTotal(t *testing.T) {
	records := []batchcandidategates.Record{
		{SelectedUniqueIntentIDs: []string{"a", "b"}},
		{SelectedUniqueIntentIDs: []string{"c"}},
	}

	if report := ValidateTotalSelected(records, 3); !report.Passed() {
		t.Fatalf("expected total passed: %v", report.Messages())
	}
	report := ValidateTotalSelected(records, 4)
	if report.Passed() {
		t.Fatal("unexpected total passed, want guard failure")
	}
	if !containsMessage(report.Messages(), "selected=3 expected=4") {
		t.Fatalf("guard did not report selected/expected totals: %v", report.Messages())
	}
}

func containsMessage(messages []string, expected string) bool {
	for _, message := range messages {
		if message == "batch_candidate_unexpected_total_selected: "+expected {
			return true
		}
	}
	return false
}

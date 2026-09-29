package batchexpansionapply

import (
	"testing"

	"portaljuridico/internal/paidintent"
)

func TestEligibleForStrategyExpansionUsesPrevidenciarioFlexOnlyInsideBatch(t *testing.T) {
	if !eligibleForStrategyExpansion(paidintent.Record{
		BatchID:          "batch-familia-digital",
		PaidIntentStatus: paidintent.PassedBlockedStatus,
		IndexPolicy:      "noindex",
	}) {
		t.Fatalf("passed paid-intent record should remain eligible")
	}
	if !eligibleForStrategyExpansion(paidintent.Record{
		BatchID:          "batch-previdenciario-digital",
		PaidIntentStatus: paidintent.PrevidenciarioInformationalStatus,
		IndexPolicy:      "noindex",
	}) {
		t.Fatalf("previdenciario informational record should be eligible for blocked internal strategy expansion")
	}
	if eligibleForStrategyExpansion(paidintent.Record{
		BatchID:          "batch-consumidor-financeiro-digital",
		PaidIntentStatus: paidintent.PrevidenciarioInformationalStatus,
		IndexPolicy:      "noindex",
	}) {
		t.Fatalf("forged previdenciario informational status outside previdenciario batch must not be eligible")
	}
}

package contract_test

import (
	"testing"

	"portaljuridico/internal/paidintent"
	"portaljuridico/internal/paidintentrefinement"
)

func TestPaidIntentRefinementsPersistNaturalBodyRewritesWithoutPublishing(t *testing.T) {
	report := paidintentrefinement.Validate(".")
	if !report.Passed() {
		t.Fatalf("paid intent refinement file failed: %v", report.Messages())
	}
	entries, loadReport := paidintentrefinement.LoadRecords(".")
	if !loadReport.Passed() {
		t.Fatalf("could not load paid intent refinements: %v", loadReport.Messages())
	}
	if len(entries) < 100 {
		t.Fatalf("refinement records=%d, want mass refinement for blocked paid-intent candidates", len(entries))
	}

	seenCTAOnly := 0
	seenLowBusiness := 0
	seenMissing := 0
	for _, entry := range entries {
		record := entry.Record
		switch record.OriginalPaidIntentStatus {
		case paidintent.MissingPaidSignalStatus:
			seenMissing++
		case paidintent.CTAOnlyBlockedStatus:
			seenCTAOnly++
		case paidintent.LowBusinessScoreStatus:
			seenLowBusiness++
		case paidintent.PublicAssistanceBlockedStatus, paidintent.AdminSelfServiceBlockedStatus:
			t.Fatalf("refinement tried to rescue blocked free/self-service intent=%s status=%s", record.UniqueIntentID, record.OriginalPaidIntentStatus)
		default:
			t.Fatalf("unexpected refinement original status for %s: %s", record.UniqueIntentID, record.OriginalPaidIntentStatus)
		}
		if record.AfterPaidIntentStatus != paidintent.PassedBlockedStatus {
			t.Fatalf("intent=%s after status=%s, want %s", record.UniqueIntentID, record.AfterPaidIntentStatus, paidintent.PassedBlockedStatus)
		}
		if record.AfterHumanScore < 85 || record.AfterAILikeScore > 20 {
			t.Fatalf("intent=%s weak human score after refinement: human=%d ai=%d", record.UniqueIntentID, record.AfterHumanScore, record.AfterAILikeScore)
		}
		if record.RenderAllowed || record.SitemapAllowed || record.PublicationAllowed || record.PublicPath != "" || record.IndexPolicy != "noindex" {
			t.Fatalf("intent=%s refinement escaped blocked publication contract", record.UniqueIntentID)
		}
	}
	if seenMissing+seenCTAOnly+seenLowBusiness < 100 {
		t.Fatalf("refinement status coverage missing=%d cta_only=%d low_business=%d", seenMissing, seenCTAOnly, seenLowBusiness)
	}

	paidRecords, paidReport := paidintent.LoadRecords(".")
	if !paidReport.Passed() {
		t.Fatalf("could not load paid intent gates: %v", paidReport.Messages())
	}
	for _, entry := range paidRecords {
		status := entry.Record.PaidIntentStatus
		if status == paidintent.MissingPaidSignalStatus || status == paidintent.CTAOnlyBlockedStatus {
			t.Fatalf("line=%d intent=%s still has status=%s after refinement", entry.Line, entry.Record.UniqueIntentID, status)
		}
	}
}

func TestPaidIntentRefinementCommandIsIdempotentAfterAppliedCycle(t *testing.T) {
	plan, planReport := paidintentrefinement.BuildPlan(".")
	if !planReport.Passed() {
		t.Fatalf("could not build paid-intent refinement plan: %v", planReport.Messages())
	}
	if len(plan.Records) != 0 {
		t.Fatalf("paid-intent refinement plan still has %d pending records; apply refinements before idempotence check", len(plan.Records))
	}

	result, applyReport := paidintentrefinement.RefineRepository(".")
	if !applyReport.Passed() {
		t.Fatalf("idempotent paid-intent refinement failed: %v", applyReport.Messages())
	}
	if len(result.Refinements) != 0 || result.ArchiveRefined != 0 || result.FinalRefined != 0 {
		t.Fatalf("idempotent refinement changed data: refinements=%d archive=%d final=%d", len(result.Refinements), result.ArchiveRefined, result.FinalRefined)
	}
}

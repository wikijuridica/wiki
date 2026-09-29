package contract_test

import (
	"testing"

	"portaljuridico/internal/batchexpansionstrategy"
)

func TestBatchExpansionStrategyPlansNextScaleWithoutPublishing(t *testing.T) {
	report := batchexpansionstrategy.Validate(".")
	if !report.Passed() {
		t.Fatalf("batch expansion strategy failed contract: %v", report.Messages())
	}

	records, loadReport := batchexpansionstrategy.LoadRecords(".")
	if !loadReport.Passed() {
		t.Fatalf("could not load batch expansion strategy: %v", loadReport.Messages())
	}
	if len(records) != 6 {
		t.Fatalf("strategy records=%d, want one per batch family", len(records))
	}

	readyFamilies := 0
	paidBlockedFamilies := 0
	archiveGrowthFamilies := 0
	totalCurrent := 0
	totalNext := 0
	for _, entry := range records {
		record := entry.Record
		totalCurrent += record.CurrentCandidateCount
		totalNext += record.NextCandidateTarget
		if record.StrategyStatus == batchexpansionstrategy.ReadyNextCandidateGateStatus {
			readyFamilies++
			if record.NextCandidateTarget < 60 {
				t.Fatalf("line=%d next target=%d, want at least 60 for ready family", entry.Line, record.NextCandidateTarget)
			}
		}
		if record.StrategyStatus == batchexpansionstrategy.BlockedPaidIntentStatus {
			paidBlockedFamilies++
			if record.NextCandidateTarget != record.CurrentCandidateCount {
				t.Fatalf("line=%d paid blocked target=%d, want current=%d until paid-intent is refined", entry.Line, record.NextCandidateTarget, record.CurrentCandidateCount)
			}
			if record.PaidIntentBlockedCount == 0 {
				t.Fatalf("line=%d paid blocked strategy without blocked paid-intent count", entry.Line)
			}
			if record.ArchiveRecordsObserved <= record.CurrentCandidateCount {
				t.Fatalf("line=%d paid blocked family should already have archive headroom, archive=%d current=%d", entry.Line, record.ArchiveRecordsObserved, record.CurrentCandidateCount)
			}
		}
		if record.StrategyStatus == batchexpansionstrategy.ArchiveGrowthRequiredStatus {
			archiveGrowthFamilies++
			if record.ArchiveRecordsObserved != record.CurrentCandidateCount {
				t.Fatalf("line=%d archive growth status requires current at archive limit, archive=%d current=%d", entry.Line, record.ArchiveRecordsObserved, record.CurrentCandidateCount)
			}
			if record.NextCandidateTarget != record.CurrentCandidateCount {
				t.Fatalf("line=%d archive growth target=%d, want current=%d until archive expands", entry.Line, record.NextCandidateTarget, record.CurrentCandidateCount)
			}
			if record.NextAction == "" {
				t.Fatalf("line=%d archive growth status needs next archive action", entry.Line)
			}
		}
		if !record.RequiresVerifiedSource || !record.RequiresPaidIntent || !record.RequiresContextualCTA || !record.RequiresHumanScore || !record.RequiresSemanticDiversity {
			t.Fatalf("line=%d missing intelligent expansion guard", entry.Line)
		}
		if len(record.SemanticAxes) < 6 || len(record.ValidationCommands) < 5 {
			t.Fatalf("line=%d weak semantic axes or validation plan", entry.Line)
		}
		if record.IndexPolicy != "noindex" || record.ManifestAllowed || record.RenderAllowed || record.SitemapAllowed || record.PublicationAllowed || record.PublicPath != "" {
			t.Fatalf("line=%d strategy escaped blocked contract", entry.Line)
		}
	}
	if readyFamilies+archiveGrowthFamilies < 1 {
		t.Fatalf("strategy did not plan candidate growth or archive growth")
	}
	if readyFamilies > 0 && totalNext <= totalCurrent {
		t.Fatalf("strategy did not plan candidate growth: current=%d next=%d", totalCurrent, totalNext)
	}
	if readyFamilies == 0 && archiveGrowthFamilies != len(records) {
		t.Fatalf("ready families=%d archive growth families=%d records=%d", readyFamilies, archiveGrowthFamilies, len(records))
	}
}

func TestBatchExpansionStrategyRejectsPassiveOrPublicPlan(t *testing.T) {
	index := batchexpansionstrategy.StrategyIndex{
		ReadinessByBatch: map[string]batchexpansionstrategy.ReadinessSnapshot{
			"batch-familia-digital": {
				BatchID:                "batch-familia-digital",
				LegalArea:              "familia",
				ReadinessStatus:        "batch_candidate_expansion_ready_blocked_publication",
				ArchiveRecordsObserved: 100,
				CurrentCandidateCount:  30,
				PaidIntentBlockedCount: 0,
			},
		},
	}
	record := batchexpansionstrategy.Record{
		StrategyID:                "bad-strategy",
		BatchID:                   "batch-familia-digital",
		LegalArea:                 "familia",
		StrategyStatus:            "ready_to_publish",
		CurrentCandidateCount:     30,
		NextCandidateTarget:       30,
		MaxGrowthStep:             1000,
		RequiresVerifiedSource:    false,
		RequiresPaidIntent:        false,
		RequiresContextualCTA:     false,
		RequiresHumanScore:        false,
		RequiresSemanticDiversity: false,
		SemanticAxes:              []string{"keyword"},
		ValidationCommands:        []string{"./tools/check-all"},
		NextAction:                "",
		IndexPolicy:               "index",
		ManifestAllowed:           true,
		RenderAllowed:             true,
		SitemapAllowed:            true,
		PublicationAllowed:        true,
		PublicPath:                "/temas/familia/",
		CheckedAt:                 "2026-06-09",
	}

	report := batchexpansionstrategy.ValidateRecordAgainstIndex(record, index)
	if report.Passed() {
		t.Fatal("ValidateRecordAgainstIndex passed, want passive/public expansion failures")
	}
	for _, code := range []string{
		"batch_expansion_strategy_status_not_blocked",
		"batch_expansion_strategy_not_growing_ready_family",
		"batch_expansion_strategy_growth_step_too_large",
		"batch_expansion_strategy_without_verified_source",
		"batch_expansion_strategy_without_paid_intent",
		"batch_expansion_strategy_without_contextual_cta",
		"batch_expansion_strategy_without_human_score",
		"batch_expansion_strategy_without_semantic_diversity",
		"batch_expansion_strategy_weak_semantic_axes",
		"batch_expansion_strategy_weak_validation_plan",
		"batch_expansion_strategy_missing_next_action",
		"batch_expansion_strategy_invalid_index_policy",
		"batch_expansion_strategy_manifest_allowed",
		"batch_expansion_strategy_render_allowed",
		"batch_expansion_strategy_sitemap_allowed",
		"batch_expansion_strategy_publication_allowed",
		"batch_expansion_strategy_has_public_path",
	} {
		if !report.HasIssue(code) {
			t.Fatalf("missing issue %q in %v", code, report.Codes())
		}
	}
}

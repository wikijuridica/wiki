package contract_test

import (
	"encoding/json"
	"fmt"
	"testing"

	"portaljuridico/internal/batchcandidateexpansion"
	"portaljuridico/internal/storage"
)

func TestBatchCandidateExpansionReadinessUsesArchiveWithoutPublishing(t *testing.T) {
	report := batchcandidateexpansion.Validate(".")
	if !report.Passed() {
		t.Fatalf("batch candidate expansion readiness failed contract: %v", report.Messages())
	}

	records, loadReport := batchcandidateexpansion.LoadRecords(".")
	if !loadReport.Passed() {
		t.Fatalf("could not load batch candidate expansion readiness: %v", loadReport.Messages())
	}
	if len(records) != 6 {
		t.Fatalf("readiness records=%d, want one per batch family", len(records))
	}
	index, indexReport := batchcandidateexpansion.BuildExpansionIndex(".")
	if !indexReport.Passed() {
		t.Fatalf("could not build expansion index: %v", indexReport.Messages())
	}

	for _, entry := range records {
		record := entry.Record
		candidateIntentIDs := batchcandidateexpansion.CandidateIntentIDs(record, index)
		if record.PaidIntentGatePath != batchcandidateexpansion.PaidIntentGatePath {
			t.Fatalf("line=%d paid gate path=%q", entry.Line, record.PaidIntentGatePath)
		}
		if record.PaidIntentMissingCount != 0 {
			t.Fatalf("line=%d paid-intent missing=%d, want 0 after generated gates", entry.Line, record.PaidIntentMissingCount)
		}
		if record.PaidIntentBlockedCount > 0 && record.ReadinessStatus != batchcandidateexpansion.PaidGateBlockedStatus {
			t.Fatalf("line=%d paid blocked=%d status=%q", entry.Line, record.PaidIntentBlockedCount, record.ReadinessStatus)
		}
		if record.PaidIntentBlockedCount == 0 && record.ReadinessStatus != batchcandidateexpansion.ReadyBlockedStatus {
			t.Fatalf("line=%d paid clear status=%q", entry.Line, record.ReadinessStatus)
		}
		totalPaidClassified := record.PaidIntentPassedCount + record.PaidIntentBlockedCount + record.PaidIntentMissingCount
		if totalPaidClassified != len(candidateIntentIDs) {
			t.Fatalf("line=%d paid classified=%d intents=%d", entry.Line, totalPaidClassified, len(candidateIntentIDs))
		}
		if record.TargetCandidateCount < 30 {
			t.Fatalf("line=%d target=%d, want at least 30", entry.Line, record.TargetCandidateCount)
		}
		if record.ReadinessStatus == batchcandidateexpansion.ReadyBlockedStatus && record.TargetCandidateCount < 60 {
			t.Fatalf("line=%d ready target=%d, want at least 60 from expansion strategy", entry.Line, record.TargetCandidateCount)
		}
		if len(candidateIntentIDs) < record.TargetCandidateCount {
			t.Fatalf("line=%d selected=%d target=%d", entry.Line, len(candidateIntentIDs), record.TargetCandidateCount)
		}
		if !record.PaidIntentRequired || !record.CTAContextRequired || !record.SourceURLAuditRequired || !record.SourceSpecificityRequired {
			t.Fatalf("line=%d missing required expansion guards", entry.Line)
		}
		if record.IndexPolicy != "noindex" || record.ManifestAllowed || record.RenderAllowed || record.SitemapAllowed || record.PublicationAllowed || record.PublicPath != "" {
			t.Fatalf("line=%d readiness escaped blocked contract: index=%q manifest=%t render=%t sitemap=%t publication=%t public_path=%q", entry.Line, record.IndexPolicy, record.ManifestAllowed, record.RenderAllowed, record.SitemapAllowed, record.PublicationAllowed, record.PublicPath)
		}
		if record.LegalArea == "" || record.TargetCandidateTier == "" || len(record.ActionableBlockers) == 0 {
			t.Fatalf("line=%d missing area, tier or actionable blockers", entry.Line)
		}
	}
}

func TestBatchCandidateExpansionReadinessRecordsStayLightweight(t *testing.T) {
	contract, err := storage.LoadContract(".")
	if err != nil {
		t.Fatal(err)
	}
	layer, ok := contract.LayerByName("batch_candidate_expansion_readiness")
	if !ok {
		t.Fatal("missing batch_candidate_expansion_readiness storage layer")
	}
	records, loadReport := batchcandidateexpansion.LoadRecords(".")
	if !loadReport.Passed() {
		t.Fatalf("could not load expansion readiness records: %v", loadReport.Messages())
	}
	index, indexReport := batchcandidateexpansion.BuildExpansionIndex(".")
	if !indexReport.Passed() {
		t.Fatalf("could not build expansion index: %v", indexReport.Messages())
	}
	for _, entry := range records {
		record := entry.Record
		encoded, err := json.Marshal(record)
		if err != nil {
			t.Fatalf("line=%d marshal failed: %v", entry.Line, err)
		}
		if len(encoded)+1 > layer.RecordMaxBytes {
			t.Fatalf("line=%d encoded bytes=%d, want <=%d", entry.Line, len(encoded)+1, layer.RecordMaxBytes)
		}
		if record.ExpansionCandidateSelector != batchcandidateexpansion.ExpansionCandidateSelectorArchivePrefix {
			t.Fatalf("line=%d selector=%q", entry.Line, record.ExpansionCandidateSelector)
		}
		if len(record.ExpansionCandidateIntentIDs) != 0 {
			t.Fatalf("line=%d stores full expansion intents=%d, want selector plus sample only", entry.Line, len(record.ExpansionCandidateIntentIDs))
		}
		if len(record.ExpansionCandidateSampleIntentIDs) == 0 || len(record.ExpansionCandidateSampleIntentIDs) > batchcandidateexpansion.MaxExpansionCandidateSampleIntentIDs {
			t.Fatalf("line=%d sample intents=%d", entry.Line, len(record.ExpansionCandidateSampleIntentIDs))
		}
		if len(batchcandidateexpansion.CandidateIntentIDs(record, index)) < record.TargetCandidateCount {
			t.Fatalf("line=%d derived intents below target", entry.Line)
		}
	}
}

func TestBatchCandidateExpansionReadinessRejectsWeakOrPublicExpansion(t *testing.T) {
	index := batchcandidateexpansion.ExpansionIndex{
		ArchiveByIntent: map[string]batchcandidateexpansion.ArchiveDraft{
			"familia-divorcio-consensual-filhos-bens-prova-documental": {
				BatchID:        "batch-familia-digital",
				UniqueIntentID: "familia-divorcio-consensual-filhos-bens-prova-documental",
				HumanScore:     100,
				SourceMatrixID: "familia-divorcio-consensual-filhos-bens",
				CTAContext:     "Origem: familia-divorcio-consensual-filhos-bens-prova-documental; Fonte: https://www.cnj.jus.br/; documentos esperados para triagem por WhatsApp.",
			},
		},
		ArchiveCountByBatch: map[string]int{"batch-familia-digital": 100},
		CurrentCandidateCountByBatch: map[string]int{
			"batch-familia-digital": 3,
		},
		MaxSimilarity: 0.52,
	}
	record := batchcandidateexpansion.Record{
		ReadinessID:                 "bad-expansion",
		BatchID:                     "batch-familia-digital",
		ReadinessStatus:             "ready_to_publish",
		SourceArchivePath:           "data/editorial/outro.jsonl",
		ArchiveRecordsRequired:      100,
		ArchiveRecordsObserved:      10,
		CurrentCandidateCount:       1,
		TargetCandidateCount:        5,
		TargetCandidateTier:         "target_1000",
		ExpansionCandidateIntentIDs: []string{"intent-inexistente"},
		PaidIntentGatePath:          "data/editorial/outro.jsonl",
		PaidIntentPassedCount:       0,
		PaidIntentBlockedCount:      0,
		PaidIntentMissingCount:      0,
		MinimumHumanScore:           70,
		MaxSimilarityAllowed:        0.90,
		MaxSimilarityObserved:       0.52,
		CTAContextRequired:          false,
		SourceURLAuditRequired:      false,
		SourceSpecificityRequired:   false,
		PaidIntentRequired:          false,
		PublicationBlockReason:      "",
		IndexPolicy:                 "index",
		ManifestAllowed:             true,
		RenderAllowed:               true,
		SitemapAllowed:              true,
		PublicationAllowed:          true,
		PublicPath:                  "/temas/familia/",
		CheckedAt:                   "2026-06-09",
	}

	report := batchcandidateexpansion.ValidateRecordAgainstIndex(record, index)
	if report.Passed() {
		t.Fatal("ValidateRecordAgainstIndex passed, want blocked expansion failures")
	}
	for _, code := range []string{
		"batch_candidate_expansion_status_not_blocked",
		"batch_candidate_expansion_wrong_archive_path",
		"batch_candidate_expansion_observed_count_mismatch",
		"batch_candidate_expansion_current_count_mismatch",
		"batch_candidate_expansion_target_too_low",
		"batch_candidate_expansion_too_few_intents",
		"batch_candidate_expansion_intent_missing",
		"batch_candidate_expansion_paid_gate_path_invalid",
		"batch_candidate_expansion_paid_gate_missing",
		"batch_candidate_expansion_paid_counts_mismatch",
		"batch_candidate_expansion_missing_legal_area",
		"batch_candidate_expansion_target_tier_invalid",
		"batch_candidate_expansion_missing_actionable_blocker",
		"batch_candidate_expansion_invalid_index_policy",
		"batch_candidate_expansion_manifest_allowed",
		"batch_candidate_expansion_similarity_limit_invalid",
		"batch_candidate_expansion_minimum_score_too_low",
		"batch_candidate_expansion_without_cta_requirement",
		"batch_candidate_expansion_without_source_audit",
		"batch_candidate_expansion_without_source_specificity",
		"batch_candidate_expansion_without_paid_intent",
		"batch_candidate_expansion_missing_block_reason",
		"batch_candidate_expansion_render_allowed",
		"batch_candidate_expansion_sitemap_allowed",
		"batch_candidate_expansion_publication_allowed",
		"batch_candidate_expansion_has_public_path",
	} {
		if !report.HasIssue(code) {
			t.Fatalf("missing issue %q in %v", code, report.Codes())
		}
	}
}

func TestBatchCandidateExpansionRefreshUsesCurrentArchiveCounts(t *testing.T) {
	record := batchcandidateexpansion.Record{
		ReadinessID:                 "candidate-expansion-familia",
		BatchID:                     "batch-familia-digital",
		LegalArea:                   "familia",
		ReadinessStatus:             batchcandidateexpansion.ReadyBlockedStatus,
		SourceArchivePath:           batchcandidateexpansion.ArchivePath,
		ArchiveRecordsRequired:      100,
		ArchiveRecordsObserved:      100,
		CurrentCandidateCount:       100,
		TargetCandidateCount:        100,
		ExpansionCandidateIntentIDs: []string{"familia-divorcio-consensual-filhos-bens"},
		PaidIntentGatePath:          batchcandidateexpansion.PaidIntentGatePath,
		MinimumHumanScore:           88,
		MaxSimilarityAllowed:        0.64,
		MaxSimilarityObserved:       0.64,
		CTAContextRequired:          true,
		SourceURLAuditRequired:      true,
		SourceSpecificityRequired:   true,
		PaidIntentRequired:          true,
		ActionableBlockers:          []string{"batch_candidate_gate_pending"},
		NextGate:                    "batch_candidate_gates",
		PublicationBlockReason:      "bloqueado",
		IndexPolicy:                 "noindex",
		CheckedAt:                   "2026-06-09",
	}
	index := batchcandidateexpansion.ExpansionIndex{
		ArchiveCountByBatch:          map[string]int{"batch-familia-digital": 130},
		CurrentCandidateCountByBatch: map[string]int{"batch-familia-digital": 100},
		MaxSimilarity:                0.60,
	}

	refreshed := batchcandidateexpansion.RefreshRecordAgainstIndex(record, index)
	if refreshed.ArchiveRecordsObserved != 130 {
		t.Fatalf("archive_records_observed=%d, want current archive count 130", refreshed.ArchiveRecordsObserved)
	}
	if refreshed.MaxSimilarityObserved != 0.60 {
		t.Fatalf("max_similarity_observed=%.2f, want current archive similarity 0.60", refreshed.MaxSimilarityObserved)
	}
}

func TestBatchCandidateExpansionRefreshRecomputesTargetsWhenArchiveGrows(t *testing.T) {
	intentIDs := make([]string, 0, 160)
	for index := 1; index <= 160; index++ {
		intentIDs = append(intentIDs, fmt.Sprintf("familia-intencao-%03d", index))
	}
	record := batchcandidateexpansion.Record{
		ReadinessID:                 "candidate-expansion-familia",
		BatchID:                     "batch-familia-digital",
		LegalArea:                   "familia",
		ReadinessStatus:             batchcandidateexpansion.ReadyBlockedStatus,
		SourceArchivePath:           batchcandidateexpansion.ArchivePath,
		ArchiveRecordsRequired:      100,
		ArchiveRecordsObserved:      130,
		CurrentCandidateCount:       130,
		TargetCandidateCount:        130,
		TargetCandidateTier:         "target_130",
		ExpansionCandidateIntentIDs: intentIDs[:130],
		PaidIntentGatePath:          batchcandidateexpansion.PaidIntentGatePath,
		MinimumHumanScore:           88,
		MaxSimilarityAllowed:        0.64,
		MaxSimilarityObserved:       0.60,
		CTAContextRequired:          true,
		SourceURLAuditRequired:      true,
		SourceSpecificityRequired:   true,
		PaidIntentRequired:          true,
		ActionableBlockers:          []string{"batch_candidate_gate_pending"},
		NextGate:                    "batch_candidate_gates",
		PublicationBlockReason:      "bloqueado",
		IndexPolicy:                 "noindex",
		CheckedAt:                   "2026-06-09",
	}
	index := batchcandidateexpansion.ExpansionIndex{
		ArchiveCountByBatch:          map[string]int{"batch-familia-digital": 160},
		ArchiveIntentIDsByBatch:      map[string][]string{"batch-familia-digital": intentIDs},
		CurrentCandidateCountByBatch: map[string]int{"batch-familia-digital": 130},
		MaxSimilarity:                0.59,
	}

	refreshed := batchcandidateexpansion.RefreshRecordAgainstIndex(record, index)
	if refreshed.TargetCandidateCount != 160 {
		t.Fatalf("target_candidate_count=%d, want 160 from archive growth and max step", refreshed.TargetCandidateCount)
	}
	if refreshed.TargetCandidateTier != "target_160" {
		t.Fatalf("target_candidate_tier=%q, want target_160", refreshed.TargetCandidateTier)
	}
	if len(refreshed.ExpansionCandidateIntentIDs) != 0 {
		t.Fatalf("stored expansion intents=%d, want selector plus sample only", len(refreshed.ExpansionCandidateIntentIDs))
	}
	derivedIntentIDs := batchcandidateexpansion.CandidateIntentIDs(refreshed, index)
	if len(derivedIntentIDs) != 160 {
		t.Fatalf("derived expansion intents=%d, want 160 after archive growth", len(derivedIntentIDs))
	}
	if derivedIntentIDs[159] != "familia-intencao-160" {
		t.Fatalf("last expansion intent=%q", derivedIntentIDs[159])
	}
	if len(refreshed.ExpansionCandidateSampleIntentIDs) != batchcandidateexpansion.MaxExpansionCandidateSampleIntentIDs {
		t.Fatalf("sample intents=%d", len(refreshed.ExpansionCandidateSampleIntentIDs))
	}
}

func TestBatchCandidateExpansionRefreshDoesNotKeepStaleZeroableMetrics(t *testing.T) {
	record := batchcandidateexpansion.Record{
		ReadinessID:                 "candidate-expansion-familia",
		BatchID:                     "batch-familia-digital",
		LegalArea:                   "familia",
		ReadinessStatus:             batchcandidateexpansion.ReadyBlockedStatus,
		SourceArchivePath:           batchcandidateexpansion.ArchivePath,
		ArchiveRecordsRequired:      100,
		ArchiveRecordsObserved:      130,
		CurrentCandidateCount:       100,
		TargetCandidateCount:        130,
		ExpansionCandidateIntentIDs: []string{"familia-divorcio-consensual-filhos-bens"},
		PaidIntentGatePath:          batchcandidateexpansion.PaidIntentGatePath,
		MinimumHumanScore:           88,
		MaxSimilarityAllowed:        0.64,
		MaxSimilarityObserved:       0.60,
		CTAContextRequired:          true,
		SourceURLAuditRequired:      true,
		SourceSpecificityRequired:   true,
		PaidIntentRequired:          true,
		ActionableBlockers:          []string{"batch_candidate_gate_pending"},
		NextGate:                    "batch_candidate_gates",
		PublicationBlockReason:      "bloqueado",
		IndexPolicy:                 "noindex",
		CheckedAt:                   "2026-06-09",
	}
	index := batchcandidateexpansion.ExpansionIndex{
		ArchiveCountByBatch:          map[string]int{"batch-familia-digital": 0},
		CurrentCandidateCountByBatch: map[string]int{"batch-familia-digital": 0},
		MaxSimilarity:                0,
	}

	refreshed := batchcandidateexpansion.RefreshRecordAgainstIndex(record, index)
	if refreshed.ArchiveRecordsObserved != 0 {
		t.Fatalf("archive_records_observed=%d, want 0 from current index", refreshed.ArchiveRecordsObserved)
	}
	if refreshed.MaxSimilarityObserved != 0 {
		t.Fatalf("max_similarity_observed=%.2f, want 0 from current index", refreshed.MaxSimilarityObserved)
	}
	if refreshed.CurrentCandidateCount != 0 {
		t.Fatalf("current_candidate_count=%d, want 0 from current index", refreshed.CurrentCandidateCount)
	}
}

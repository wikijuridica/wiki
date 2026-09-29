package contract_test

import (
	"encoding/json"
	"testing"

	"portaljuridico/internal/batchcandidateexpansion"
	"portaljuridico/internal/batchcandidategates"
	"portaljuridico/internal/batchcandidatepromotion"
	"portaljuridico/internal/batchexpansionstrategy"
	"portaljuridico/internal/paidintent"
	"portaljuridico/internal/storage"
)

func TestBatchCandidateGatesSelectArchiveDraftsWithoutPublishing(t *testing.T) {
	report := batchcandidategates.Validate(".")
	if !report.Passed() {
		t.Fatalf("batch candidate gates failed contract: %v", report.Messages())
	}

	records, loadReport := batchcandidategates.LoadRecords(".")
	if !loadReport.Passed() {
		t.Fatalf("could not load batch candidate gates: %v", loadReport.Messages())
	}
	selectedByBatch := make(map[string][]string)
	metadataByBatch := make(map[string]batchcandidategates.Record)
	totalSelected := 0
	for _, entry := range records {
		record := entry.Record
		selectedByBatch[record.BatchID] = append(selectedByBatch[record.BatchID], record.SelectedUniqueIntentIDs...)
		if _, ok := metadataByBatch[record.BatchID]; !ok {
			metadataByBatch[record.BatchID] = record
		}
		totalSelected += len(record.SelectedUniqueIntentIDs)
		if record.RenderAllowed || record.SitemapAllowed || record.PublicationAllowed || record.PublicPath != "" {
			t.Fatalf("%s escaped blocked gate: render=%t sitemap=%t publication=%t public_path=%q", record.BatchID, record.RenderAllowed, record.SitemapAllowed, record.PublicationAllowed, record.PublicPath)
		}
		if record.BaseURLMode != "official_configured" || !record.OfficialURLLocked {
			t.Fatalf("%s must track locked official URL while preserving blocked publication, mode=%q locked=%t", record.BatchID, record.BaseURLMode, record.OfficialURLLocked)
		}
	}
	if len(selectedByBatch) != 6 {
		t.Fatalf("candidate gate batch families=%d, want 6", len(selectedByBatch))
	}
	readinessRecords, readinessReport := batchcandidateexpansion.LoadRecords(".")
	if !readinessReport.Passed() {
		t.Fatalf("could not load expansion readiness: %v", readinessReport.Messages())
	}
	expansionIndex, expansionIndexReport := batchcandidateexpansion.BuildExpansionIndex(".")
	if !expansionIndexReport.Passed() {
		t.Fatalf("could not build expansion index: %v", expansionIndexReport.Messages())
	}
	readinessByBatch := make(map[string]batchcandidateexpansion.Record)
	for _, entry := range readinessRecords {
		readinessByBatch[entry.Record.BatchID] = entry.Record
	}
	paidRecords, paidReport := paidintent.LoadRecords(".")
	if !paidReport.Passed() {
		t.Fatalf("could not load paid intent gates: %v", paidReport.Messages())
	}
	paidByIntent := make(map[string]paidintent.Record)
	for _, entry := range paidRecords {
		paidByIntent[entry.Record.UniqueIntentID] = entry.Record
	}
	strategyRecords, strategyReport := batchexpansionstrategy.LoadRecords(".")
	if !strategyReport.Passed() {
		t.Fatalf("could not load expansion strategy: %v", strategyReport.Messages())
	}
	strategyByBatch := make(map[string]batchexpansionstrategy.Record)
	for _, entry := range strategyRecords {
		strategyByBatch[entry.Record.BatchID] = entry.Record
	}

	totalExpectedCurrent := 0
	for batchID, selectedIntents := range selectedByBatch {
		record := metadataByBatch[batchID]
		readiness, ok := readinessByBatch[record.BatchID]
		if !ok {
			t.Fatalf("%s missing expansion readiness", record.BatchID)
		}
		strategy, ok := strategyByBatch[record.BatchID]
		if !ok {
			t.Fatalf("%s missing expansion strategy", record.BatchID)
		}
		totalExpectedCurrent += strategy.CurrentCandidateCount
		expectedPaidPassed := make([]string, 0)
		for _, intentID := range batchcandidateexpansion.CandidateIntentIDs(readiness, expansionIndex) {
			if paidintent.AllowsExpansion(paidByIntent[intentID]) {
				expectedPaidPassed = append(expectedPaidPassed, intentID)
			}
			if len(expectedPaidPassed) == strategy.CurrentCandidateCount {
				break
			}
		}
		if !sameStringSet(selectedIntents, expectedPaidPassed) {
			t.Fatalf("%s selected intents do not match strategy current paid-passed candidates: selected=%d expected=%d", record.BatchID, len(selectedIntents), len(expectedPaidPassed))
		}
		if len(selectedIntents) < 18 {
			t.Fatalf("%s selected intents=%d, want at least 18 paid-passed expansion candidates", record.BatchID, len(selectedIntents))
		}
		if len(selectedIntents) != strategy.CurrentCandidateCount {
			t.Fatalf("%s selected intents=%d, want materialized strategy current count=%d", record.BatchID, len(selectedIntents), strategy.CurrentCandidateCount)
		}
		if strategy.StrategyStatus == batchexpansionstrategy.ReadyNextCandidateGateStatus && strategy.NextCandidateTarget <= strategy.CurrentCandidateCount {
			t.Fatalf("%s next target=%d, want growth beyond current=%d", record.BatchID, strategy.NextCandidateTarget, strategy.CurrentCandidateCount)
		}
	}
	if totalSelected != totalExpectedCurrent {
		t.Fatalf("selected intents=%d, want materialized strategy current total=%d", totalSelected, totalExpectedCurrent)
	}
}

func TestBatchCandidateGateRecordsStayWithinShardBudget(t *testing.T) {
	contract, err := storage.LoadContract(".")
	if err != nil {
		t.Fatal(err)
	}
	layer, ok := contract.LayerByName("batch_candidate_gates")
	if !ok {
		t.Fatal("missing batch_candidate_gates storage layer")
	}
	records, loadReport := batchcandidategates.LoadRecords(".")
	if !loadReport.Passed() {
		t.Fatalf("could not load batch candidate gates: %v", loadReport.Messages())
	}
	for _, entry := range records {
		record := entry.Record
		if len(record.SelectedUniqueIntentIDs) > batchcandidategates.MaxSelectedIntentIDsPerRecord {
			t.Fatalf("line %d %s selected intents=%d, want <=%d", entry.Line, record.GateID, len(record.SelectedUniqueIntentIDs), batchcandidategates.MaxSelectedIntentIDsPerRecord)
		}
		if record.ShardCount > 1 {
			if record.GateGroupID == "" || record.ShardIndex <= 0 || record.SelectedTotal <= len(record.SelectedUniqueIntentIDs) {
				t.Fatalf("line %d %s has invalid shard metadata: group=%q index=%d count=%d total=%d selected=%d", entry.Line, record.GateID, record.GateGroupID, record.ShardIndex, record.ShardCount, record.SelectedTotal, len(record.SelectedUniqueIntentIDs))
			}
		}
		data, err := json.Marshal(record)
		if err != nil {
			t.Fatalf("could not marshal line %d %s: %v", entry.Line, record.GateID, err)
		}
		if len(data)+1 > layer.RecordMaxBytes {
			t.Fatalf("line %d %s encoded bytes=%d, want <=%d", entry.Line, record.GateID, len(data)+1, layer.RecordMaxBytes)
		}
	}
}

func TestBatchCandidatePromotionUsesDerivedReadinessTargets(t *testing.T) {
	records, report := batchcandidatepromotion.AdvanceToNextTargets(".")
	if !report.Passed() {
		t.Fatalf("promotion from derived readiness failed: %v", report.Messages())
	}
	totalSelected := 0
	for _, record := range records {
		totalSelected += len(record.SelectedUniqueIntentIDs)
		if len(record.SelectedUniqueIntentIDs) == 0 {
			t.Fatalf("%s selected no intents from derived readiness", record.BatchID)
		}
		if record.RenderAllowed || record.SitemapAllowed || record.PublicationAllowed || record.PublicPath != "" {
			t.Fatalf("%s promotion escaped blocked publication", record.BatchID)
		}
	}
	strategyRecords, strategyReport := batchexpansionstrategy.LoadRecords(".")
	if !strategyReport.Passed() {
		t.Fatalf("could not load expansion strategy: %v", strategyReport.Messages())
	}
	expected := 0
	for _, entry := range strategyRecords {
		expected += entry.Record.CurrentCandidateCount
	}
	if totalSelected != expected {
		t.Fatalf("promotion selected=%d, want current strategy total=%d", totalSelected, expected)
	}
}

func TestBatchCandidateGateRejectsPublicOrUnknownArchiveIntent(t *testing.T) {
	record := batchcandidategates.Record{
		GateID:                  "candidate-saude",
		BatchID:                 "batch-saude-suplementar-digital",
		GateStatus:              "blocked_batch_candidate",
		SourceArchivePath:       "data/editorial/batch_draft_expansion_archive.jsonl",
		ArchiveMinimumRecords:   100,
		SelectedUniqueIntentIDs: []string{"intent-inexistente"},
		CandidatePathPrefix:     "/temas/",
		BaseURLMode:             "lab_placeholder",
		OfficialURLLocked:       false,
		MaxSimilarityAllowed:    0.64,
		MaxSimilarityObserved:   0.64,
		MinimumHumanScore:       88,
		CTAContextRequired:      true,
		SourceURLAuditRequired:  true,
		PublicationBlockReason:  "",
		RenderAllowed:           true,
		SitemapAllowed:          true,
		PublicationAllowed:      true,
		PublicPath:              "/temas/intent-inexistente/",
		CheckedAt:               "2026-06-09",
	}

	report := batchcandidategates.ValidateRecordAgainstArchive(record, batchcandidategates.ArchiveIndex{
		BaseURLMode:       "official_configured",
		OfficialURLLocked: true,
	})
	for _, code := range []string{
		"batch_candidate_selected_intent_missing",
		"batch_candidate_render_allowed",
		"batch_candidate_sitemap_allowed",
		"batch_candidate_publication_allowed",
		"batch_candidate_has_public_path",
		"batch_candidate_missing_block_reason",
		"batch_candidate_url_config_mismatch",
	} {
		if !report.HasIssue(code) {
			t.Fatalf("missing issue %q in %v", code, report.Codes())
		}
	}
}

func sameStringSet(left []string, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	seen := make(map[string]int)
	for _, value := range left {
		seen[value]++
	}
	for _, value := range right {
		seen[value]--
	}
	for _, count := range seen {
		if count != 0 {
			return false
		}
	}
	return true
}

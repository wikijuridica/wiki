package batchcandidategates

import (
	"fmt"
	"testing"
)

func TestNormalizeShardRecordsIsIdempotentForExistingShards(t *testing.T) {
	base := Record{
		GateID:                  "candidate-familia-digital",
		BatchID:                 "batch-familia-digital",
		SelectedUniqueIntentIDs: makeIntentIDs(220),
	}

	first := NormalizeShardRecords([]Record{base})
	second := NormalizeShardRecords(first)

	if len(second) != len(first) {
		t.Fatalf("normalized shard count=%d, want %d", len(second), len(first))
	}
	seenGateIDs := make(map[string]bool)
	total := 0
	for index, record := range second {
		if record.GateGroupID != "candidate-familia-digital" {
			t.Fatalf("shard %d group=%q, want candidate-familia-digital", index+1, record.GateGroupID)
		}
		if record.ShardIndex != index+1 || record.ShardCount != len(first) || record.SelectedTotal != 220 {
			t.Fatalf("shard %d metadata index=%d count=%d total=%d", index+1, record.ShardIndex, record.ShardCount, record.SelectedTotal)
		}
		if seenGateIDs[record.GateID] {
			t.Fatalf("duplicate gate id after idempotent normalize: %s", record.GateID)
		}
		seenGateIDs[record.GateID] = true
		if len(record.SelectedUniqueIntentIDs) > MaxSelectedIntentIDsPerRecord {
			t.Fatalf("shard %d selected=%d, want <=%d", index+1, len(record.SelectedUniqueIntentIDs), MaxSelectedIntentIDsPerRecord)
		}
		total += len(record.SelectedUniqueIntentIDs)
	}
	if total != 220 {
		t.Fatalf("selected total=%d, want 220", total)
	}
}

func makeIntentIDs(total int) []string {
	values := make([]string, 0, total)
	for index := 0; index < total; index++ {
		values = append(values, fmt.Sprintf("intent-%03d", index+1))
	}
	return values
}

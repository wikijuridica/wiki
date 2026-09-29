package contract_test

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"portaljuridico/internal/batchdraftarchive"
	"portaljuridico/internal/batchdraftgen"
	"portaljuridico/internal/batchdrafts"
)

func TestBatchDraftGeneratorProducesDeterministicBlockedScoredDrafts(t *testing.T) {
	options := batchdraftgen.Options{SamplesPerBatch: 5, CheckedAt: "2026-06-09"}
	result, report := batchdraftgen.Generate(".", options)
	if !report.Passed() {
		t.Fatalf("batch draft generation failed: %v", report.Messages())
	}

	again, secondReport := batchdraftgen.Generate(".", options)
	if !secondReport.Passed() {
		t.Fatalf("second generation failed: %v", secondReport.Messages())
	}
	if !reflect.DeepEqual(result.DraftIDs(), again.DraftIDs()) {
		t.Fatalf("generation is not deterministic\nfirst=%v\nsecond=%v", result.DraftIDs(), again.DraftIDs())
	}

	if len(result.Drafts) < 30 {
		t.Fatalf("generated drafts=%d, want at least 30", len(result.Drafts))
	}
	if len(result.Metrics) < 6 {
		t.Fatalf("generation metrics=%d, want at least 6", len(result.Metrics))
	}
	if result.RewrittenCount() < 6 {
		t.Fatalf("rewritten drafts=%d, want at least one automatic rewrite per batch", result.RewrittenCount())
	}
	if result.MaximumPairSimilarity() > 0.64 {
		t.Fatalf("max generated similarity=%.2f, want <=0.64", result.MaximumPairSimilarity())
	}

	for _, draft := range result.Drafts {
		if validation := batchdrafts.ValidateRecord(draft); !validation.Passed() {
			t.Fatalf("generated draft %s failed batch draft contract: %v", draft.UniqueIntentID, validation.Messages())
		}
		if draft.RenderAllowed || draft.SitemapAllowed || draft.PublicationAllowed || draft.PublicPath != "" {
			t.Fatalf("generated draft %s escaped blocked-publication lab contract", draft.UniqueIntentID)
		}
		if draft.CTAContext == "" || draft.UniqueIntentID == "" || !batchdraftgen.ContainsOrigin(draft.CTAContext, draft.UniqueIntentID) {
			t.Fatalf("generated draft %s lacks contextual WhatsApp origin: %q", draft.UniqueIntentID, draft.CTAContext)
		}
	}

	metricReport := batchdraftgen.ValidateMetrics(result.Metrics)
	if !metricReport.Passed() {
		t.Fatalf("generated metrics failed contract: %v", metricReport.Messages())
	}

	storedMetricReport := batchdraftgen.ValidateStoredMetrics(".")
	if !storedMetricReport.Passed() {
		t.Fatalf("stored generation metrics failed contract: %v", storedMetricReport.Messages())
	}
}

func TestBatchDraftGenerationRejectsUnsafeOptionsAndPublicMetrics(t *testing.T) {
	_, report := batchdraftgen.Generate(".", batchdraftgen.Options{SamplesPerBatch: 1, CheckedAt: "2026-06-09"})
	if !report.HasIssue("generation_samples_too_low") {
		t.Fatalf("missing generation_samples_too_low in %v", report.Codes())
	}

	metric := batchdraftgen.Metric{
		BatchID:            "batch-saude-suplementar-digital",
		GenerationStatus:   "published",
		GeneratedSamples:   1,
		PassedSamples:      0,
		MinimumHumanScore:  40,
		MaximumAILikeScore: 90,
		MaximumSimilarity:  0.91,
		RenderAllowed:      true,
		SitemapAllowed:     true,
		PublicationAllowed: true,
		PublicPath:         "/temas/unsafe/",
		CheckedAt:          "2026-06-09",
	}
	report = batchdraftgen.ValidateMetric(metric)
	for _, code := range []string{
		"generation_metric_invalid_status",
		"generation_metric_too_few_samples",
		"generation_metric_unpassed_samples",
		"generation_metric_human_score_too_low",
		"generation_metric_ai_score_too_high",
		"generation_metric_similarity_too_high",
		"generation_metric_render_allowed",
		"generation_metric_sitemap_allowed",
		"generation_metric_publication_allowed",
		"generation_metric_has_public_path",
	} {
		if !report.HasIssue(code) {
			t.Fatalf("missing issue %q in %v", code, report.Codes())
		}
	}
}

func TestBatchDraftGeneratorPersistsExpandedArchiveThroughStorageContract(t *testing.T) {
	result, report := batchdraftgen.Generate(".", batchdraftgen.Options{SamplesPerBatch: 130, CheckedAt: "2026-06-09"})
	if !report.Passed() {
		t.Fatalf("expanded generation failed before archive persistence: %v", report.Messages())
	}

	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module archive_persistence_test\n"), 0644); err != nil {
		t.Fatal(err)
	}
	contentDir := filepath.Join(root, "content")
	if err := os.MkdirAll(contentDir, 0755); err != nil {
		t.Fatal(err)
	}
	contract, err := os.ReadFile(filepath.Join("..", "..", "content", "storage_contract.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(contentDir, "storage_contract.json"), contract, 0644); err != nil {
		t.Fatal(err)
	}

	if err := batchdraftgen.WriteArchive(root, result.Drafts); err != nil {
		t.Fatalf("WriteArchive failed: %v", err)
	}
	archiveReport := batchdraftarchive.Validate(root)
	if !archiveReport.Passed() {
		t.Fatalf("persisted expansion archive failed contract: %v", archiveReport.Messages())
	}
	entries, loadReport := batchdraftarchive.LoadRecords(root)
	if !loadReport.Passed() {
		t.Fatalf("persisted expansion archive could not be loaded: %v", loadReport.Messages())
	}
	if len(entries) < 780 {
		t.Fatalf("persisted archive entries=%d, want at least 780", len(entries))
	}
}

func TestBatchDraftGeneratorWriteArchiveRejectsDuplicateBeforeWriting(t *testing.T) {
	result, report := batchdraftgen.Generate(".", batchdraftgen.Options{SamplesPerBatch: 130, CheckedAt: "2026-06-09"})
	if !report.Passed() {
		t.Fatalf("expanded generation failed before duplicate write test: %v", report.Messages())
	}
	root := tempArchiveRoot(t)
	existingPath := filepath.Join(root, "data", "editorial", "batch_draft_expansion_archive.jsonl")
	existing := []byte("existing-safe-archive\n")
	if err := os.WriteFile(existingPath, existing, 0644); err != nil {
		t.Fatal(err)
	}

	unsafe := append([]batchdrafts.Record{}, result.Drafts...)
	unsafe[1].UniqueIntentID = unsafe[0].UniqueIntentID
	unsafe[1].ReaderProblem = unsafe[1].ReaderProblem + " Documento extra apenas para provar que duplicidade de intencao nao pode ser mascarada por texto diferente."
	if err := batchdraftgen.WriteArchive(root, unsafe); err == nil {
		t.Fatal("WriteArchive accepted duplicate unique_intent_id, want failure before writing")
	}
	after, err := os.ReadFile(existingPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(after) != string(existing) {
		t.Fatalf("archive changed after rejected duplicate write: got %q want %q", string(after), string(existing))
	}
}

func TestBatchDraftGeneratorWriteArchiveValidatesPayloadBeforeTruncating(t *testing.T) {
	result, report := batchdraftgen.Generate(".", batchdraftgen.Options{SamplesPerBatch: 130, CheckedAt: "2026-06-09"})
	if !report.Passed() {
		t.Fatalf("expanded generation failed before oversized write test: %v", report.Messages())
	}
	root := tempArchiveRoot(t)
	existingPath := filepath.Join(root, "data", "editorial", "batch_draft_expansion_archive.jsonl")
	existing := []byte("existing-safe-archive\n")
	if err := os.WriteFile(existingPath, existing, 0644); err != nil {
		t.Fatal(err)
	}

	unsafe := append([]batchdrafts.Record{}, result.Drafts...)
	unsafe[0].CheckedAt = "2026-06-09" + strings.Repeat("x", 70000)
	if err := batchdraftgen.WriteArchive(root, unsafe); err == nil {
		t.Fatal("WriteArchive accepted oversized payload, want failure before truncating archive")
	}
	after, err := os.ReadFile(existingPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(after) != string(existing) {
		t.Fatalf("archive changed after rejected oversized write: got %q want %q", string(after), string(existing))
	}
}

func tempArchiveRoot(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module archive_write_safety_test\n"), 0644); err != nil {
		t.Fatal(err)
	}
	contentDir := filepath.Join(root, "content")
	if err := os.MkdirAll(contentDir, 0755); err != nil {
		t.Fatal(err)
	}
	contract, err := os.ReadFile(filepath.Join("..", "..", "content", "storage_contract.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(contentDir, "storage_contract.json"), contract, 0644); err != nil {
		t.Fatal(err)
	}
	archiveDir := filepath.Join(root, "data", "editorial")
	if err := os.MkdirAll(archiveDir, 0755); err != nil {
		t.Fatal(err)
	}
	return root
}

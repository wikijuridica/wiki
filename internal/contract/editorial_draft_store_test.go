package contract_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"portaljuridico/internal/draftlab"
	"portaljuridico/internal/editorialdrafts"
	"portaljuridico/internal/terms"
)

func TestEditorialDraftStorePersistsNoindexDraftWithoutPublicPath(t *testing.T) {
	root := copyStorageFixture(t)
	seed := terms.Seed{
		TermID:       "responsabilidade-civil",
		Term:         "responsabilidade civil",
		Language:     "pt-BR",
		QualityState: "draft_only",
		SourceID:     "lexml",
		SourceURL:    "https://www.lexml.gov.br/",
		CheckedAt:    "2026-06-09",
		IntentHint:   "explicar o conceito juridico em linguagem informativa antes de qualquer publicacao",
	}
	draft, err := draftlab.Build(seed)
	if err != nil {
		t.Fatal(err)
	}

	if err := editorialdrafts.Append(root, draft); err != nil {
		t.Fatal(err)
	}

	report := editorialdrafts.Validate(root)
	if !report.Passed() {
		t.Fatalf("editorial draft validation failed: %v", report.Messages())
	}
	data := readFile(t, filepath.Join(root, "data", "editorial", "drafts.jsonl"))
	for _, token := range []string{`"status":"draft"`, `"index_policy":"noindex"`, `"public_path":""`, `"source_id":"lexml"`} {
		if !strings.Contains(data, token) {
			t.Fatalf("draft store missing %q in %s", token, data)
		}
	}
}

func copyStorageFixture(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	writeTestFile(t, filepath.Join(root, "go.mod"), "module fixture\n")
	for _, path := range []string{
		"content/storage_contract.json",
		"data/research/high_intent_terms.jsonl",
		"data/terms/legal_terms.jsonl",
		"data/terms/intent_candidates.jsonl",
		"data/source-audit/robots_terms.jsonl",
		"data/source-audit/batch_source_urls.jsonl",
		"data/source-snapshots/payloads.jsonl",
		"data/editorial/drafts.jsonl",
		"data/editorial/content_briefs.jsonl",
		"data/editorial/authorial_drafts.jsonl",
		"data/editorial/review_queue.jsonl",
		"data/editorial/approved_drafts.jsonl",
		"data/editorial/publication_blockers.jsonl",
		"data/editorial/source_blockers.jsonl",
		"data/editorial/source_resolutions.jsonl",
		"data/editorial/prepublication_gates.jsonl",
		"data/editorial/legal_reviews.jsonl",
		"data/editorial/human_content_scores.jsonl",
		"data/editorial/scalable_content_batches.jsonl",
		"data/editorial/batch_drafts.jsonl",
		"data/editorial/batch_draft_expansion_archive.jsonl",
		"data/editorial/batch_candidate_gates.jsonl",
		"data/editorial/batch_generation_metrics.jsonl",
		"data/editorial/batch_source_matrix.jsonl",
		"data/editorial/published_manifest.jsonl",
	} {
		data, err := os.ReadFile(filepath.Join(findRoot(t), path))
		if err != nil {
			t.Fatal(err)
		}
		if path == "data/editorial/drafts.jsonl" || path == "data/editorial/content_briefs.jsonl" || path == "data/editorial/authorial_drafts.jsonl" || path == "data/editorial/review_queue.jsonl" || path == "data/editorial/approved_drafts.jsonl" || path == "data/editorial/publication_blockers.jsonl" || path == "data/editorial/source_blockers.jsonl" || path == "data/editorial/source_resolutions.jsonl" || path == "data/editorial/prepublication_gates.jsonl" || path == "data/editorial/legal_reviews.jsonl" || path == "data/editorial/human_content_scores.jsonl" || path == "data/editorial/scalable_content_batches.jsonl" || path == "data/editorial/batch_drafts.jsonl" || path == "data/editorial/batch_draft_expansion_archive.jsonl" || path == "data/editorial/batch_candidate_gates.jsonl" || path == "data/editorial/batch_generation_metrics.jsonl" || path == "data/editorial/batch_source_matrix.jsonl" {
			data = []byte{}
		}
		writeTestFile(t, filepath.Join(root, path), string(data))
	}
	return root
}

func writeTestFile(t *testing.T, path string, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}

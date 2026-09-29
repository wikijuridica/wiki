package storage_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"portaljuridico/internal/storage"
)

func TestAppendJSONLWritesNamedLayerAndRejectsOversizedRecord(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "go.mod"), "module teststorage\n")
	writeFile(t, filepath.Join(root, "content", "storage_contract.json"), `{
  "format": "jsonl_file_store_v1",
  "dependency_policy": "stdlib_only_no_external_database_dependency",
  "production_cpu_policy": "append_only_small_records_no_runtime_full_scan",
  "separates_source_audit_from_editorial": true,
  "separates_term_seed_from_public_content": true,
  "content_start_policy": "term_ingestion_may_seed_draft_only_until_editorial_source_quality_and_seo_gates_pass",
  "layers": [
    {
      "name": "term_seeds",
      "path": "data/terms/legal_terms.jsonl",
      "record_type": "legal_term_seed",
      "record_max_bytes": 128,
      "public_indexable": false,
      "allows_raw_official_text": false,
      "allows_editorial_content": false,
      "requires_source_provenance": true,
      "requires_quality_state": true,
      "description": "test layer"
    }
  ]
}`)
	writeFile(t, filepath.Join(root, "data", "terms", "legal_terms.jsonl"), "")

	err := storage.AppendJSONL(root, "term_seeds", map[string]string{
		"term":          "responsabilidade civil",
		"quality_state": "draft_only",
	})
	if err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(filepath.Join(root, "data", "terms", "legal_terms.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), `"responsabilidade civil"`) {
		t.Fatalf("term seed was not appended: %q", string(data))
	}

	err = storage.AppendJSONL(root, "term_seeds", map[string]string{
		"term": strings.Repeat("x", 200),
	})
	if err == nil || !strings.Contains(err.Error(), "record_too_large") {
		t.Fatalf("oversized record error = %v, want record_too_large", err)
	}
}

func TestValidateRejectsInvalidJSONLRecords(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "go.mod"), "module teststorage\n")
	writeFile(t, filepath.Join(root, "content", "storage_contract.json"), `{
  "format": "jsonl_file_store_v1",
  "dependency_policy": "stdlib_only_no_external_database_dependency",
  "production_cpu_policy": "append_only_small_records_no_runtime_full_scan",
  "separates_source_audit_from_editorial": true,
  "separates_term_seed_from_public_content": true,
  "content_start_policy": "term_ingestion_may_seed_draft_only_until_editorial_source_quality_and_seo_gates_pass",
  "layers": [
    {"name":"term_seeds","path":"data/terms/legal_terms.jsonl","record_type":"legal_term_seed","record_max_bytes":128,"public_indexable":false,"allows_raw_official_text":false,"allows_editorial_content":false,"requires_source_provenance":true,"requires_quality_state":true,"description":"test layer"},
    {"name":"source_audits","path":"data/source-audit/robots_terms.jsonl","record_type":"source_audit_event","record_max_bytes":128,"public_indexable":false,"allows_raw_official_text":false,"allows_editorial_content":false,"requires_source_provenance":true,"requires_quality_state":true,"description":"test layer"},
    {"name":"source_snapshots","path":"data/source-snapshots/payloads.jsonl","record_type":"source_payload_snapshot","record_max_bytes":128,"public_indexable":false,"allows_raw_official_text":true,"allows_editorial_content":false,"requires_source_provenance":true,"requires_quality_state":true,"description":"test layer"},
    {"name":"editorial_drafts","path":"data/editorial/drafts.jsonl","record_type":"editorial_draft","record_max_bytes":128,"public_indexable":false,"allows_raw_official_text":false,"allows_editorial_content":true,"requires_source_provenance":true,"requires_quality_state":true,"description":"test layer"},
    {"name":"published_manifest","path":"data/editorial/published_manifest.jsonl","record_type":"published_content_manifest","record_max_bytes":128,"public_indexable":false,"allows_raw_official_text":false,"allows_editorial_content":false,"requires_source_provenance":true,"requires_quality_state":true,"description":"test layer"}
  ]
}`)
	writeFile(t, filepath.Join(root, "data", "terms", "legal_terms.jsonl"), "{bad json\n")
	writeFile(t, filepath.Join(root, "data", "source-audit", "robots_terms.jsonl"), "")
	writeFile(t, filepath.Join(root, "data", "source-snapshots", "payloads.jsonl"), "")
	writeFile(t, filepath.Join(root, "data", "editorial", "drafts.jsonl"), "")
	writeFile(t, filepath.Join(root, "data", "editorial", "published_manifest.jsonl"), "")

	report := storage.Validate(root)
	if report.Passed() {
		t.Fatal("Validate passed, want invalid JSONL failure")
	}
	messages := strings.Join(report.Messages(), "\n")
	if !strings.Contains(messages, "invalid_jsonl_record") {
		t.Fatalf("missing invalid_jsonl_record in %s", messages)
	}
}

func TestValidateReportsOversizedJSONLRecordInsteadOfScannerFailure(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "go.mod"), "module teststorage\n")
	writeFile(t, filepath.Join(root, "content", "storage_contract.json"), `{
  "format": "jsonl_file_store_v1",
  "dependency_policy": "stdlib_only_no_external_database_dependency",
  "production_cpu_policy": "append_only_small_records_no_runtime_full_scan",
  "separates_source_audit_from_editorial": true,
  "separates_term_seed_from_public_content": true,
  "content_start_policy": "term_ingestion_may_seed_draft_only_until_editorial_source_quality_and_seo_gates_pass",
  "layers": [
    {"name":"term_seeds","path":"data/terms/legal_terms.jsonl","record_type":"legal_term_seed","record_max_bytes":128,"public_indexable":false,"allows_raw_official_text":false,"allows_editorial_content":false,"requires_source_provenance":true,"requires_quality_state":true,"description":"test layer"},
    {"name":"source_audits","path":"data/source-audit/robots_terms.jsonl","record_type":"source_audit_event","record_max_bytes":128,"public_indexable":false,"allows_raw_official_text":false,"allows_editorial_content":false,"requires_source_provenance":true,"requires_quality_state":true,"description":"test layer"},
    {"name":"source_snapshots","path":"data/source-snapshots/payloads.jsonl","record_type":"source_payload_snapshot","record_max_bytes":128,"public_indexable":false,"allows_raw_official_text":true,"allows_editorial_content":false,"requires_source_provenance":true,"requires_quality_state":true,"description":"test layer"},
    {"name":"editorial_drafts","path":"data/editorial/drafts.jsonl","record_type":"editorial_draft","record_max_bytes":128,"public_indexable":false,"allows_raw_official_text":false,"allows_editorial_content":true,"requires_source_provenance":true,"requires_quality_state":true,"description":"test layer"},
    {"name":"published_manifest","path":"data/editorial/published_manifest.jsonl","record_type":"published_content_manifest","record_max_bytes":128,"public_indexable":false,"allows_raw_official_text":false,"allows_editorial_content":false,"requires_source_provenance":true,"requires_quality_state":true,"description":"test layer"}
  ]
}`)
	writeFile(t, filepath.Join(root, "data", "terms", "legal_terms.jsonl"), `{"term":"`+strings.Repeat("x", 5000)+`"}`+"\n")
	writeFile(t, filepath.Join(root, "data", "source-audit", "robots_terms.jsonl"), "")
	writeFile(t, filepath.Join(root, "data", "source-snapshots", "payloads.jsonl"), "")
	writeFile(t, filepath.Join(root, "data", "editorial", "drafts.jsonl"), "")
	writeFile(t, filepath.Join(root, "data", "editorial", "published_manifest.jsonl"), "")

	report := storage.Validate(root)
	if report.Passed() {
		t.Fatal("Validate passed, want oversized JSONL failure")
	}
	messages := strings.Join(report.Messages(), "\n")
	if !strings.Contains(messages, "jsonl_record_too_large") {
		t.Fatalf("missing jsonl_record_too_large in %s", messages)
	}
	if strings.Contains(messages, "scan_layer_file_failed") {
		t.Fatalf("scanner failure masked record budget issue: %s", messages)
	}
}

func writeFile(t *testing.T, path string, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}

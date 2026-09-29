package contract_test

import (
	"path/filepath"
	"strings"
	"testing"

	"portaljuridico/internal/storage"
)

func TestTermIngestionUsesSeparatedLightweightStorage(t *testing.T) {
	contract, err := storage.LoadContract(".")
	if err != nil {
		t.Fatal(err)
	}

	if contract.Format != "jsonl_file_store_v1" {
		t.Fatalf("Format = %q, want jsonl_file_store_v1", contract.Format)
	}
	if contract.DependencyPolicy != "stdlib_only_no_external_database_dependency" {
		t.Fatalf("DependencyPolicy = %q, want stdlib-only storage", contract.DependencyPolicy)
	}
	if !contract.SeparatesSourceAuditFromEditorial {
		t.Fatal("source audit data must be separated from editorial data")
	}
	if !contract.SeparatesTermSeedFromPublicContent {
		t.Fatal("term ingestion must be separated from public content")
	}
	if !strings.Contains(contract.ContentStartPolicy, "draft_only") {
		t.Fatalf("ContentStartPolicy must allow only draft starts from terms, got %q", contract.ContentStartPolicy)
	}

	required := []string{
		"term_seeds",
		"source_audits",
		"source_snapshots",
		"editorial_drafts",
		"published_manifest",
	}
	seenPaths := make(map[string]string)
	for _, name := range required {
		layer, ok := contract.LayerByName(name)
		if !ok {
			t.Fatalf("missing storage layer %q", name)
		}
		if layer.Path == "" || !strings.HasPrefix(layer.Path, "data/") {
			t.Fatalf("layer %q path = %q, want path under data/", name, layer.Path)
		}
		if previous := seenPaths[layer.Path]; previous != "" {
			t.Fatalf("layers %q and %q share path %q", previous, name, layer.Path)
		}
		seenPaths[layer.Path] = name
		requireFile(t, filepath.Join(findRoot(t), layer.Path))
		if layer.RecordMaxBytes <= 0 || layer.RecordMaxBytes > 32768 {
			t.Fatalf("layer %q RecordMaxBytes = %d, want lightweight record budget", name, layer.RecordMaxBytes)
		}
		if layer.PublicIndexable {
			t.Fatalf("storage layer %q cannot be directly indexable", name)
		}
	}

	termSeeds, _ := contract.LayerByName("term_seeds")
	if termSeeds.AllowsRawOfficialText || termSeeds.AllowsEditorialContent {
		t.Fatalf("term seeds must not store raw official text or editorial copy: %+v", termSeeds)
	}
	if !termSeeds.RequiresSourceProvenance || !termSeeds.RequiresQualityState {
		t.Fatalf("term seeds must require provenance and quality state: %+v", termSeeds)
	}

	editorialDrafts, _ := contract.LayerByName("editorial_drafts")
	if !editorialDrafts.AllowsEditorialContent || editorialDrafts.AllowsRawOfficialText {
		t.Fatalf("editorial drafts must store own editorial text, not raw official payloads: %+v", editorialDrafts)
	}
}

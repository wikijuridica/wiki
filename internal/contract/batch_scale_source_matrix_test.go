package contract_test

import (
	"testing"

	"portaljuridico/internal/batchdraftgen"
	"portaljuridico/internal/batchdrafts"
	"portaljuridico/internal/batchsourcematrix"
)

func TestBatchDraftGeneratorScalesWithSourceMatrixCoverage(t *testing.T) {
	matrixReport := batchsourcematrix.Validate(".")
	if !matrixReport.Passed() {
		t.Fatalf("source matrix failed contract: %v", matrixReport.Messages())
	}

	records, loadReport := batchsourcematrix.LoadRecords(".")
	if !loadReport.Passed() {
		t.Fatalf("could not load source matrix: %v", loadReport.Messages())
	}
	if len(records) < 30 {
		t.Fatalf("source matrix records=%d, want at least 30 subthemes", len(records))
	}

	result, report := batchdraftgen.Generate(".", batchdraftgen.Options{SamplesPerBatch: 10, CheckedAt: "2026-06-09"})
	if !report.Passed() {
		t.Fatalf("scaled batch draft generation failed: %v", report.Messages())
	}
	if len(result.Drafts) < 60 {
		t.Fatalf("generated drafts=%d, want at least 60", len(result.Drafts))
	}
	if result.StructuralPatternRisk() > 0.35 {
		t.Fatalf("structural risk=%.2f, want <=0.35", result.StructuralPatternRisk())
	}
	if result.MaximumPairSimilarity() > 0.64 {
		t.Fatalf("max generated similarity=%.2f, want <=0.64", result.MaximumPairSimilarity())
	}

	coverage := batchsourcematrix.ValidateDraftCoverage(records, result.Drafts)
	if !coverage.Passed() {
		t.Fatalf("generated drafts lack source matrix coverage: %v", coverage.Messages())
	}
	for _, metric := range result.Metrics {
		if metric.GeneratedSamples < 10 {
			t.Fatalf("%s generated_samples=%d, want >=10", metric.BatchID, metric.GeneratedSamples)
		}
		if metric.SourceMatrixCoveredSamples != metric.GeneratedSamples {
			t.Fatalf("%s source_matrix_covered=%d generated=%d", metric.BatchID, metric.SourceMatrixCoveredSamples, metric.GeneratedSamples)
		}
		if metric.StructuralPatternRisk > 0.35 {
			t.Fatalf("%s structural risk=%.2f, want <=0.35", metric.BatchID, metric.StructuralPatternRisk)
		}
		if metric.LabEstimatedCPUUnits <= 0 {
			t.Fatalf("%s missing lab cpu estimate", metric.BatchID)
		}
	}
}

func TestBatchDraftGeneratorHandlesHundredsPerFamilyWithoutMechanicalSimilarity(t *testing.T) {
	records, loadReport := batchsourcematrix.LoadRecords(".")
	if !loadReport.Passed() {
		t.Fatalf("could not load source matrix: %v", loadReport.Messages())
	}

	result, report := batchdraftgen.Generate(".", batchdraftgen.Options{SamplesPerBatch: 100, CheckedAt: "2026-06-09"})
	if !report.Passed() {
		pair := generatedDraftSimilarityPair(result.Drafts)
		t.Fatalf("hundreds-scale batch draft generation failed: pair=%s/%s %.4f\n%s\ninvalid=%s\nissues=%v", pair.LeftID, pair.RightID, pair.Score, generatedDraftPairDiagnostic(result.Drafts, pair), generatedFirstInvalidDraftDiagnostic(result.Drafts), report.Messages())
	}
	if len(result.Drafts) < 600 {
		t.Fatalf("generated drafts=%d, want at least 600", len(result.Drafts))
	}
	if result.StructuralPatternRisk() > 0.35 {
		t.Fatalf("structural risk=%.2f, want <=0.35", result.StructuralPatternRisk())
	}
	if result.MaximumPairSimilarity() > 0.64 {
		t.Fatalf("max generated similarity=%.2f, want <=0.64", result.MaximumPairSimilarity())
	}

	coverage := batchsourcematrix.ValidateDraftCoverage(records, result.Drafts)
	if !coverage.Passed() {
		t.Fatalf("generated hundreds-scale drafts lack source matrix coverage: %v", coverage.Messages())
	}
	for _, metric := range result.Metrics {
		if metric.GeneratedSamples < 100 {
			t.Fatalf("%s generated_samples=%d, want >=100", metric.BatchID, metric.GeneratedSamples)
		}
		if metric.SourceMatrixCoveredSamples != metric.GeneratedSamples {
			t.Fatalf("%s source_matrix_covered=%d generated=%d", metric.BatchID, metric.SourceMatrixCoveredSamples, metric.GeneratedSamples)
		}
		if metric.StructuralPatternRisk > 0.35 {
			t.Fatalf("%s structural risk=%.2f, want <=0.35", metric.BatchID, metric.StructuralPatternRisk)
		}
		if metric.LabEstimatedCPUUnits < 1200 {
			t.Fatalf("%s lab cpu units=%d, want >=1200 for hundreds-scale validation", metric.BatchID, metric.LabEstimatedCPUUnits)
		}
	}
}

func TestBatchDraftGeneratorExpandsArchiveBeyondOneHundredPerFamily(t *testing.T) {
	result, report := batchdraftgen.Generate(".", batchdraftgen.Options{SamplesPerBatch: 130, CheckedAt: "2026-06-09"})
	if !report.Passed() {
		pair := generatedDraftSimilarityPair(result.Drafts)
		t.Fatalf("expanded archive batch draft generation failed: pair=%s/%s %.4f\n%s\nissues=%v", pair.LeftID, pair.RightID, pair.Score, generatedDraftPairDiagnostic(result.Drafts, pair), report.Messages())
	}
	if len(result.Drafts) < 780 {
		t.Fatalf("generated drafts=%d, want at least 780 for archive growth beyond 100 per family", len(result.Drafts))
	}
	if result.MaximumPairSimilarity() > 0.64 {
		t.Fatalf("max expanded archive similarity=%.2f, want <=0.64", result.MaximumPairSimilarity())
	}
	for _, metric := range result.Metrics {
		if metric.GeneratedSamples < 130 {
			t.Fatalf("%s generated_samples=%d, want >=130", metric.BatchID, metric.GeneratedSamples)
		}
		if metric.StructuralPatternRisk > 0.35 {
			t.Fatalf("%s structural risk=%.2f, want <=0.35", metric.BatchID, metric.StructuralPatternRisk)
		}
	}
}

func TestBatchDraftGeneratorKeepsSourceMatrixDiversityBeyondOneHundredSixtyPerFamily(t *testing.T) {
	result, report := batchdraftgen.Generate(".", batchdraftgen.Options{SamplesPerBatch: 190, CheckedAt: "2026-06-09"})
	if !report.Passed() {
		pair := generatedDraftSimilarityPair(result.Drafts)
		t.Fatalf("source-matrix diverse generation beyond 160 failed: pair=%s/%s %.4f\n%s\nissues=%v", pair.LeftID, pair.RightID, pair.Score, generatedDraftPairDiagnostic(result.Drafts, pair), report.Messages())
	}
	if len(result.Drafts) < 1140 {
		t.Fatalf("generated drafts=%d, want at least 1140 for archive growth beyond 160 per family", len(result.Drafts))
	}
	entries := make([]batchdrafts.Entry, 0, len(result.Drafts))
	for index, draft := range result.Drafts {
		entries = append(entries, batchdrafts.Entry{Line: index + 1, Record: draft})
	}
	if diversity := batchdrafts.ValidateSourceMatrixDiversity(entries); !diversity.Passed() {
		t.Fatalf("source-matrix diversity failed after generation: %v", diversity.Messages())
	}
}

func TestBatchDraftGeneratorKeepsSourceMatrixDiversityAtTwoHundredEightyPerFamily(t *testing.T) {
	result, report := batchdraftgen.Generate(".", batchdraftgen.Options{SamplesPerBatch: 280, CheckedAt: "2026-06-10"})
	if !report.Passed() {
		pair := generatedDraftSimilarityPair(result.Drafts)
		t.Fatalf("source-matrix diverse generation at 280 failed: pair=%s/%s %.4f\n%s\nissues=%v", pair.LeftID, pair.RightID, pair.Score, generatedDraftPairDiagnostic(result.Drafts, pair), report.Messages())
	}
	if len(result.Drafts) < 1680 {
		t.Fatalf("generated drafts=%d, want at least 1680 for archive growth to 280 per family", len(result.Drafts))
	}
	entries := make([]batchdrafts.Entry, 0, len(result.Drafts))
	for index, draft := range result.Drafts {
		entries = append(entries, batchdrafts.Entry{Line: index + 1, Record: draft})
	}
	if diversity := batchdrafts.ValidateSourceMatrixDiversity(entries); !diversity.Passed() {
		t.Fatalf("source-matrix diversity failed after 280 generation: %v", diversity.Messages())
	}
}

func generatedDraftSimilarityPair(drafts []batchdrafts.Record) batchdrafts.SimilarityPair {
	entries := make([]batchdrafts.Entry, 0, len(drafts))
	for index, draft := range drafts {
		entries = append(entries, batchdrafts.Entry{Line: index + 1, Record: draft})
	}
	return batchdrafts.MaximumPairSimilarityDetail(entries)
}

func generatedDraftPairDiagnostic(drafts []batchdrafts.Record, pair batchdrafts.SimilarityPair) string {
	byID := make(map[string]batchdrafts.Record)
	for _, draft := range drafts {
		byID[draft.UniqueIntentID] = draft
	}
	left := byID[pair.LeftID]
	right := byID[pair.RightID]
	return "left=" + generatedDraftDiagnostic(left) + "\nright=" + generatedDraftDiagnostic(right)
}

func generatedFirstInvalidDraftDiagnostic(drafts []batchdrafts.Record) string {
	for _, draft := range drafts {
		if report := batchdrafts.ValidateRecord(draft); !report.Passed() {
			return generatedDraftDiagnostic(draft) + " issues=" + report.Messages()[0]
		}
	}
	return "none"
}

func generatedDraftDiagnostic(draft batchdrafts.Record) string {
	return "source_matrix=" + draft.SourceMatrixID +
		" area=" + draft.LegalArea +
		" term=" + draft.Term +
		" reader=" + draft.ReaderProblem +
		" source=" + draft.SourceHook +
		" document=" + draft.DocumentContext +
		" risk=" + draft.RiskContext +
		" action=" + draft.DigitalAction
}

func TestBatchSourceMatrixRejectsWeakOrPublicSourceRecord(t *testing.T) {
	record := batchsourcematrix.Record{
		MatrixID:                "unsafe",
		BatchID:                 "batch-saude-suplementar-digital",
		LegalArea:               "saude-suplementar",
		SubthemeID:              "unsafe",
		SourceStatus:            "published",
		SourceURLs:              []string{"https://example.com/blog"},
		SourceTypes:             []string{"blog"},
		SourceSpecificityScore:  30,
		OfficialSourcesVerified: false,
		RobotsReviewRequired:    false,
		UsePolicy:               "copy_text",
		RenderAllowed:           true,
		SitemapAllowed:          true,
		PublicationAllowed:      true,
		PublicPath:              "/temas/unsafe/",
		CheckedAt:               "2026-06-09",
	}
	report := batchsourcematrix.ValidateRecord(record)
	for _, code := range []string{
		"source_matrix_invalid_status",
		"source_matrix_too_few_official_urls",
		"source_matrix_too_few_source_types",
		"source_matrix_specificity_too_low",
		"source_matrix_not_verified",
		"source_matrix_robots_review_missing",
		"source_matrix_use_policy_invalid",
		"source_matrix_render_allowed",
		"source_matrix_sitemap_allowed",
		"source_matrix_publication_allowed",
		"source_matrix_has_public_path",
	} {
		if !report.HasIssue(code) {
			t.Fatalf("missing issue %q in %v", code, report.Codes())
		}
	}
}

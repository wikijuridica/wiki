package contract_test

import (
	"testing"

	"portaljuridico/internal/batchsourceaudit"
	"portaljuridico/internal/batchsourcematrix"
)

func TestBatchSourceURLAuditCoversEveryMatrixURL(t *testing.T) {
	matrixEntries, matrixLoad := batchsourcematrix.LoadRecords(".")
	if !matrixLoad.Passed() {
		t.Fatalf("could not load source matrix: %v", matrixLoad.Messages())
	}

	auditReport := batchsourceaudit.Validate(".")
	if !auditReport.Passed() {
		t.Fatalf("batch source URL audit failed: %v", auditReport.Messages())
	}

	auditEntries, auditLoad := batchsourceaudit.LoadRecords(".")
	if !auditLoad.Passed() {
		t.Fatalf("could not load source URL audits: %v", auditLoad.Messages())
	}
	if count := batchsourceaudit.UniqueAuditedURLCount(auditEntries); count < 20 {
		t.Fatalf("audited unique URLs=%d, want at least 20 official URLs", count)
	}

	coverage := batchsourceaudit.ValidateMatrixCoverage(matrixEntries, auditEntries)
	if !coverage.Passed() {
		t.Fatalf("source URL audits do not cover matrix URLs: %v", coverage.Messages())
	}
}

func TestBatchSourceURLAuditRejectsScrapingOrPublicRecord(t *testing.T) {
	record := batchsourceaudit.Record{
		AuditID:            "unsafe-url",
		SourceURL:          "https://example.com/copied-law-blog",
		SourceURLHash:      "",
		MatrixIDs:          []string{"saude-suplementar-procedimento-urgente-negado"},
		SourceType:         "blog",
		OfficialHost:       false,
		AuditStatus:        "published",
		RobotsURL:          "https://example.com/robots.txt",
		RobotsStatus:       "pending",
		RobotsCheckedAt:    "",
		TermsURL:           "https://example.com/terms",
		TermsStatus:        "pending",
		TermsCheckedAt:     "",
		UsePolicy:          "copy_or_scrape",
		LiveCheckStatus:    "",
		ScrapingAllowed:    true,
		IngestionAllowed:   true,
		RenderAllowed:      true,
		SitemapAllowed:     true,
		PublicationAllowed: true,
		PublicPath:         "/temas/copied-law-blog/",
		CheckedAt:          "2026-06-09",
		EvidenceNote:       "",
	}
	report := batchsourceaudit.ValidateRecord(record)
	for _, code := range []string{
		"source_url_audit_invalid_status",
		"source_url_audit_unofficial_url",
		"source_url_audit_hash_missing",
		"source_url_audit_robots_pending",
		"source_url_audit_terms_pending",
		"source_url_audit_use_policy_invalid",
		"source_url_audit_scraping_allowed",
		"source_url_audit_ingestion_allowed",
		"source_url_audit_render_allowed",
		"source_url_audit_sitemap_allowed",
		"source_url_audit_publication_allowed",
		"source_url_audit_has_public_path",
		"source_url_audit_note_missing",
	} {
		if !report.HasIssue(code) {
			t.Fatalf("missing issue %q in %v", code, report.Codes())
		}
	}
}

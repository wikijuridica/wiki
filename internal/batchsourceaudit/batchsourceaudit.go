package batchsourceaudit

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"portaljuridico/internal/batchsourcematrix"
)

type Record struct {
	AuditID            string   `json:"audit_id"`
	SourceURL          string   `json:"source_url"`
	SourceURLHash      string   `json:"source_url_hash"`
	MatrixIDs          []string `json:"matrix_ids"`
	SourceType         string   `json:"source_type"`
	OfficialHost       bool     `json:"official_host"`
	AuditStatus        string   `json:"audit_status"`
	RobotsURL          string   `json:"robots_url"`
	RobotsStatus       string   `json:"robots_status"`
	RobotsCheckedAt    string   `json:"robots_checked_at"`
	TermsURL           string   `json:"terms_url"`
	TermsStatus        string   `json:"terms_status"`
	TermsCheckedAt     string   `json:"terms_checked_at"`
	UsePolicy          string   `json:"use_policy"`
	LiveCheckStatus    string   `json:"live_check_status"`
	ScrapingAllowed    bool     `json:"scraping_allowed"`
	IngestionAllowed   bool     `json:"ingestion_allowed"`
	RenderAllowed      bool     `json:"render_allowed"`
	SitemapAllowed     bool     `json:"sitemap_allowed"`
	PublicationAllowed bool     `json:"publication_allowed"`
	PublicPath         string   `json:"public_path"`
	CheckedAt          string   `json:"checked_at"`
	EvidenceNote       string   `json:"evidence_note"`
}

type Entry struct {
	Line   int
	Record Record
}

type Issue struct {
	Code    string
	Message string
}

type Report struct {
	Issues []Issue
}

func Validate(root string) Report {
	entries, report := LoadRecords(root)
	if !report.Passed() {
		return report
	}

	issues := make([]Issue, 0)
	if UniqueAuditedURLCount(entries) < 20 {
		issues = append(issues, Issue{Code: "source_url_audit_too_few_urls", Message: fmt.Sprintf("urls=%d", UniqueAuditedURLCount(entries))})
	}

	seenIDs := make(map[string]int)
	seenURLs := make(map[string]int)
	seenHashes := make(map[string]int)
	for _, entry := range entries {
		recordReport := ValidateRecord(entry.Record)
		for _, issue := range recordReport.Issues {
			issue.Message = fmt.Sprintf("line=%d %s", entry.Line, issue.Message)
			issues = append(issues, issue)
		}
		if previous := seenIDs[entry.Record.AuditID]; previous > 0 {
			issues = append(issues, Issue{Code: "source_url_audit_duplicate_id", Message: fmt.Sprintf("line=%d previous_line=%d id=%s", entry.Line, previous, entry.Record.AuditID)})
		}
		seenIDs[entry.Record.AuditID] = entry.Line
		if previous := seenURLs[entry.Record.SourceURL]; previous > 0 {
			issues = append(issues, Issue{Code: "source_url_audit_duplicate_url", Message: fmt.Sprintf("line=%d previous_line=%d url=%s", entry.Line, previous, entry.Record.SourceURL)})
		}
		seenURLs[entry.Record.SourceURL] = entry.Line
		if entry.Record.SourceURLHash != "" {
			if previous := seenHashes[entry.Record.SourceURLHash]; previous > 0 {
				issues = append(issues, Issue{Code: "source_url_audit_duplicate_hash", Message: fmt.Sprintf("line=%d previous_line=%d hash=%s", entry.Line, previous, entry.Record.SourceURLHash)})
			}
			seenHashes[entry.Record.SourceURLHash] = entry.Line
		}
	}

	matrixEntries, matrixReport := batchsourcematrix.LoadRecords(root)
	if matrixReport.Passed() {
		coverage := ValidateMatrixCoverage(matrixEntries, entries)
		issues = append(issues, coverage.Issues...)
	}
	return Report{Issues: issues}
}

func LoadRecords(root string) ([]Entry, Report) {
	projectRoot, err := findProjectRoot(root)
	if err != nil {
		return nil, Report{Issues: []Issue{{Code: "project_root_not_found", Message: err.Error()}}}
	}
	path := filepath.Join(projectRoot, "data", "source-audit", "batch_source_urls.jsonl")
	file, err := os.Open(path)
	if err != nil {
		return nil, Report{Issues: []Issue{{Code: "source_url_audit_missing", Message: err.Error()}}}
	}
	defer file.Close()

	entries := make([]Entry, 0)
	issues := make([]Issue, 0)
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 0, 4096), 65536)
	lineNumber := 0
	for scanner.Scan() {
		lineNumber++
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var record Record
		if err := json.Unmarshal([]byte(line), &record); err != nil {
			issues = append(issues, Issue{Code: "source_url_audit_invalid_json", Message: fmt.Sprintf("line=%d %s", lineNumber, err.Error())})
			continue
		}
		entries = append(entries, Entry{Line: lineNumber, Record: record})
	}
	if err := scanner.Err(); err != nil {
		issues = append(issues, Issue{Code: "source_url_audit_scan_failed", Message: err.Error()})
	}
	return entries, Report{Issues: issues}
}

func ValidateRecord(record Record) Report {
	issues := make([]Issue, 0)
	if record.AuditID == "" || record.SourceURL == "" || record.SourceType == "" {
		issues = append(issues, Issue{Code: "source_url_audit_missing_identity", Message: record.AuditID})
	}
	if len(record.MatrixIDs) == 0 {
		issues = append(issues, Issue{Code: "source_url_audit_missing_matrix_ids", Message: record.AuditID})
	}
	if record.AuditStatus != "source_url_audited_reference_only_blocked" {
		issues = append(issues, Issue{Code: "source_url_audit_invalid_status", Message: record.AuditStatus})
	}
	if !record.OfficialHost || !isOfficialURL(record.SourceURL) {
		issues = append(issues, Issue{Code: "source_url_audit_unofficial_url", Message: record.SourceURL})
	}
	if record.SourceURLHash == "" {
		issues = append(issues, Issue{Code: "source_url_audit_hash_missing", Message: record.AuditID})
	} else if record.SourceURLHash != URLHash(record.SourceURL) {
		issues = append(issues, Issue{Code: "source_url_audit_hash_mismatch", Message: record.AuditID})
	}
	if record.RobotsURL == "" || record.RobotsStatus == "" || record.RobotsCheckedAt == "" {
		issues = append(issues, Issue{Code: "source_url_audit_robots_missing", Message: record.AuditID})
	}
	if isPending(record.RobotsStatus) {
		issues = append(issues, Issue{Code: "source_url_audit_robots_pending", Message: record.AuditID})
	}
	if record.TermsURL == "" || record.TermsStatus == "" || record.TermsCheckedAt == "" {
		issues = append(issues, Issue{Code: "source_url_audit_terms_missing", Message: record.AuditID})
	}
	if isPending(record.TermsStatus) {
		issues = append(issues, Issue{Code: "source_url_audit_terms_pending", Message: record.AuditID})
	}
	if record.UsePolicy != "reference_only_no_scraping_no_ingestion" {
		issues = append(issues, Issue{Code: "source_url_audit_use_policy_invalid", Message: record.UsePolicy})
	}
	if record.LiveCheckStatus == "" {
		issues = append(issues, Issue{Code: "source_url_audit_live_check_missing", Message: record.AuditID})
	}
	if record.ScrapingAllowed {
		issues = append(issues, Issue{Code: "source_url_audit_scraping_allowed", Message: record.AuditID})
	}
	if record.IngestionAllowed {
		issues = append(issues, Issue{Code: "source_url_audit_ingestion_allowed", Message: record.AuditID})
	}
	if record.RenderAllowed {
		issues = append(issues, Issue{Code: "source_url_audit_render_allowed", Message: record.AuditID})
	}
	if record.SitemapAllowed {
		issues = append(issues, Issue{Code: "source_url_audit_sitemap_allowed", Message: record.AuditID})
	}
	if record.PublicationAllowed {
		issues = append(issues, Issue{Code: "source_url_audit_publication_allowed", Message: record.AuditID})
	}
	if record.PublicPath != "" {
		issues = append(issues, Issue{Code: "source_url_audit_has_public_path", Message: record.PublicPath})
	}
	if record.CheckedAt == "" {
		issues = append(issues, Issue{Code: "source_url_audit_without_checked_at", Message: record.AuditID})
	}
	if strings.TrimSpace(record.EvidenceNote) == "" {
		issues = append(issues, Issue{Code: "source_url_audit_note_missing", Message: record.AuditID})
	}
	return Report{Issues: issues}
}

func ValidateMatrixCoverage(matrixEntries []batchsourcematrix.Entry, auditEntries []Entry) Report {
	auditsByURL := make(map[string]Record)
	for _, entry := range auditEntries {
		auditsByURL[entry.Record.SourceURL] = entry.Record
	}
	issues := make([]Issue, 0)
	for _, matrixEntry := range matrixEntries {
		for _, url := range matrixEntry.Record.SourceURLs {
			audit, ok := auditsByURL[url]
			if !ok {
				issues = append(issues, Issue{Code: "source_url_audit_matrix_url_missing", Message: matrixEntry.Record.MatrixID + ":" + url})
				continue
			}
			if !contains(audit.MatrixIDs, matrixEntry.Record.MatrixID) {
				issues = append(issues, Issue{Code: "source_url_audit_matrix_link_missing", Message: audit.AuditID + ":" + matrixEntry.Record.MatrixID})
			}
		}
	}
	return Report{Issues: issues}
}

func UniqueAuditedURLCount(entries []Entry) int {
	seen := make(map[string]bool)
	for _, entry := range entries {
		if entry.Record.SourceURL != "" {
			seen[entry.Record.SourceURL] = true
		}
	}
	return len(seen)
}

func URLHash(url string) string {
	sum := sha256.Sum256([]byte(url))
	return "urlsha256:" + hex.EncodeToString(sum[:])
}

func isPending(value string) bool {
	value = strings.ToLower(strings.TrimSpace(value))
	return value == "" || strings.Contains(value, "pending") || strings.Contains(value, "pendente")
}

func isOfficialURL(url string) bool {
	prefixes := []string{
		"https://www.gov.br/",
		"https://www.planalto.gov.br/",
		"https://www.cnj.jus.br/",
		"https://atos.cnj.jus.br/",
		"https://www.bcb.gov.br/",
		"https://www.tst.jus.br/",
	}
	for _, prefix := range prefixes {
		if strings.HasPrefix(url, prefix) {
			return true
		}
	}
	return false
}

func contains(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func (r Report) Passed() bool { return len(r.Issues) == 0 }

func (r Report) HasIssue(code string) bool {
	for _, issue := range r.Issues {
		if issue.Code == code {
			return true
		}
	}
	return false
}

func (r Report) Codes() []string {
	codes := make([]string, 0, len(r.Issues))
	for _, issue := range r.Issues {
		codes = append(codes, issue.Code)
	}
	sort.Strings(codes)
	return codes
}

func (r Report) Messages() []string {
	messages := make([]string, 0, len(r.Issues))
	for _, issue := range r.Issues {
		messages = append(messages, issue.Code+": "+issue.Message)
	}
	return messages
}

func findProjectRoot(start string) (string, error) {
	current, err := filepath.Abs(start)
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Stat(filepath.Join(current, "go.mod")); err == nil {
			return current, nil
		}
		parent := filepath.Dir(current)
		if parent == current {
			return "", os.ErrNotExist
		}
		current = parent
	}
}

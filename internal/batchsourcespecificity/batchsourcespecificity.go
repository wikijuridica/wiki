package batchsourcespecificity

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"portaljuridico/internal/batchprepublication"
	"portaljuridico/internal/batchsourceaudit"
	"portaljuridico/internal/content"
	"portaljuridico/internal/router"
	"portaljuridico/internal/seo"
)

const (
	LockedStatus  = "final_source_locked_reference_only"
	BlockedStatus = "final_source_blocked_needs_specific_url"
	UsePolicy     = "reference_only_no_scraping_no_ingestion"
)

type Record struct {
	ResolutionID            string   `json:"resolution_id"`
	PrepublicationID        string   `json:"prepublication_id"`
	ReviewID                string   `json:"review_id"`
	BatchID                 string   `json:"batch_id"`
	UniqueIntentID          string   `json:"unique_intent_id"`
	SourceMatrixID          string   `json:"source_matrix_id"`
	Term                    string   `json:"term"`
	CandidatePath           string   `json:"candidate_path"`
	CandidateCanonicalURL   string   `json:"candidate_canonical_url"`
	CandidateRobots         string   `json:"candidate_robots"`
	SourceSpecificityStatus string   `json:"source_specificity_status"`
	SelectedSourceURLs      []string `json:"selected_source_urls"`
	BlockingReason          string   `json:"blocking_reason"`
	NeededSourceDetail      string   `json:"needed_source_detail"`
	UsePolicy               string   `json:"use_policy"`
	ScrapingAllowed         bool     `json:"scraping_allowed"`
	IngestionAllowed        bool     `json:"ingestion_allowed"`
	RenderAllowed           bool     `json:"render_allowed"`
	SitemapAllowed          bool     `json:"sitemap_allowed"`
	PublicationAllowed      bool     `json:"publication_allowed"`
	PublicPath              string   `json:"public_path"`
	CheckedAt               string   `json:"checked_at"`
}

type Entry struct {
	Line   int
	Record Record
}

type PrepublicationCandidate struct {
	PrepublicationID      string
	ReviewID              string
	BatchID               string
	UniqueIntentID        string
	SourceMatrixID        string
	Term                  string
	CandidatePath         string
	CandidateCanonicalURL string
	CandidateRobots       string
}

type AuditedSource struct {
	SourceURL          string
	MatrixIDs          []string
	OfficialHost       bool
	AuditStatus        string
	UsePolicy          string
	SourceType         string
	ScrapingAllowed    bool
	IngestionAllowed   bool
	RenderAllowed      bool
	SitemapAllowed     bool
	PublicationAllowed bool
	PublicPath         string
}

type SourceIndex struct {
	BaseURL                string
	PrepublicationByIntent map[string]PrepublicationCandidate
	AuditsByURL            map[string]AuditedSource
}

type Issue struct {
	Code    string
	Message string
}

type Report struct {
	Issues []Issue
}

var idPattern = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)

func Validate(root string) Report {
	entries, report := LoadRecords(root)
	if !report.Passed() {
		return report
	}
	index, indexReport := BuildSourceIndex(root)
	issues := append([]Issue{}, indexReport.Issues...)
	if len(entries) != len(index.PrepublicationByIntent) {
		issues = append(issues, Issue{Code: "batch_source_specificity_count_mismatch", Message: fmt.Sprintf("resolutions=%d prepublication=%d", len(entries), len(index.PrepublicationByIntent))})
	}

	seenIntent := make(map[string]int)
	seenResolution := make(map[string]int)
	for _, entry := range entries {
		recordReport := ValidateRecordAgainstSourceIndex(entry.Record, index)
		for _, issue := range recordReport.Issues {
			issue.Message = fmt.Sprintf("line=%d %s", entry.Line, issue.Message)
			issues = append(issues, issue)
		}
		if previous := seenIntent[entry.Record.UniqueIntentID]; previous > 0 {
			issues = append(issues, Issue{Code: "batch_source_specificity_duplicate_intent", Message: fmt.Sprintf("line=%d previous_line=%d id=%s", entry.Line, previous, entry.Record.UniqueIntentID)})
		}
		seenIntent[entry.Record.UniqueIntentID] = entry.Line
		if previous := seenResolution[entry.Record.ResolutionID]; previous > 0 {
			issues = append(issues, Issue{Code: "batch_source_specificity_duplicate_resolution", Message: fmt.Sprintf("line=%d previous_line=%d id=%s", entry.Line, previous, entry.Record.ResolutionID)})
		}
		seenResolution[entry.Record.ResolutionID] = entry.Line
	}
	for intentID := range index.PrepublicationByIntent {
		if seenIntent[intentID] == 0 {
			issues = append(issues, Issue{Code: "batch_source_specificity_missing_prepublication", Message: intentID})
		}
	}
	return Report{Issues: issues}
}

func BuildSourceIndex(root string) (SourceIndex, Report) {
	issues := convertPrepublicationIssues(batchprepublication.Validate(root))
	repo, err := content.LoadRepository(root)
	if err != nil {
		issues = append(issues, Issue{Code: "batch_source_specificity_site_config_unavailable", Message: err.Error()})
	}
	baseURL := "https://wikijuridica.com.br"
	if repo.BaseURL != "" {
		baseURL = repo.BaseURL
	}
	baseURL = strings.TrimRight(baseURL, "/")

	preEntries, preLoadReport := batchprepublication.LoadRecords(root)
	issues = append(issues, convertPrepublicationIssues(preLoadReport)...)
	auditEntries, auditLoadReport := batchsourceaudit.LoadRecords(root)
	issues = append(issues, convertSourceAuditIssues(auditLoadReport)...)
	issues = append(issues, convertSourceAuditIssues(batchsourceaudit.Validate(root))...)

	index := SourceIndex{
		BaseURL:                baseURL,
		PrepublicationByIntent: make(map[string]PrepublicationCandidate),
		AuditsByURL:            make(map[string]AuditedSource),
	}
	for _, entry := range preEntries {
		record := entry.Record
		index.PrepublicationByIntent[record.UniqueIntentID] = PrepublicationCandidate{
			PrepublicationID:      record.PrepublicationID,
			ReviewID:              record.ReviewID,
			BatchID:               record.BatchID,
			UniqueIntentID:        record.UniqueIntentID,
			SourceMatrixID:        record.SourceMatrixID,
			Term:                  record.Term,
			CandidatePath:         record.CandidatePath,
			CandidateCanonicalURL: record.CandidateCanonicalURL,
			CandidateRobots:       record.CandidateRobots,
		}
	}
	for _, entry := range auditEntries {
		record := entry.Record
		index.AuditsByURL[record.SourceURL] = AuditedSource{
			SourceURL:          record.SourceURL,
			MatrixIDs:          append([]string{}, record.MatrixIDs...),
			OfficialHost:       record.OfficialHost,
			AuditStatus:        record.AuditStatus,
			UsePolicy:          record.UsePolicy,
			SourceType:         record.SourceType,
			ScrapingAllowed:    record.ScrapingAllowed,
			IngestionAllowed:   record.IngestionAllowed,
			RenderAllowed:      record.RenderAllowed,
			SitemapAllowed:     record.SitemapAllowed,
			PublicationAllowed: record.PublicationAllowed,
			PublicPath:         record.PublicPath,
		}
	}
	return index, Report{Issues: issues}
}

func BuildSpecificSourceURLsByMatrix(root string) (map[string][]string, Report) {
	auditEntries, auditLoadReport := batchsourceaudit.LoadRecords(root)
	issues := convertSourceAuditIssues(auditLoadReport)
	issues = append(issues, convertSourceAuditIssues(batchsourceaudit.Validate(root))...)

	urlsByMatrix := make(map[string][]string)
	seenByMatrix := make(map[string]map[string]bool)
	for _, entry := range auditEntries {
		record := entry.Record
		source := AuditedSource{
			SourceURL:          record.SourceURL,
			MatrixIDs:          append([]string{}, record.MatrixIDs...),
			OfficialHost:       record.OfficialHost,
			AuditStatus:        record.AuditStatus,
			UsePolicy:          record.UsePolicy,
			SourceType:         record.SourceType,
			ScrapingAllowed:    record.ScrapingAllowed,
			IngestionAllowed:   record.IngestionAllowed,
			RenderAllowed:      record.RenderAllowed,
			SitemapAllowed:     record.SitemapAllowed,
			PublicationAllowed: record.PublicationAllowed,
			PublicPath:         record.PublicPath,
		}
		if !IsSpecificAuditedSource(source) {
			continue
		}
		for _, matrixID := range record.MatrixIDs {
			matrixID = strings.TrimSpace(matrixID)
			if matrixID == "" {
				continue
			}
			if seenByMatrix[matrixID] == nil {
				seenByMatrix[matrixID] = make(map[string]bool)
			}
			if seenByMatrix[matrixID][record.SourceURL] {
				continue
			}
			seenByMatrix[matrixID][record.SourceURL] = true
			urlsByMatrix[matrixID] = append(urlsByMatrix[matrixID], record.SourceURL)
		}
	}
	for matrixID := range urlsByMatrix {
		sort.Strings(urlsByMatrix[matrixID])
	}
	return urlsByMatrix, Report{Issues: issues}
}

func IsSpecificAuditedSource(source AuditedSource) bool {
	if !source.OfficialHost || !isOfficialURL(source.SourceURL) {
		return false
	}
	if source.AuditStatus != "source_url_audited_reference_only_blocked" || source.UsePolicy != UsePolicy {
		return false
	}
	if source.ScrapingAllowed || source.IngestionAllowed || source.RenderAllowed || source.SitemapAllowed || source.PublicationAllowed || source.PublicPath != "" {
		return false
	}
	return !isBroadSource(source)
}

func ValidateRecordAgainstSourceIndex(record Record, index SourceIndex) Report {
	issues := make([]Issue, 0)
	for _, id := range []struct {
		code  string
		value string
	}{
		{"batch_source_specificity_invalid_id", record.ResolutionID},
		{"batch_source_specificity_invalid_prepublication_id", record.PrepublicationID},
		{"batch_source_specificity_invalid_review_id", record.ReviewID},
		{"batch_source_specificity_invalid_batch_id", record.BatchID},
		{"batch_source_specificity_invalid_intent", record.UniqueIntentID},
		{"batch_source_specificity_invalid_source_matrix", record.SourceMatrixID},
	} {
		if id.value == "" || !idPattern.MatchString(id.value) {
			issues = append(issues, Issue{Code: id.code, Message: id.value})
		}
	}

	prepublication, ok := index.PrepublicationByIntent[record.UniqueIntentID]
	if !ok {
		issues = append(issues, Issue{Code: "batch_source_specificity_missing_prepublication", Message: record.UniqueIntentID})
	} else {
		if record.PrepublicationID != prepublication.PrepublicationID {
			issues = append(issues, Issue{Code: "batch_source_specificity_prepublication_mismatch", Message: record.UniqueIntentID})
		}
		if record.ReviewID != prepublication.ReviewID {
			issues = append(issues, Issue{Code: "batch_source_specificity_review_mismatch", Message: record.UniqueIntentID})
		}
		if record.BatchID != prepublication.BatchID {
			issues = append(issues, Issue{Code: "batch_source_specificity_batch_mismatch", Message: record.UniqueIntentID})
		}
		if record.SourceMatrixID != prepublication.SourceMatrixID {
			issues = append(issues, Issue{Code: "batch_source_specificity_source_matrix_mismatch", Message: fmt.Sprintf("record=%s prepublication=%s", record.SourceMatrixID, prepublication.SourceMatrixID)})
		}
		if record.Term != prepublication.Term {
			issues = append(issues, Issue{Code: "batch_source_specificity_term_mismatch", Message: record.UniqueIntentID})
		}
		if record.CandidatePath != prepublication.CandidatePath {
			issues = append(issues, Issue{Code: "batch_source_specificity_candidate_path_mismatch", Message: record.UniqueIntentID})
		}
		if record.CandidateCanonicalURL != prepublication.CandidateCanonicalURL {
			issues = append(issues, Issue{Code: "batch_source_specificity_canonical_prepublication_mismatch", Message: record.UniqueIntentID})
		}
		if record.CandidateRobots != prepublication.CandidateRobots {
			issues = append(issues, Issue{Code: "batch_source_specificity_robots_prepublication_mismatch", Message: record.UniqueIntentID})
		}
	}

	if strings.Contains(record.CandidatePath, "://") || !router.IsCleanPublicPath(record.CandidatePath) || strings.ContainsAny(record.CandidatePath, "?#") {
		issues = append(issues, Issue{Code: "batch_source_specificity_candidate_path_not_clean", Message: record.CandidatePath})
	}
	expectedCanonical := strings.TrimRight(index.BaseURL, "/") + record.CandidatePath
	if !seo.IsAbsoluteHTTPSURL(record.CandidateCanonicalURL) {
		issues = append(issues, Issue{Code: "batch_source_specificity_canonical_not_https", Message: record.CandidateCanonicalURL})
	} else if record.CandidateCanonicalURL != expectedCanonical {
		issues = append(issues, Issue{Code: "batch_source_specificity_canonical_mismatch", Message: record.CandidateCanonicalURL})
	}
	if record.CandidateRobots != "noindex,follow" {
		issues = append(issues, Issue{Code: "batch_source_specificity_not_noindex", Message: record.CandidateRobots})
	}
	if record.SourceSpecificityStatus != LockedStatus && record.SourceSpecificityStatus != BlockedStatus {
		issues = append(issues, Issue{Code: "batch_source_specificity_invalid_status", Message: record.SourceSpecificityStatus})
	}
	if len(record.SelectedSourceURLs) == 0 {
		issues = append(issues, Issue{Code: "batch_source_specificity_missing_selected_source_urls", Message: record.UniqueIntentID})
	}

	specificURLs := 0
	for _, selectedURL := range record.SelectedSourceURLs {
		selectedURL = strings.TrimSpace(selectedURL)
		audit, ok := index.AuditsByURL[selectedURL]
		if !ok {
			issues = append(issues, Issue{Code: "batch_source_specificity_source_url_not_audited", Message: selectedURL})
			continue
		}
		if !contains(audit.MatrixIDs, record.SourceMatrixID) {
			issues = append(issues, Issue{Code: "batch_source_specificity_source_url_not_linked", Message: record.SourceMatrixID + ":" + selectedURL})
		}
		if !audit.OfficialHost || !isOfficialURL(selectedURL) {
			issues = append(issues, Issue{Code: "batch_source_specificity_unofficial_source_url", Message: selectedURL})
		}
		if audit.AuditStatus != "source_url_audited_reference_only_blocked" {
			issues = append(issues, Issue{Code: "batch_source_specificity_source_audit_invalid", Message: selectedURL + ":" + audit.AuditStatus})
		}
		if audit.UsePolicy != UsePolicy {
			issues = append(issues, Issue{Code: "batch_source_specificity_source_use_policy_invalid", Message: selectedURL + ":" + audit.UsePolicy})
		}
		if audit.ScrapingAllowed || audit.IngestionAllowed || audit.RenderAllowed || audit.SitemapAllowed || audit.PublicationAllowed || audit.PublicPath != "" {
			issues = append(issues, Issue{Code: "batch_source_specificity_source_public_flags", Message: selectedURL})
		}
		if !isBroadSource(audit) {
			specificURLs++
		}
	}
	if record.SourceSpecificityStatus == LockedStatus {
		if specificURLs == 0 {
			issues = append(issues, Issue{Code: "batch_source_specificity_locked_without_specific_url", Message: record.UniqueIntentID})
		}
		if strings.TrimSpace(record.BlockingReason) != "" || strings.TrimSpace(record.NeededSourceDetail) != "" {
			issues = append(issues, Issue{Code: "batch_source_specificity_locked_has_blocking_fields", Message: record.UniqueIntentID})
		}
	}
	if record.SourceSpecificityStatus == BlockedStatus {
		if strings.TrimSpace(record.BlockingReason) == "" {
			issues = append(issues, Issue{Code: "batch_source_specificity_missing_blocking_reason", Message: record.UniqueIntentID})
		}
		if strings.TrimSpace(record.NeededSourceDetail) == "" {
			issues = append(issues, Issue{Code: "batch_source_specificity_missing_needed_source_detail", Message: record.UniqueIntentID})
		}
	}
	if record.UsePolicy != UsePolicy {
		issues = append(issues, Issue{Code: "batch_source_specificity_invalid_use_policy", Message: record.UsePolicy})
	}
	if record.ScrapingAllowed {
		issues = append(issues, Issue{Code: "batch_source_specificity_scraping_allowed", Message: record.UniqueIntentID})
	}
	if record.IngestionAllowed {
		issues = append(issues, Issue{Code: "batch_source_specificity_ingestion_allowed", Message: record.UniqueIntentID})
	}
	if record.RenderAllowed {
		issues = append(issues, Issue{Code: "batch_source_specificity_render_allowed", Message: record.UniqueIntentID})
	}
	if record.SitemapAllowed {
		issues = append(issues, Issue{Code: "batch_source_specificity_sitemap_allowed", Message: record.UniqueIntentID})
	}
	if record.PublicationAllowed {
		issues = append(issues, Issue{Code: "batch_source_specificity_publication_allowed", Message: record.UniqueIntentID})
	}
	if record.PublicPath != "" {
		issues = append(issues, Issue{Code: "batch_source_specificity_has_public_path", Message: record.PublicPath})
	}
	if record.CheckedAt == "" {
		issues = append(issues, Issue{Code: "batch_source_specificity_without_checked_at", Message: record.UniqueIntentID})
	}
	return Report{Issues: issues}
}

func LoadRecords(root string) ([]Entry, Report) {
	projectRoot, err := findProjectRoot(root)
	if err != nil {
		return nil, Report{Issues: []Issue{{Code: "project_root_not_found", Message: err.Error()}}}
	}
	path := filepath.Join(projectRoot, "data", "editorial", "batch_source_specificity_resolutions.jsonl")
	file, err := os.Open(path)
	if err != nil {
		return nil, Report{Issues: []Issue{{Code: "batch_source_specificity_missing", Message: err.Error()}}}
	}
	defer file.Close()

	issues := make([]Issue, 0)
	entries := make([]Entry, 0)
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
			issues = append(issues, Issue{Code: "batch_source_specificity_invalid_json", Message: fmt.Sprintf("line=%d %s", lineNumber, err.Error())})
			continue
		}
		entries = append(entries, Entry{Line: lineNumber, Record: record})
	}
	if err := scanner.Err(); err != nil {
		issues = append(issues, Issue{Code: "batch_source_specificity_scan_failed", Message: err.Error()})
	}
	return entries, Report{Issues: issues}
}

func isBroadSource(source AuditedSource) bool {
	if strings.HasSuffix(source.SourceURL, ".gov.br/") || strings.HasSuffix(source.SourceURL, ".jus.br/") {
		return true
	}
	if source.SourceURL == "https://www.gov.br/consumidor/pt-br" || source.SourceURL == "https://www.gov.br/inss/pt-br" || source.SourceURL == "https://www.gov.br/trabalho-e-emprego/pt-br" {
		return true
	}
	broadTypes := []string{"codigo_civil", "codigo_consumidor", "clt_compilada", "orgao_judiciario", "regulador_financeiro", "orgao_previdenciario", "tribunal_trabalhista", "orientacao_administrativa"}
	for _, sourceType := range broadTypes {
		if source.SourceType == sourceType {
			return true
		}
	}
	return false
}

func isOfficialURL(url string) bool {
	prefixes := []string{
		"https://www.gov.br/",
		"https://www.planalto.gov.br/",
		"https://www.cnj.jus.br/",
		"https://atos.cnj.jus.br/",
		"https://www.bcb.gov.br/",
		"https://www.tst.jus.br/",
		"https://www.stj.jus.br/",
		"https://www.stf.jus.br/",
	}
	for _, prefix := range prefixes {
		if strings.HasPrefix(url, prefix) {
			return true
		}
	}
	return false
}

func contains(values []string, needle string) bool {
	for _, value := range values {
		if value == needle {
			return true
		}
	}
	return false
}

func convertPrepublicationIssues(report batchprepublication.Report) []Issue {
	issues := make([]Issue, 0, len(report.Issues))
	for _, issue := range report.Issues {
		issues = append(issues, Issue{Code: "batch_source_specificity_prepublication_" + issue.Code, Message: issue.Message})
	}
	return issues
}

func convertSourceAuditIssues(report batchsourceaudit.Report) []Issue {
	issues := make([]Issue, 0, len(report.Issues))
	for _, issue := range report.Issues {
		issues = append(issues, Issue{Code: "batch_source_specificity_source_audit_" + issue.Code, Message: issue.Message})
	}
	return issues
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

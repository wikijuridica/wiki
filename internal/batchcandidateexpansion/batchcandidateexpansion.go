package batchcandidateexpansion

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"portaljuridico/internal/batchcandidategates"
	"portaljuridico/internal/batchdraftarchive"
	"portaljuridico/internal/batchdrafts"
	"portaljuridico/internal/batchsourcespecificity"
	"portaljuridico/internal/paidintent"
)

const (
	ReadyBlockedStatus    = "batch_candidate_expansion_ready_blocked_publication"
	PaidGateMissingStatus = "batch_candidate_expansion_blocked_paid_gate_missing"
	PaidGateBlockedStatus = "batch_candidate_expansion_blocked_paid_gate_failed"
	ArchivePath           = "data/editorial/batch_draft_expansion_archive.jsonl"
	PaidIntentGatePath    = "data/editorial/batch_paid_intent_gates.jsonl"

	ExpansionCandidateSelectorArchivePrefix = "archive_prefix_by_batch"
	MaxExpansionCandidateSampleIntentIDs    = 12
)

type Record struct {
	ReadinessID                       string   `json:"readiness_id"`
	BatchID                           string   `json:"batch_id"`
	LegalArea                         string   `json:"legal_area"`
	ReadinessStatus                   string   `json:"readiness_status"`
	TargetCandidateTier               string   `json:"target_candidate_tier"`
	SourceArchivePath                 string   `json:"source_archive_path"`
	ArchiveRecordsRequired            int      `json:"archive_records_required"`
	ArchiveRecordsObserved            int      `json:"archive_records_observed"`
	CurrentCandidateCount             int      `json:"current_candidate_count"`
	TargetCandidateCount              int      `json:"target_candidate_count"`
	ExpansionCandidateSelector        string   `json:"expansion_candidate_selector,omitempty"`
	ExpansionCandidateSampleIntentIDs []string `json:"expansion_candidate_sample_intent_ids,omitempty"`
	ExpansionCandidateIntentIDs       []string `json:"expansion_candidate_intent_ids,omitempty"`
	KnownSourceBlockerIntentIDs       []string `json:"known_source_blocker_intent_ids"`
	PaidIntentGatePath                string   `json:"paid_intent_gate_path"`
	PaidIntentPassedCount             int      `json:"paid_intent_passed_count"`
	PaidIntentBlockedCount            int      `json:"paid_intent_blocked_count"`
	PaidIntentMissingCount            int      `json:"paid_intent_missing_count"`
	PaidIntentMissingIntentIDs        []string `json:"paid_intent_missing_intent_ids"`
	PaidIntentBlockedIntentIDs        []string `json:"paid_intent_blocked_intent_ids"`
	ActionableBlockers                []string `json:"actionable_blockers"`
	MinimumHumanScore                 int      `json:"minimum_human_score"`
	MaxSimilarityAllowed              float64  `json:"max_similarity_allowed"`
	MaxSimilarityObserved             float64  `json:"max_similarity_observed"`
	CTAContextRequired                bool     `json:"cta_context_required"`
	SourceURLAuditRequired            bool     `json:"source_url_audit_required"`
	SourceSpecificityRequired         bool     `json:"source_specificity_required"`
	PaidIntentRequired                bool     `json:"paid_intent_required"`
	NextGate                          string   `json:"next_gate"`
	PublicationBlockReason            string   `json:"publication_block_reason"`
	IndexPolicy                       string   `json:"index_policy"`
	ManifestAllowed                   bool     `json:"manifest_allowed"`
	RenderAllowed                     bool     `json:"render_allowed"`
	SitemapAllowed                    bool     `json:"sitemap_allowed"`
	PublicationAllowed                bool     `json:"publication_allowed"`
	PublicPath                        string   `json:"public_path"`
	CheckedAt                         string   `json:"checked_at"`
}

type Entry struct {
	Line   int
	Record Record
}

type ArchiveDraft struct {
	BatchID            string
	UniqueIntentID     string
	HumanScore         int
	SourceMatrixID     string
	CTAContext         string
	RenderAllowed      bool
	SitemapAllowed     bool
	PublicationAllowed bool
	PublicPath         string
}

type ExpansionIndex struct {
	ArchiveByIntent              map[string]ArchiveDraft
	ArchiveIntentIDsByBatch      map[string][]string
	PaidIntentByIntent           map[string]paidintent.Record
	ArchiveCountByBatch          map[string]int
	CurrentCandidateCountByBatch map[string]int
	SourceBlockersByBatch        map[string][]string
	MaxSimilarity                float64
	MaxSimilarityPair            batchdrafts.SimilarityPair
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
	index, indexReport := BuildExpansionIndex(root)
	issues := append([]Issue{}, indexReport.Issues...)
	if len(entries) != 6 {
		issues = append(issues, Issue{Code: "batch_candidate_expansion_record_count_invalid", Message: fmt.Sprintf("records=%d", len(entries))})
	}
	seen := make(map[string]int)
	for _, entry := range entries {
		recordReport := ValidateRecordAgainstIndex(entry.Record, index)
		for _, issue := range recordReport.Issues {
			issue.Message = fmt.Sprintf("line=%d %s", entry.Line, issue.Message)
			issues = append(issues, issue)
		}
		if previous := seen[entry.Record.BatchID]; previous > 0 {
			issues = append(issues, Issue{Code: "batch_candidate_expansion_duplicate_batch", Message: fmt.Sprintf("line=%d previous_line=%d batch=%s", entry.Line, previous, entry.Record.BatchID)})
		}
		seen[entry.Record.BatchID] = entry.Line
	}
	return Report{Issues: issues}
}

func BuildExpansionIndex(root string) (ExpansionIndex, Report) {
	archiveEntries, archiveReport := batchdraftarchive.LoadRecords(root)
	candidateEntries, candidateReport := batchcandidategates.LoadRecords(root)
	sourceEntries, sourceReport := batchsourcespecificity.LoadRecords(root)
	paidEntries, paidReport := paidintent.LoadRecords(root)
	issues := append(convertArchiveIssues(archiveReport), convertCandidateIssues(candidateReport)...)
	issues = append(issues, convertSourceSpecificityIssues(sourceReport)...)
	issues = append(issues, convertPaidIntentIssues(paidReport)...)

	index := ExpansionIndex{
		ArchiveByIntent:              make(map[string]ArchiveDraft),
		ArchiveIntentIDsByBatch:      make(map[string][]string),
		PaidIntentByIntent:           make(map[string]paidintent.Record),
		ArchiveCountByBatch:          make(map[string]int),
		CurrentCandidateCountByBatch: make(map[string]int),
		SourceBlockersByBatch:        make(map[string][]string),
		MaxSimilarityPair:            batchdrafts.MaximumPairSimilarityDetail(archiveEntries),
	}
	index.MaxSimilarity = index.MaxSimilarityPair.Score
	for _, entry := range archiveEntries {
		record := entry.Record
		index.ArchiveByIntent[record.UniqueIntentID] = ArchiveDraft{
			BatchID:            record.BatchID,
			UniqueIntentID:     record.UniqueIntentID,
			HumanScore:         record.HumanScore,
			SourceMatrixID:     record.SourceMatrixID,
			CTAContext:         record.CTAContext,
			RenderAllowed:      record.RenderAllowed,
			SitemapAllowed:     record.SitemapAllowed,
			PublicationAllowed: record.PublicationAllowed,
			PublicPath:         record.PublicPath,
		}
		index.ArchiveCountByBatch[record.BatchID]++
		index.ArchiveIntentIDsByBatch[record.BatchID] = append(index.ArchiveIntentIDsByBatch[record.BatchID], record.UniqueIntentID)
	}
	for _, entry := range paidEntries {
		index.PaidIntentByIntent[entry.Record.UniqueIntentID] = entry.Record
	}
	for _, entry := range candidateEntries {
		index.CurrentCandidateCountByBatch[entry.Record.BatchID] += len(entry.Record.SelectedUniqueIntentIDs)
	}
	for _, entry := range sourceEntries {
		if entry.Record.SourceSpecificityStatus == batchsourcespecificity.BlockedStatus {
			index.SourceBlockersByBatch[entry.Record.BatchID] = append(index.SourceBlockersByBatch[entry.Record.BatchID], entry.Record.UniqueIntentID)
		}
	}
	for batchID := range index.SourceBlockersByBatch {
		sort.Strings(index.SourceBlockersByBatch[batchID])
	}
	return index, Report{Issues: issues}
}

func ValidateRecordAgainstIndex(record Record, index ExpansionIndex) Report {
	issues := make([]Issue, 0)
	if record.ReadinessID == "" || !idPattern.MatchString(record.ReadinessID) {
		issues = append(issues, Issue{Code: "batch_candidate_expansion_invalid_readiness_id", Message: record.ReadinessID})
	}
	if record.BatchID == "" || !idPattern.MatchString(record.BatchID) {
		issues = append(issues, Issue{Code: "batch_candidate_expansion_invalid_batch_id", Message: record.BatchID})
	}
	if strings.TrimSpace(record.LegalArea) == "" {
		issues = append(issues, Issue{Code: "batch_candidate_expansion_missing_legal_area", Message: record.BatchID})
	}
	if record.ReadinessStatus != ReadyBlockedStatus && record.ReadinessStatus != PaidGateMissingStatus && record.ReadinessStatus != PaidGateBlockedStatus {
		issues = append(issues, Issue{Code: "batch_candidate_expansion_status_not_blocked", Message: record.ReadinessStatus})
	}
	if expected := targetTier(record.TargetCandidateCount); record.TargetCandidateTier != expected {
		issues = append(issues, Issue{Code: "batch_candidate_expansion_target_tier_invalid", Message: fmt.Sprintf("record=%s expected=%s", record.TargetCandidateTier, expected)})
	}
	if record.SourceArchivePath != ArchivePath {
		issues = append(issues, Issue{Code: "batch_candidate_expansion_wrong_archive_path", Message: record.SourceArchivePath})
	}
	if record.ArchiveRecordsRequired < 100 {
		issues = append(issues, Issue{Code: "batch_candidate_expansion_archive_requirement_too_low", Message: fmt.Sprintf("%d", record.ArchiveRecordsRequired)})
	}
	if observed := index.ArchiveCountByBatch[record.BatchID]; observed > 0 && record.ArchiveRecordsObserved != observed {
		issues = append(issues, Issue{Code: "batch_candidate_expansion_observed_count_mismatch", Message: fmt.Sprintf("record=%d observed=%d", record.ArchiveRecordsObserved, observed)})
	}
	if record.ArchiveRecordsObserved < record.ArchiveRecordsRequired {
		issues = append(issues, Issue{Code: "batch_candidate_expansion_archive_too_small", Message: fmt.Sprintf("%s=%d", record.BatchID, record.ArchiveRecordsObserved)})
	}
	if current := index.CurrentCandidateCountByBatch[record.BatchID]; current > 0 && record.CurrentCandidateCount != current {
		issues = append(issues, Issue{Code: "batch_candidate_expansion_current_count_mismatch", Message: fmt.Sprintf("record=%d current=%d", record.CurrentCandidateCount, current)})
	}
	if record.TargetCandidateCount < 30 {
		issues = append(issues, Issue{Code: "batch_candidate_expansion_target_too_low", Message: fmt.Sprintf("%d", record.TargetCandidateCount)})
	}
	candidateIntentIDs := CandidateIntentIDs(record, index)
	if record.ExpansionCandidateSelector != "" && record.ExpansionCandidateSelector != ExpansionCandidateSelectorArchivePrefix {
		issues = append(issues, Issue{Code: "batch_candidate_expansion_selector_invalid", Message: record.ExpansionCandidateSelector})
	}
	if record.ExpansionCandidateSelector == "" && len(record.ExpansionCandidateIntentIDs) == 0 {
		issues = append(issues, Issue{Code: "batch_candidate_expansion_selector_missing", Message: record.BatchID})
	}
	if len(record.ExpansionCandidateIntentIDs) > MaxExpansionCandidateSampleIntentIDs {
		issues = append(issues, Issue{Code: "batch_candidate_expansion_stores_full_intent_list", Message: fmt.Sprintf("stored=%d max_sample=%d", len(record.ExpansionCandidateIntentIDs), MaxExpansionCandidateSampleIntentIDs)})
	}
	if len(record.ExpansionCandidateSampleIntentIDs) > MaxExpansionCandidateSampleIntentIDs {
		issues = append(issues, Issue{Code: "batch_candidate_expansion_sample_too_large", Message: fmt.Sprintf("sample=%d max=%d", len(record.ExpansionCandidateSampleIntentIDs), MaxExpansionCandidateSampleIntentIDs)})
	}
	if record.ExpansionCandidateSelector == ExpansionCandidateSelectorArchivePrefix {
		expectedSample := sampleCandidateIntentIDs(candidateIntentIDs)
		if !sameStringSlice(record.ExpansionCandidateSampleIntentIDs, expectedSample) {
			issues = append(issues, Issue{Code: "batch_candidate_expansion_sample_mismatch", Message: fmt.Sprintf("sample=%d expected=%d", len(record.ExpansionCandidateSampleIntentIDs), len(expectedSample))})
		}
	}
	if len(candidateIntentIDs) < record.TargetCandidateCount {
		issues = append(issues, Issue{Code: "batch_candidate_expansion_too_few_intents", Message: fmt.Sprintf("selected=%d target=%d", len(candidateIntentIDs), record.TargetCandidateCount)})
	}
	paidPassed, paidBlocked, paidMissing := classifyPaidIntent(candidateIntentIDs, index.PaidIntentByIntent)
	seenIntents := make(map[string]bool)
	for _, intentID := range candidateIntentIDs {
		if !idPattern.MatchString(intentID) {
			issues = append(issues, Issue{Code: "batch_candidate_expansion_invalid_intent_id", Message: intentID})
		}
		if seenIntents[intentID] {
			issues = append(issues, Issue{Code: "batch_candidate_expansion_duplicate_intent", Message: intentID})
		}
		seenIntents[intentID] = true
		draft, ok := index.ArchiveByIntent[intentID]
		if !ok {
			issues = append(issues, Issue{Code: "batch_candidate_expansion_intent_missing", Message: intentID})
			continue
		}
		if draft.BatchID != record.BatchID {
			issues = append(issues, Issue{Code: "batch_candidate_expansion_intent_wrong_batch", Message: fmt.Sprintf("%s:%s", intentID, draft.BatchID)})
		}
		if draft.HumanScore < record.MinimumHumanScore {
			issues = append(issues, Issue{Code: "batch_candidate_expansion_intent_score_too_low", Message: fmt.Sprintf("%s=%d", intentID, draft.HumanScore)})
		}
		if record.CTAContextRequired && strings.TrimSpace(draft.CTAContext) == "" {
			issues = append(issues, Issue{Code: "batch_candidate_expansion_intent_without_cta", Message: intentID})
		}
		if strings.TrimSpace(draft.SourceMatrixID) == "" {
			issues = append(issues, Issue{Code: "batch_candidate_expansion_intent_without_source_matrix", Message: intentID})
		}
		if draft.RenderAllowed || draft.SitemapAllowed || draft.PublicationAllowed || draft.PublicPath != "" {
			issues = append(issues, Issue{Code: "batch_candidate_expansion_intent_public_flag", Message: intentID})
		}
	}
	expectedBlockers := index.SourceBlockersByBatch[record.BatchID]
	if len(expectedBlockers) > 0 && !sameStringSet(record.KnownSourceBlockerIntentIDs, expectedBlockers) {
		issues = append(issues, Issue{Code: "batch_candidate_expansion_source_blockers_mismatch", Message: fmt.Sprintf("record=%s expected=%s", strings.Join(record.KnownSourceBlockerIntentIDs, ","), strings.Join(expectedBlockers, ","))})
	}
	if record.PaidIntentGatePath != PaidIntentGatePath {
		issues = append(issues, Issue{Code: "batch_candidate_expansion_paid_gate_path_invalid", Message: record.PaidIntentGatePath})
	}
	if record.PaidIntentPassedCount != len(paidPassed) || record.PaidIntentBlockedCount != len(paidBlocked) || record.PaidIntentMissingCount != len(paidMissing) {
		issues = append(issues, Issue{Code: "batch_candidate_expansion_paid_counts_mismatch", Message: fmt.Sprintf("record=%d/%d/%d expected=%d/%d/%d", record.PaidIntentPassedCount, record.PaidIntentBlockedCount, record.PaidIntentMissingCount, len(paidPassed), len(paidBlocked), len(paidMissing))})
	}
	if !sameStringSet(record.PaidIntentMissingIntentIDs, paidMissing) {
		issues = append(issues, Issue{Code: "batch_candidate_expansion_paid_missing_ids_mismatch", Message: fmt.Sprintf("record=%s expected=%s", strings.Join(record.PaidIntentMissingIntentIDs, ","), strings.Join(paidMissing, ","))})
	}
	if !sameStringSet(record.PaidIntentBlockedIntentIDs, paidBlocked) {
		issues = append(issues, Issue{Code: "batch_candidate_expansion_paid_blocked_ids_mismatch", Message: fmt.Sprintf("record=%s expected=%s", strings.Join(record.PaidIntentBlockedIntentIDs, ","), strings.Join(paidBlocked, ","))})
	}
	expectedActionable := expectedActionableBlockers(len(paidMissing), len(paidBlocked), len(expectedBlockers))
	if !sameStringSet(record.ActionableBlockers, expectedActionable) {
		issues = append(issues, Issue{Code: "batch_candidate_expansion_actionable_blockers_mismatch", Message: fmt.Sprintf("record=%s expected=%s", strings.Join(record.ActionableBlockers, ","), strings.Join(expectedActionable, ","))})
	}
	if len(record.ActionableBlockers) == 0 {
		issues = append(issues, Issue{Code: "batch_candidate_expansion_missing_actionable_blocker", Message: record.BatchID})
	}
	if record.MaxSimilarityAllowed <= 0 || record.MaxSimilarityAllowed > 0.64 {
		issues = append(issues, Issue{Code: "batch_candidate_expansion_similarity_limit_invalid", Message: fmt.Sprintf("%.4f", record.MaxSimilarityAllowed)})
	}
	if record.MaxSimilarityObserved > record.MaxSimilarityAllowed {
		issues = append(issues, Issue{Code: "batch_candidate_expansion_similarity_observed_too_high", Message: fmt.Sprintf("%.4f", record.MaxSimilarityObserved)})
	}
	if index.MaxSimilarity > record.MaxSimilarityAllowed+0.0001 {
		issues = append(issues, Issue{Code: "batch_candidate_expansion_archive_similarity_too_high", Message: fmt.Sprintf("%.4f left=%s right=%s", index.MaxSimilarity, index.MaxSimilarityPair.LeftID, index.MaxSimilarityPair.RightID)})
	}
	if record.MinimumHumanScore < 88 {
		issues = append(issues, Issue{Code: "batch_candidate_expansion_minimum_score_too_low", Message: fmt.Sprintf("%d", record.MinimumHumanScore)})
	}
	if !record.CTAContextRequired {
		issues = append(issues, Issue{Code: "batch_candidate_expansion_without_cta_requirement", Message: record.BatchID})
	}
	if !record.SourceURLAuditRequired {
		issues = append(issues, Issue{Code: "batch_candidate_expansion_without_source_audit", Message: record.BatchID})
	}
	if !record.SourceSpecificityRequired {
		issues = append(issues, Issue{Code: "batch_candidate_expansion_without_source_specificity", Message: record.BatchID})
	}
	if !record.PaidIntentRequired {
		issues = append(issues, Issue{Code: "batch_candidate_expansion_without_paid_intent", Message: record.BatchID})
	}
	if len(paidMissing) > 0 && record.ReadinessStatus != PaidGateMissingStatus {
		issues = append(issues, Issue{Code: "batch_candidate_expansion_paid_gate_missing", Message: fmt.Sprintf("%s missing=%d", record.BatchID, len(paidMissing))})
	}
	if record.PaidIntentRequired && len(paidMissing) == 0 && len(paidBlocked) > 0 && record.ReadinessStatus != PaidGateBlockedStatus {
		issues = append(issues, Issue{Code: "batch_candidate_expansion_paid_gate_blocked", Message: fmt.Sprintf("%s blocked=%d", record.BatchID, len(paidBlocked))})
	}
	if record.PaidIntentRequired && len(paidMissing) == 0 && len(paidBlocked) == 0 && record.ReadinessStatus != ReadyBlockedStatus {
		issues = append(issues, Issue{Code: "batch_candidate_expansion_ready_status_mismatch", Message: record.BatchID})
	}
	if record.NextGate != "batch_candidate_gates" {
		issues = append(issues, Issue{Code: "batch_candidate_expansion_wrong_next_gate", Message: record.NextGate})
	}
	if strings.TrimSpace(record.PublicationBlockReason) == "" {
		issues = append(issues, Issue{Code: "batch_candidate_expansion_missing_block_reason", Message: record.BatchID})
	}
	if record.IndexPolicy != "noindex" {
		issues = append(issues, Issue{Code: "batch_candidate_expansion_invalid_index_policy", Message: record.IndexPolicy})
	}
	if record.ManifestAllowed {
		issues = append(issues, Issue{Code: "batch_candidate_expansion_manifest_allowed", Message: record.BatchID})
	}
	if record.RenderAllowed {
		issues = append(issues, Issue{Code: "batch_candidate_expansion_render_allowed", Message: record.BatchID})
	}
	if record.SitemapAllowed {
		issues = append(issues, Issue{Code: "batch_candidate_expansion_sitemap_allowed", Message: record.BatchID})
	}
	if record.PublicationAllowed {
		issues = append(issues, Issue{Code: "batch_candidate_expansion_publication_allowed", Message: record.BatchID})
	}
	if record.PublicPath != "" {
		issues = append(issues, Issue{Code: "batch_candidate_expansion_has_public_path", Message: record.PublicPath})
	}
	if record.CheckedAt == "" {
		issues = append(issues, Issue{Code: "batch_candidate_expansion_without_checked_at", Message: record.BatchID})
	}
	return Report{Issues: issues}
}

func RefreshRecords(root string) ([]Record, Report) {
	entries, loadReport := LoadRecords(root)
	index, indexReport := BuildExpansionIndex(root)
	issues := append([]Issue{}, loadReport.Issues...)
	issues = append(issues, indexReport.Issues...)
	if len(issues) > 0 {
		return nil, Report{Issues: issues}
	}

	refreshed := make([]Record, 0, len(entries))
	for _, entry := range entries {
		refreshed = append(refreshed, RefreshRecordAgainstIndex(entry.Record, index))
	}
	return refreshed, Report{}
}

func RefreshRecordAgainstIndex(record Record, index ExpansionIndex) Record {
	current := index.CurrentCandidateCountByBatch[record.BatchID]
	archiveCount := index.ArchiveCountByBatch[record.BatchID]
	target := nextTargetCandidateCount(current, archiveCount)
	if target > 0 {
		record.TargetCandidateCount = target
		record.TargetCandidateTier = targetTier(target)
	}
	record.ExpansionCandidateSelector = ExpansionCandidateSelectorArchivePrefix
	candidateIntentIDs := CandidateIntentIDs(record, index)
	record.ExpansionCandidateSampleIntentIDs = sampleCandidateIntentIDs(candidateIntentIDs)
	record.ExpansionCandidateIntentIDs = nil
	paidPassed, paidBlocked, paidMissing := classifyPaidIntent(candidateIntentIDs, index.PaidIntentByIntent)
	record.PaidIntentPassedCount = len(paidPassed)
	record.PaidIntentBlockedCount = len(paidBlocked)
	record.PaidIntentMissingCount = len(paidMissing)
	record.PaidIntentMissingIntentIDs = paidMissing
	record.PaidIntentBlockedIntentIDs = paidBlocked
	record.KnownSourceBlockerIntentIDs = append([]string{}, index.SourceBlockersByBatch[record.BatchID]...)
	record.CurrentCandidateCount = current
	record.ArchiveRecordsObserved = archiveCount
	record.MaxSimilarityObserved = index.MaxSimilarity
	record.ActionableBlockers = expectedActionableBlockers(len(paidMissing), len(paidBlocked), len(record.KnownSourceBlockerIntentIDs))
	if len(paidMissing) > 0 {
		record.ReadinessStatus = PaidGateMissingStatus
	} else if len(paidBlocked) > 0 {
		record.ReadinessStatus = PaidGateBlockedStatus
	} else {
		record.ReadinessStatus = ReadyBlockedStatus
	}
	return record
}

func CandidateIntentIDs(record Record, index ExpansionIndex) []string {
	if record.ExpansionCandidateSelector == ExpansionCandidateSelectorArchivePrefix || len(record.ExpansionCandidateIntentIDs) == 0 {
		return firstN(index.ArchiveIntentIDsByBatch[record.BatchID], record.TargetCandidateCount)
	}
	return append([]string{}, record.ExpansionCandidateIntentIDs...)
}

func sampleCandidateIntentIDs(intentIDs []string) []string {
	return firstN(intentIDs, MaxExpansionCandidateSampleIntentIDs)
}

func nextTargetCandidateCount(current int, archiveCount int) int {
	if archiveCount <= 0 {
		return current
	}
	target := current + 30
	if target < 30 {
		target = 30
	}
	if target > archiveCount {
		target = archiveCount
	}
	return target
}

func targetTier(target int) string {
	return fmt.Sprintf("target_%d", target)
}

func firstN(values []string, count int) []string {
	if count <= 0 || len(values) == 0 {
		return nil
	}
	if count > len(values) {
		count = len(values)
	}
	copyValues := append([]string{}, values[:count]...)
	return copyValues
}

func sameStringSlice(left []string, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}

func LoadRecords(root string) ([]Entry, Report) {
	projectRoot, err := findProjectRoot(root)
	if err != nil {
		return nil, Report{Issues: []Issue{{Code: "project_root_not_found", Message: err.Error()}}}
	}
	path := filepath.Join(projectRoot, "data", "editorial", "batch_candidate_expansion_readiness.jsonl")
	file, err := os.Open(path)
	if err != nil {
		return nil, Report{Issues: []Issue{{Code: "batch_candidate_expansion_readiness_missing", Message: err.Error()}}}
	}
	defer file.Close()

	entries := make([]Entry, 0)
	issues := make([]Issue, 0)
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 0, 4096), 131072)
	lineNumber := 0
	for scanner.Scan() {
		lineNumber++
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var record Record
		if err := json.Unmarshal([]byte(line), &record); err != nil {
			issues = append(issues, Issue{Code: "batch_candidate_expansion_readiness_invalid_json", Message: fmt.Sprintf("line=%d %s", lineNumber, err.Error())})
			continue
		}
		entries = append(entries, Entry{Line: lineNumber, Record: record})
	}
	if err := scanner.Err(); err != nil {
		issues = append(issues, Issue{Code: "batch_candidate_expansion_readiness_scan_failed", Message: err.Error()})
	}
	return entries, Report{Issues: issues}
}

func convertArchiveIssues(report batchdraftarchive.Report) []Issue {
	issues := make([]Issue, 0, len(report.Issues))
	for _, issue := range report.Issues {
		issues = append(issues, Issue{Code: "batch_candidate_expansion_archive_" + issue.Code, Message: issue.Message})
	}
	return issues
}

func convertCandidateIssues(report batchcandidategates.Report) []Issue {
	issues := make([]Issue, 0, len(report.Issues))
	for _, issue := range report.Issues {
		issues = append(issues, Issue{Code: "batch_candidate_expansion_candidate_" + issue.Code, Message: issue.Message})
	}
	return issues
}

func convertSourceSpecificityIssues(report batchsourcespecificity.Report) []Issue {
	issues := make([]Issue, 0, len(report.Issues))
	for _, issue := range report.Issues {
		issues = append(issues, Issue{Code: "batch_candidate_expansion_source_" + issue.Code, Message: issue.Message})
	}
	return issues
}

func convertPaidIntentIssues(report paidintent.Report) []Issue {
	issues := make([]Issue, 0, len(report.Issues))
	for _, issue := range report.Issues {
		issues = append(issues, Issue{Code: "batch_candidate_expansion_paid_intent_" + issue.Code, Message: issue.Message})
	}
	return issues
}

func classifyPaidIntent(intentIDs []string, paidByIntent map[string]paidintent.Record) ([]string, []string, []string) {
	passed := make([]string, 0)
	blocked := make([]string, 0)
	missing := make([]string, 0)
	for _, intentID := range intentIDs {
		record, ok := paidByIntent[intentID]
		if !ok {
			missing = append(missing, intentID)
			continue
		}
		if paidintent.AllowsExpansion(record) {
			passed = append(passed, intentID)
			continue
		}
		blocked = append(blocked, intentID)
	}
	sort.Strings(passed)
	sort.Strings(blocked)
	sort.Strings(missing)
	return passed, blocked, missing
}

func expectedActionableBlockers(missingPaid int, blockedPaid int, sourceBlockers int) []string {
	blockers := make([]string, 0, 3)
	if missingPaid > 0 {
		blockers = append(blockers, "paid_intent_missing_for_expansion")
	}
	if blockedPaid > 0 {
		blockers = append(blockers, "paid_intent_blocked_commercial_risk")
	}
	if sourceBlockers > 0 {
		blockers = append(blockers, "source_specificity_blocked")
	}
	if len(blockers) == 0 {
		blockers = append(blockers, "batch_candidate_gate_pending")
	}
	sort.Strings(blockers)
	return blockers
}

func sameStringSet(left []string, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	leftCopy := append([]string{}, left...)
	rightCopy := append([]string{}, right...)
	sort.Strings(leftCopy)
	sort.Strings(rightCopy)
	for index := range leftCopy {
		if leftCopy[index] != rightCopy[index] {
			return false
		}
	}
	return true
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

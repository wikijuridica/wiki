package batchexpansionstrategy

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"portaljuridico/internal/batchcandidateexpansion"
)

const (
	Path                         = "data/editorial/batch_expansion_strategy.jsonl"
	ReadyNextCandidateGateStatus = "batch_expansion_strategy_blocked_next_candidate_gate"
	BlockedPaidIntentStatus      = "batch_expansion_strategy_blocked_paid_intent"
	ArchiveGrowthRequiredStatus  = "batch_expansion_strategy_blocked_archive_growth_required"
	IndexPolicy                  = "noindex"
)

type Record struct {
	StrategyID                string   `json:"strategy_id"`
	BatchID                   string   `json:"batch_id"`
	LegalArea                 string   `json:"legal_area"`
	StrategyStatus            string   `json:"strategy_status"`
	CurrentCandidateCount     int      `json:"current_candidate_count"`
	NextCandidateTarget       int      `json:"next_candidate_target"`
	MaxGrowthStep             int      `json:"max_growth_step"`
	ArchiveRecordsObserved    int      `json:"archive_records_observed"`
	PaidIntentPassedCount     int      `json:"paid_intent_passed_count"`
	PaidIntentBlockedCount    int      `json:"paid_intent_blocked_count"`
	RequiresVerifiedSource    bool     `json:"requires_verified_source"`
	RequiresPaidIntent        bool     `json:"requires_paid_intent"`
	RequiresContextualCTA     bool     `json:"requires_contextual_cta"`
	RequiresHumanScore        bool     `json:"requires_human_score"`
	RequiresSemanticDiversity bool     `json:"requires_semantic_diversity"`
	SemanticAxes              []string `json:"semantic_axes"`
	ValidationCommands        []string `json:"validation_commands"`
	NextAction                string   `json:"next_action"`
	PublicationBlockReason    string   `json:"publication_block_reason"`
	IndexPolicy               string   `json:"index_policy"`
	ManifestAllowed           bool     `json:"manifest_allowed"`
	RenderAllowed             bool     `json:"render_allowed"`
	SitemapAllowed            bool     `json:"sitemap_allowed"`
	PublicationAllowed        bool     `json:"publication_allowed"`
	PublicPath                string   `json:"public_path"`
	CheckedAt                 string   `json:"checked_at"`
}

type Entry struct {
	Line   int
	Record Record
}

type ReadinessSnapshot struct {
	BatchID                string
	LegalArea              string
	ReadinessStatus        string
	ArchiveRecordsObserved int
	CurrentCandidateCount  int
	PaidIntentPassedCount  int
	PaidIntentBlockedCount int
}

type StrategyIndex struct {
	ReadinessByBatch map[string]ReadinessSnapshot
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
	entries, loadReport := LoadRecords(root)
	index, indexReport := BuildStrategyIndex(root)
	issues := append([]Issue{}, loadReport.Issues...)
	issues = append(issues, indexReport.Issues...)
	if len(entries) != len(index.ReadinessByBatch) {
		issues = append(issues, Issue{Code: "batch_expansion_strategy_count_mismatch", Message: fmt.Sprintf("records=%d readiness=%d", len(entries), len(index.ReadinessByBatch))})
	}
	seen := make(map[string]int)
	for _, entry := range entries {
		recordReport := ValidateRecordAgainstIndex(entry.Record, index)
		for _, issue := range recordReport.Issues {
			issue.Message = fmt.Sprintf("line=%d %s", entry.Line, issue.Message)
			issues = append(issues, issue)
		}
		if previous := seen[entry.Record.BatchID]; previous > 0 {
			issues = append(issues, Issue{Code: "batch_expansion_strategy_duplicate_batch", Message: fmt.Sprintf("line=%d previous_line=%d batch=%s", entry.Line, previous, entry.Record.BatchID)})
		}
		seen[entry.Record.BatchID] = entry.Line
	}
	for batchID := range index.ReadinessByBatch {
		if seen[batchID] == 0 {
			issues = append(issues, Issue{Code: "batch_expansion_strategy_missing_batch", Message: batchID})
		}
	}
	return Report{Issues: issues}
}

func BuildStrategyIndex(root string) (StrategyIndex, Report) {
	readinessEntries, readinessReport := batchcandidateexpansion.LoadRecords(root)
	issues := make([]Issue, 0, len(readinessReport.Issues))
	for _, issue := range readinessReport.Issues {
		issues = append(issues, Issue{Code: "batch_expansion_strategy_readiness_" + issue.Code, Message: issue.Message})
	}
	index := StrategyIndex{ReadinessByBatch: make(map[string]ReadinessSnapshot)}
	for _, entry := range readinessEntries {
		record := entry.Record
		index.ReadinessByBatch[record.BatchID] = ReadinessSnapshot{
			BatchID:                record.BatchID,
			LegalArea:              record.LegalArea,
			ReadinessStatus:        record.ReadinessStatus,
			ArchiveRecordsObserved: record.ArchiveRecordsObserved,
			CurrentCandidateCount:  record.CurrentCandidateCount,
			PaidIntentPassedCount:  record.PaidIntentPassedCount,
			PaidIntentBlockedCount: record.PaidIntentBlockedCount,
		}
	}
	return index, Report{Issues: issues}
}

func RefreshRecords(root string) ([]Record, Report) {
	index, report := BuildStrategyIndex(root)
	if !report.Passed() {
		return nil, report
	}
	batchIDs := make([]string, 0, len(index.ReadinessByBatch))
	for batchID := range index.ReadinessByBatch {
		batchIDs = append(batchIDs, batchID)
	}
	sort.Strings(batchIDs)
	records := make([]Record, 0, len(batchIDs))
	for _, batchID := range batchIDs {
		records = append(records, BuildRecord(index.ReadinessByBatch[batchID]))
	}
	return records, Report{}
}

func BuildRecord(readiness ReadinessSnapshot) Record {
	status := ReadyNextCandidateGateStatus
	nextTarget := minInt(readiness.CurrentCandidateCount+30, readiness.ArchiveRecordsObserved)
	nextAction := "expandir batch_candidate_gates para o proximo alvo usando paid-intent, fonte especifica, score humano, diversidade semantica e CTA contextual antes de novo refresh"
	if readiness.PaidIntentBlockedCount > 0 || readiness.ReadinessStatus == batchcandidateexpansion.PaidGateBlockedStatus {
		status = BlockedPaidIntentStatus
		nextTarget = readiness.CurrentCandidateCount
		nextAction = "refinar paid-intent dos bloqueados ou manter bloqueio comercial antes de qualquer expansao desta familia"
	} else if readiness.CurrentCandidateCount >= readiness.ArchiveRecordsObserved {
		status = ArchiveGrowthRequiredStatus
		nextTarget = readiness.CurrentCandidateCount
		nextAction = "gerar mais rascunhos no batch_draft_expansion_archive com diversidade semantica, fonte especifica, CTA contextual e validacao antes de novo avanço"
	}
	return Record{
		StrategyID:                "strategy-" + readiness.BatchID + "-cycle-45",
		BatchID:                   readiness.BatchID,
		LegalArea:                 readiness.LegalArea,
		StrategyStatus:            status,
		CurrentCandidateCount:     readiness.CurrentCandidateCount,
		NextCandidateTarget:       nextTarget,
		MaxGrowthStep:             30,
		ArchiveRecordsObserved:    readiness.ArchiveRecordsObserved,
		PaidIntentPassedCount:     readiness.PaidIntentPassedCount,
		PaidIntentBlockedCount:    readiness.PaidIntentBlockedCount,
		RequiresVerifiedSource:    true,
		RequiresPaidIntent:        true,
		RequiresContextualCTA:     true,
		RequiresHumanScore:        true,
		RequiresSemanticDiversity: true,
		SemanticAxes: []string{
			"problema_do_leitor",
			"fonte_oficial_especifica",
			"documentos_para_triagem",
			"risco_juridico_concreto",
			"acao_digital_possivel",
			"cta_whatsapp_com_origem",
		},
		ValidationCommands: []string{
			"./tools/check-batch-expansion-strategy",
			"./tools/advance-batch-candidate-gates --expect-total <expected-current>",
			"./tools/expand-batch-candidate-gates",
			"./tools/refresh-batch-candidate-pipeline",
			"./tools/check-batch-candidate-gates",
			"./tools/check-batch-candidate-reviews",
			"./tools/check-batch-source-specificity",
			"./tools/check-batch-final-authorial-drafts",
			"./tools/check-paid-intent",
		},
		NextAction:             nextAction,
		PublicationBlockReason: "estrategia de expansao e apenas planejamento bloqueado; publicar exige SEO final, revisao, qualidade, canonical, manifest e novo checkpoint",
		IndexPolicy:            IndexPolicy,
		CheckedAt:              "2026-06-09",
	}
}

func ValidateRecordAgainstIndex(record Record, index StrategyIndex) Report {
	issues := make([]Issue, 0)
	if record.StrategyID == "" || !idPattern.MatchString(record.StrategyID) {
		issues = append(issues, Issue{Code: "batch_expansion_strategy_invalid_strategy_id", Message: record.StrategyID})
	}
	if record.BatchID == "" || !idPattern.MatchString(record.BatchID) {
		issues = append(issues, Issue{Code: "batch_expansion_strategy_invalid_batch_id", Message: record.BatchID})
	}
	readiness, ok := index.ReadinessByBatch[record.BatchID]
	if !ok {
		issues = append(issues, Issue{Code: "batch_expansion_strategy_missing_readiness", Message: record.BatchID})
	} else {
		if record.LegalArea != readiness.LegalArea {
			issues = append(issues, Issue{Code: "batch_expansion_strategy_legal_area_mismatch", Message: record.BatchID})
		}
		if record.CurrentCandidateCount != readiness.CurrentCandidateCount {
			issues = append(issues, Issue{Code: "batch_expansion_strategy_current_count_mismatch", Message: fmt.Sprintf("record=%d readiness=%d", record.CurrentCandidateCount, readiness.CurrentCandidateCount)})
		}
		if record.ArchiveRecordsObserved != readiness.ArchiveRecordsObserved {
			issues = append(issues, Issue{Code: "batch_expansion_strategy_archive_count_mismatch", Message: fmt.Sprintf("record=%d readiness=%d", record.ArchiveRecordsObserved, readiness.ArchiveRecordsObserved)})
		}
		if record.PaidIntentPassedCount != readiness.PaidIntentPassedCount || record.PaidIntentBlockedCount != readiness.PaidIntentBlockedCount {
			issues = append(issues, Issue{Code: "batch_expansion_strategy_paid_counts_mismatch", Message: record.BatchID})
		}
		if readiness.PaidIntentBlockedCount == 0 && readiness.CurrentCandidateCount < readiness.ArchiveRecordsObserved && record.NextCandidateTarget <= record.CurrentCandidateCount {
			issues = append(issues, Issue{Code: "batch_expansion_strategy_not_growing_ready_family", Message: record.BatchID})
		}
		if readiness.PaidIntentBlockedCount > 0 && record.NextCandidateTarget > record.CurrentCandidateCount {
			issues = append(issues, Issue{Code: "batch_expansion_strategy_grows_paid_blocked_family", Message: record.BatchID})
		}
		if readiness.PaidIntentBlockedCount == 0 && readiness.CurrentCandidateCount >= readiness.ArchiveRecordsObserved {
			if record.StrategyStatus != ArchiveGrowthRequiredStatus {
				issues = append(issues, Issue{Code: "batch_expansion_strategy_archive_growth_not_required", Message: record.BatchID})
			}
			if record.NextCandidateTarget != record.CurrentCandidateCount {
				issues = append(issues, Issue{Code: "batch_expansion_strategy_archive_growth_target_mismatch", Message: record.BatchID})
			}
		}
		if record.NextCandidateTarget > readiness.ArchiveRecordsObserved {
			issues = append(issues, Issue{Code: "batch_expansion_strategy_target_exceeds_archive", Message: fmt.Sprintf("target=%d archive=%d", record.NextCandidateTarget, readiness.ArchiveRecordsObserved)})
		}
		if readiness.PaidIntentBlockedCount > 0 && record.StrategyStatus != BlockedPaidIntentStatus {
			issues = append(issues, Issue{Code: "batch_expansion_strategy_paid_blocker_not_preserved", Message: record.BatchID})
		}
		if readiness.PaidIntentBlockedCount == 0 && readiness.CurrentCandidateCount < readiness.ArchiveRecordsObserved && record.StrategyStatus != ReadyNextCandidateGateStatus {
			issues = append(issues, Issue{Code: "batch_expansion_strategy_ready_status_mismatch", Message: record.BatchID})
		}
	}
	if record.StrategyStatus != ReadyNextCandidateGateStatus && record.StrategyStatus != BlockedPaidIntentStatus && record.StrategyStatus != ArchiveGrowthRequiredStatus {
		issues = append(issues, Issue{Code: "batch_expansion_strategy_status_not_blocked", Message: record.StrategyStatus})
	}
	if record.MaxGrowthStep <= 0 || record.MaxGrowthStep > 30 {
		issues = append(issues, Issue{Code: "batch_expansion_strategy_growth_step_too_large", Message: fmt.Sprintf("%d", record.MaxGrowthStep)})
	}
	if !record.RequiresVerifiedSource {
		issues = append(issues, Issue{Code: "batch_expansion_strategy_without_verified_source", Message: record.BatchID})
	}
	if !record.RequiresPaidIntent {
		issues = append(issues, Issue{Code: "batch_expansion_strategy_without_paid_intent", Message: record.BatchID})
	}
	if !record.RequiresContextualCTA {
		issues = append(issues, Issue{Code: "batch_expansion_strategy_without_contextual_cta", Message: record.BatchID})
	}
	if !record.RequiresHumanScore {
		issues = append(issues, Issue{Code: "batch_expansion_strategy_without_human_score", Message: record.BatchID})
	}
	if !record.RequiresSemanticDiversity {
		issues = append(issues, Issue{Code: "batch_expansion_strategy_without_semantic_diversity", Message: record.BatchID})
	}
	if !containsAll(record.SemanticAxes, []string{"problema_do_leitor", "fonte_oficial_especifica", "documentos_para_triagem", "risco_juridico_concreto", "acao_digital_possivel", "cta_whatsapp_com_origem"}) {
		issues = append(issues, Issue{Code: "batch_expansion_strategy_weak_semantic_axes", Message: strings.Join(record.SemanticAxes, ",")})
	}
	if !containsAll(record.ValidationCommands, []string{"./tools/check-batch-expansion-strategy", "./tools/refresh-batch-candidate-pipeline", "./tools/check-batch-source-specificity", "./tools/check-batch-final-authorial-drafts", "./tools/check-paid-intent"}) {
		issues = append(issues, Issue{Code: "batch_expansion_strategy_weak_validation_plan", Message: strings.Join(record.ValidationCommands, ",")})
	}
	if strings.TrimSpace(record.NextAction) == "" {
		issues = append(issues, Issue{Code: "batch_expansion_strategy_missing_next_action", Message: record.BatchID})
	}
	if strings.TrimSpace(record.PublicationBlockReason) == "" {
		issues = append(issues, Issue{Code: "batch_expansion_strategy_missing_block_reason", Message: record.BatchID})
	}
	if record.IndexPolicy != IndexPolicy {
		issues = append(issues, Issue{Code: "batch_expansion_strategy_invalid_index_policy", Message: record.IndexPolicy})
	}
	if record.ManifestAllowed {
		issues = append(issues, Issue{Code: "batch_expansion_strategy_manifest_allowed", Message: record.BatchID})
	}
	if record.RenderAllowed {
		issues = append(issues, Issue{Code: "batch_expansion_strategy_render_allowed", Message: record.BatchID})
	}
	if record.SitemapAllowed {
		issues = append(issues, Issue{Code: "batch_expansion_strategy_sitemap_allowed", Message: record.BatchID})
	}
	if record.PublicationAllowed {
		issues = append(issues, Issue{Code: "batch_expansion_strategy_publication_allowed", Message: record.BatchID})
	}
	if record.PublicPath != "" {
		issues = append(issues, Issue{Code: "batch_expansion_strategy_has_public_path", Message: record.PublicPath})
	}
	if record.CheckedAt == "" {
		issues = append(issues, Issue{Code: "batch_expansion_strategy_without_checked_at", Message: record.BatchID})
	}
	return Report{Issues: issues}
}

func LoadRecords(root string) ([]Entry, Report) {
	projectRoot, err := findProjectRoot(root)
	if err != nil {
		return nil, Report{Issues: []Issue{{Code: "project_root_not_found", Message: err.Error()}}}
	}
	path := filepath.Join(projectRoot, Path)
	file, err := os.Open(path)
	if err != nil {
		return nil, Report{Issues: []Issue{{Code: "batch_expansion_strategy_missing", Message: err.Error()}}}
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
			issues = append(issues, Issue{Code: "batch_expansion_strategy_invalid_json", Message: fmt.Sprintf("line=%d %s", lineNumber, err.Error())})
			continue
		}
		entries = append(entries, Entry{Line: lineNumber, Record: record})
	}
	if err := scanner.Err(); err != nil {
		issues = append(issues, Issue{Code: "batch_expansion_strategy_scan_failed", Message: err.Error()})
	}
	return entries, Report{Issues: issues}
}

func containsAll(values []string, required []string) bool {
	seen := make(map[string]bool)
	for _, value := range values {
		seen[value] = true
	}
	for _, value := range required {
		if !seen[value] {
			return false
		}
	}
	return true
}

func minInt(left int, right int) int {
	if left < right {
		return left
	}
	return right
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

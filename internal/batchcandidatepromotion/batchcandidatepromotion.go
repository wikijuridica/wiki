package batchcandidatepromotion

import (
	"fmt"
	"sort"

	"portaljuridico/internal/batchcandidateexpansion"
	"portaljuridico/internal/batchcandidategates"
	"portaljuridico/internal/batchexpansionstrategy"
	"portaljuridico/internal/paidintent"
)

type Issue struct {
	Code    string
	Message string
}

type Report struct {
	Issues []Issue
}

func ExpandFromReadiness(root string) ([]batchcandidategates.Record, Report) {
	return promoteFromReadiness(root, materializationTarget)
}

func AdvanceToNextTargets(root string) ([]batchcandidategates.Record, Report) {
	return promoteFromReadiness(root, nextTargetMaterialization)
}

func ValidateTotalSelected(records []batchcandidategates.Record, expected int) Report {
	if expected <= 0 {
		return Report{}
	}
	total := 0
	for _, record := range records {
		total += len(record.SelectedUniqueIntentIDs)
	}
	if total != expected {
		return Report{Issues: []Issue{{
			Code:    "batch_candidate_unexpected_total_selected",
			Message: fmt.Sprintf("selected=%d expected=%d", total, expected),
		}}}
	}
	return Report{}
}

func promoteFromReadiness(root string, targetFor func(batchcandidateexpansion.Record, batchexpansionstrategy.Record) int) ([]batchcandidategates.Record, Report) {
	entries, loadReport := batchcandidategates.LoadRecords(root)
	archiveIndex, archiveReport := batchcandidategates.BuildArchiveIndex(root)
	expansionIndex, expansionIndexReport := batchcandidateexpansion.BuildExpansionIndex(root)
	readinessEntries, readinessReport := batchcandidateexpansion.LoadRecords(root)
	paidEntries, paidReport := paidintent.LoadRecords(root)
	strategyEntries, strategyReport := batchexpansionstrategy.LoadRecords(root)
	issues := append(convertGateIssues(loadReport), convertGateIssues(archiveReport)...)
	issues = append(issues, convertReadinessIssues(expansionIndexReport)...)
	issues = append(issues, convertReadinessIssues(readinessReport)...)
	issues = append(issues, convertPaidIssues(paidReport)...)
	issues = append(issues, convertStrategyIssues(strategyReport)...)
	if len(issues) > 0 {
		return nil, Report{Issues: issues}
	}

	readinessByBatch := make(map[string]batchcandidateexpansion.Record)
	for _, entry := range readinessEntries {
		readinessByBatch[entry.Record.BatchID] = entry.Record
	}
	strategyByBatch := make(map[string]batchexpansionstrategy.Record)
	for _, entry := range strategyEntries {
		strategyByBatch[entry.Record.BatchID] = entry.Record
	}
	paidByIntent := make(map[string]paidintent.Record)
	for _, entry := range paidEntries {
		paidByIntent[entry.Record.UniqueIntentID] = entry.Record
	}
	templatesByBatch := make(map[string]batchcandidategates.Record)
	batchIDs := make([]string, 0, len(entries))
	for _, entry := range entries {
		if _, ok := templatesByBatch[entry.Record.BatchID]; ok {
			continue
		}
		templatesByBatch[entry.Record.BatchID] = entry.Record
		batchIDs = append(batchIDs, entry.Record.BatchID)
	}
	sort.Strings(batchIDs)

	expanded := make([]batchcandidategates.Record, 0, len(batchIDs))
	for _, batchID := range batchIDs {
		record := templatesByBatch[batchID]
		readiness, ok := readinessByBatch[record.BatchID]
		if !ok {
			issues = append(issues, Issue{Code: "batch_candidate_missing_expansion_readiness", Message: record.BatchID})
			continue
		}
		strategy, ok := strategyByBatch[record.BatchID]
		if !ok {
			issues = append(issues, Issue{Code: "batch_candidate_missing_expansion_strategy", Message: record.BatchID})
			continue
		}
		candidateIntentIDs := batchcandidateexpansion.CandidateIntentIDs(readiness, expansionIndex)
		selected := make([]string, 0, len(candidateIntentIDs))
		for _, intentID := range candidateIntentIDs {
			if !paidintent.AllowsExpansion(paidByIntent[intentID]) {
				continue
			}
			selected = append(selected, intentID)
		}
		target := targetFor(readiness, strategy)
		if len(selected) < target {
			issues = append(issues, Issue{Code: "batch_candidate_too_few_paid_passed_intents", Message: fmt.Sprintf("%s=%d current=%d", record.BatchID, len(selected), target)})
			continue
		}
		selected = firstN(selected, target)
		if len(selected) < 18 {
			issues = append(issues, Issue{Code: "batch_candidate_too_few_paid_passed_intents", Message: fmt.Sprintf("%s=%d", record.BatchID, len(selected))})
			continue
		}
		record.SelectedUniqueIntentIDs = selected
		record.ArchiveMinimumRecords = readiness.ArchiveRecordsRequired
		record.MaxSimilarityObserved = archiveIndex.MaxSimilarity
		record.MinimumHumanScore = readiness.MinimumHumanScore
		record.PublicationBlockReason = "P0 bloqueado: seleção expandida exige fonte específica, revisão jurídico-editorial, SEO final, manifesto público finito e aprovação explícita antes de publicar."
		record.RenderAllowed = false
		record.SitemapAllowed = false
		record.PublicationAllowed = false
		record.PublicPath = ""
		record.CheckedAt = readiness.CheckedAt
		expanded = append(expanded, record)
	}
	if len(issues) > 0 {
		return expanded, Report{Issues: issues}
	}
	sort.Slice(expanded, func(left int, right int) bool {
		return expanded[left].BatchID < expanded[right].BatchID
	})
	return expanded, Report{}
}

func (r Report) Passed() bool { return len(r.Issues) == 0 }

func (r Report) Messages() []string {
	messages := make([]string, 0, len(r.Issues))
	for _, issue := range r.Issues {
		messages = append(messages, issue.Code+": "+issue.Message)
	}
	return messages
}

func convertGateIssues(report batchcandidategates.Report) []Issue {
	issues := make([]Issue, 0, len(report.Issues))
	for _, issue := range report.Issues {
		issues = append(issues, Issue{Code: "batch_candidate_promotion_gate_" + issue.Code, Message: issue.Message})
	}
	return issues
}

func convertReadinessIssues(report batchcandidateexpansion.Report) []Issue {
	issues := make([]Issue, 0, len(report.Issues))
	for _, issue := range report.Issues {
		issues = append(issues, Issue{Code: "batch_candidate_promotion_readiness_" + issue.Code, Message: issue.Message})
	}
	return issues
}

func convertPaidIssues(report paidintent.Report) []Issue {
	issues := make([]Issue, 0, len(report.Issues))
	for _, issue := range report.Issues {
		issues = append(issues, Issue{Code: "batch_candidate_promotion_paid_intent_" + issue.Code, Message: issue.Message})
	}
	return issues
}

func convertStrategyIssues(report batchexpansionstrategy.Report) []Issue {
	issues := make([]Issue, 0, len(report.Issues))
	for _, issue := range report.Issues {
		issues = append(issues, Issue{Code: "batch_candidate_promotion_strategy_" + issue.Code, Message: issue.Message})
	}
	return issues
}

func firstN(values []string, limit int) []string {
	if limit > len(values) {
		limit = len(values)
	}
	return append([]string{}, values[:limit]...)
}

func materializationTarget(readiness batchcandidateexpansion.Record, strategy batchexpansionstrategy.Record) int {
	if readiness.BatchID == "batch-previdenciario-digital" &&
		readiness.ReadinessStatus == batchcandidateexpansion.ReadyBlockedStatus &&
		strategy.CurrentCandidateCount < readiness.TargetCandidateCount &&
		readiness.TargetCandidateCount <= 30 {
		return readiness.TargetCandidateCount
	}
	return strategy.CurrentCandidateCount
}

func nextTargetMaterialization(readiness batchcandidateexpansion.Record, strategy batchexpansionstrategy.Record) int {
	if readiness.ReadinessStatus != batchcandidateexpansion.ReadyBlockedStatus {
		return strategy.CurrentCandidateCount
	}
	if strategy.StrategyStatus != batchexpansionstrategy.ReadyNextCandidateGateStatus {
		return strategy.CurrentCandidateCount
	}
	return strategy.NextCandidateTarget
}

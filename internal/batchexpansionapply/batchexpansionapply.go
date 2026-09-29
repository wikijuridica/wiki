package batchexpansionapply

import (
	"fmt"
	"sort"

	"portaljuridico/internal/batchcandidateexpansion"
	"portaljuridico/internal/batchdraftarchive"
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

func ApplyStrategy(root string, records []batchcandidateexpansion.Record) ([]batchcandidateexpansion.Record, Report) {
	strategyEntries, strategyReport := batchexpansionstrategy.LoadRecords(root)
	archiveEntries, archiveReport := batchdraftarchive.LoadRecords(root)
	expansionIndex, expansionReport := batchcandidateexpansion.BuildExpansionIndex(root)
	issues := append(convertStrategyIssues(strategyReport), convertArchiveIssues(archiveReport)...)
	issues = append(issues, convertExpansionIssues(expansionReport)...)
	if len(issues) > 0 {
		return nil, Report{Issues: issues}
	}

	strategyByBatch := make(map[string]batchexpansionstrategy.Record)
	for _, entry := range strategyEntries {
		strategyByBatch[entry.Record.BatchID] = entry.Record
	}
	eligibleByBatch := make(map[string][]string)
	for _, entry := range archiveEntries {
		draft := entry.Record
		evaluation := paidintent.EvaluateArchiveDraft(draft)
		if !eligibleForStrategyExpansion(evaluation) {
			continue
		}
		eligibleByBatch[draft.BatchID] = append(eligibleByBatch[draft.BatchID], draft.UniqueIntentID)
	}

	applied := make([]batchcandidateexpansion.Record, 0, len(records))
	for _, record := range records {
		strategy, ok := strategyByBatch[record.BatchID]
		if !ok {
			issues = append(issues, Issue{Code: "batch_expansion_apply_missing_strategy", Message: record.BatchID})
			applied = append(applied, record)
			continue
		}
		if strategy.StrategyStatus == batchexpansionstrategy.ReadyNextCandidateGateStatus {
			selected := firstN(eligibleByBatch[record.BatchID], strategy.NextCandidateTarget)
			if len(selected) < strategy.NextCandidateTarget {
				issues = append(issues, Issue{Code: "batch_expansion_apply_too_few_paid_eligible", Message: fmt.Sprintf("%s=%d target=%d", record.BatchID, len(selected), strategy.NextCandidateTarget)})
				applied = append(applied, record)
				continue
			}
			record.TargetCandidateCount = strategy.NextCandidateTarget
			record.TargetCandidateTier = targetTier(strategy.NextCandidateTarget)
			record.ExpansionCandidateIntentIDs = selected
		}
		record = batchcandidateexpansion.RefreshRecordAgainstIndex(record, expansionIndex)
		applied = append(applied, record)
	}
	if len(issues) > 0 {
		return applied, Report{Issues: issues}
	}
	sort.Slice(applied, func(left int, right int) bool {
		return applied[left].BatchID < applied[right].BatchID
	})
	return applied, Report{}
}

func eligibleForStrategyExpansion(record paidintent.Record) bool {
	if paidintent.AllowsExpansion(record) {
		return true
	}
	if record.RenderAllowed || record.SitemapAllowed || record.PublicationAllowed || record.PublicPath != "" {
		return false
	}
	switch record.PaidIntentStatus {
	case paidintent.MissingPaidSignalStatus, paidintent.CTAOnlyBlockedStatus, paidintent.LowBusinessScoreStatus:
		return true
	default:
		return false
	}
}

func firstN(values []string, limit int) []string {
	if limit > len(values) {
		limit = len(values)
	}
	out := append([]string{}, values[:limit]...)
	return out
}

func targetTier(target int) string {
	switch {
	case target >= 100:
		return "target_100"
	case target >= 60:
		return "target_60"
	default:
		return "target_30"
	}
}

func convertStrategyIssues(report batchexpansionstrategy.Report) []Issue {
	issues := make([]Issue, 0, len(report.Issues))
	for _, issue := range report.Issues {
		issues = append(issues, Issue{Code: "batch_expansion_apply_strategy_" + issue.Code, Message: issue.Message})
	}
	return issues
}

func convertArchiveIssues(report batchdraftarchive.Report) []Issue {
	issues := make([]Issue, 0, len(report.Issues))
	for _, issue := range report.Issues {
		issues = append(issues, Issue{Code: "batch_expansion_apply_archive_" + issue.Code, Message: issue.Message})
	}
	return issues
}

func convertExpansionIssues(report batchcandidateexpansion.Report) []Issue {
	issues := make([]Issue, 0, len(report.Issues))
	for _, issue := range report.Issues {
		issues = append(issues, Issue{Code: "batch_expansion_apply_expansion_" + issue.Code, Message: issue.Message})
	}
	return issues
}

func (r Report) Passed() bool { return len(r.Issues) == 0 }

func (r Report) Messages() []string {
	messages := make([]string, 0, len(r.Issues))
	for _, issue := range r.Issues {
		messages = append(messages, issue.Code+": "+issue.Message)
	}
	return messages
}

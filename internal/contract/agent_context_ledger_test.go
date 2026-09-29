package contract_test

import (
	"testing"

	"portaljuridico/internal/agentcontext"
)

func TestAgentContextLedgerPersistsSubagentSummaries(t *testing.T) {
	report := agentcontext.Validate(".")
	if !report.Passed() {
		t.Fatalf("agent context ledger failed: %v", report.Messages())
	}

	records, loadReport := agentcontext.LoadRecords(".")
	if !loadReport.Passed() {
		t.Fatalf("could not load agent context ledger: %v", loadReport.Messages())
	}
	if len(records) < 3 {
		t.Fatalf("agent context records=%d, want at least cycle 40 subagent summaries", len(records))
	}

	var hasSourceResearch, hasReadinessAudit, hasPaidIntentReview bool
	requiredCycle46 := map[string]string{
		"expansion_strategy_review":      "019eae5a-051b-7e23-bdbb-5be0b60a159d",
		"agent_contract_hardening":       "019eae5a-2292-78b2-8ee2-6aa8710f2ff4",
		"source_quality_expansion_audit": "019eae5a-3f8c-7320-9466-a1f6b31a36ac",
		"previdenciario_contract_review": "019eae7c-8e95-7402-a285-6b56d3cdc0fa",
		"previdenciario_gate_audit":      "019eae7c-8fc9-7b60-a89a-83c64f426fdb",
	}
	requiredCycle47 := map[string]string{
		"candidate_510_promotion_audit":    "019eae90-3bb8-72d3-b21f-0b29bc834706",
		"cycle_47_performance_shard_audit": "019eae90-535f-7f90-85ec-4821ff346729",
		"previdenciario_60_semantic_audit": "019eae90-73f9-7470-8798-9ceacefc4a2f",
	}
	seenCycle46 := make(map[string]agentcontext.Record)
	seenCycle47 := make(map[string]agentcontext.Record)
	for _, entry := range records {
		record := entry.Record
		if record.Cycle < 40 {
			continue
		}
		switch record.UsagePolicy {
		case agentcontext.ReferenceOnlyPolicy:
			if record.RepoWriteAllowed {
				t.Fatalf("line=%d reference policy allowed repo write", entry.Line)
			}
		case agentcontext.DelegatedWritePolicy:
			if !record.RepoWriteAllowed {
				t.Fatalf("line=%d delegated write policy missing repo write flag", entry.Line)
			}
		default:
			t.Fatalf("line=%d policy=%q", entry.Line, record.UsagePolicy)
		}
		if !record.CodexValidationRequired || !record.ClosedBeforeCheckpoint {
			t.Fatalf("line=%d unsafe context flags validation=%t closed=%t", entry.Line, record.CodexValidationRequired, record.ClosedBeforeCheckpoint)
		}
		switch record.TaskKind {
		case "source_research":
			hasSourceResearch = true
		case "readiness_audit":
			hasReadinessAudit = true
		case "paid_intent_review":
			hasPaidIntentReview = true
		}
		if record.Cycle == 46 {
			seenCycle46[record.TaskKind] = record
		}
		if record.Cycle == 47 {
			seenCycle47[record.TaskKind] = record
		}
	}
	if !hasSourceResearch || !hasReadinessAudit || !hasPaidIntentReview {
		t.Fatalf("missing expected cycle 40 agent summaries: source=%t readiness=%t paid=%t", hasSourceResearch, hasReadinessAudit, hasPaidIntentReview)
	}
	for taskKind, agentID := range requiredCycle46 {
		record, ok := seenCycle46[taskKind]
		if !ok {
			t.Fatalf("missing cycle 46 agent context for %s", taskKind)
		}
		if record.AgentID != agentID {
			t.Fatalf("cycle 46 %s agent_id=%q, want %q", taskKind, record.AgentID, agentID)
		}
		if len(record.Evidence) < 2 || len(record.Risks) == 0 || record.IntegrationDecision == "" {
			t.Fatalf("cycle 46 %s lacks durable context: evidence=%d risks=%d integration=%q", taskKind, len(record.Evidence), len(record.Risks), record.IntegrationDecision)
		}
		if !record.CodexValidationRequired || !record.ClosedBeforeCheckpoint {
			t.Fatalf("cycle 46 %s not closed for checkpoint validation=%t closed=%t", taskKind, record.CodexValidationRequired, record.ClosedBeforeCheckpoint)
		}
	}
	for taskKind, agentID := range requiredCycle47 {
		record, ok := seenCycle47[taskKind]
		if !ok {
			t.Fatalf("missing cycle 47 agent context for %s", taskKind)
		}
		if record.AgentID != agentID {
			t.Fatalf("cycle 47 %s agent_id=%q, want %q", taskKind, record.AgentID, agentID)
		}
		if len(record.Evidence) < 2 || len(record.Risks) == 0 || record.IntegrationDecision == "" {
			t.Fatalf("cycle 47 %s lacks durable context: evidence=%d risks=%d integration=%q", taskKind, len(record.Evidence), len(record.Risks), record.IntegrationDecision)
		}
		if !record.CodexValidationRequired || !record.ClosedBeforeCheckpoint {
			t.Fatalf("cycle 47 %s not closed for checkpoint validation=%t closed=%t", taskKind, record.CodexValidationRequired, record.ClosedBeforeCheckpoint)
		}
	}
}

func TestAgentContextLedgerAllowsDelegatedRepoWriteOnlyAfterCodexValidation(t *testing.T) {
	record := agentcontext.Record{
		Cycle:                   46,
		AgentID:                 "019eae5a-delegated-example",
		Nickname:                "Ada",
		TaskKind:                "contract_patch",
		Scope:                   "editar documentos em escopo disjunto e deixar contexto persistente para o Codex principal",
		Status:                  "integrated",
		UsagePolicy:             agentcontext.DelegatedWritePolicy,
		Summary:                 "Agente editou escopo disjunto, registrou evidencias, riscos e deixou integracao pendente de validacao pelo Codex principal.",
		Evidence:                []string{"AGENTS.md", "internal/contract/agent_context_ledger_test.go"},
		Risks:                   []string{"mudanca delegada pode conflitar se o Codex principal nao revisar o diff antes do commit"},
		IntegrationDecision:     "Codex principal revisou o diff, rodou teste de contrato e integrou a alteracao antes do checkpoint.",
		RepoWriteAllowed:        true,
		CodexValidationRequired: true,
		ClosedBeforeCheckpoint:  true,
		RecordedAt:              "2026-06-09T22:30:00-03:00",
	}

	report := agentcontext.ValidateRecord(record)
	if !report.Passed() {
		t.Fatalf("delegated write record failed: %v", report.Messages())
	}
}

func TestAgentContextLedgerRejectsUnsafeDelegationState(t *testing.T) {
	record := agentcontext.Record{
		Cycle:                   41,
		AgentID:                 "019eadc3-example",
		Nickname:                "Unsafe",
		TaskKind:                "source_research",
		Scope:                   "pesquisa de fontes oficiais",
		Status:                  "completed",
		UsagePolicy:             "direct_write",
		Summary:                 "saida curta",
		Evidence:                []string{},
		Risks:                   []string{},
		IntegrationDecision:     "",
		RepoWriteAllowed:        true,
		CodexValidationRequired: false,
		ClosedBeforeCheckpoint:  false,
		RecordedAt:              "2026-06-09T21:00:00-03:00",
	}

	report := agentcontext.ValidateRecord(record)
	if report.Passed() {
		t.Fatal("unsafe agent context record passed, want failures")
	}
	for _, code := range []string{
		"agent_context_usage_policy_invalid",
		"agent_context_repo_write_allowed",
		"agent_context_validation_not_required",
		"agent_context_not_closed_before_checkpoint",
		"agent_context_missing_integration_decision",
		"agent_context_missing_evidence",
		"agent_context_missing_risk",
	} {
		if !report.HasIssue(code) {
			t.Fatalf("missing issue %q in %v", code, report.Codes())
		}
	}
}

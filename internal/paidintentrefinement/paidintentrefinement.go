package paidintentrefinement

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"portaljuridico/internal/batchdraftarchive"
	"portaljuridico/internal/batchdrafts"
	"portaljuridico/internal/batchfinaldrafts"
	"portaljuridico/internal/humanscore"
	"portaljuridico/internal/paidintent"
)

const (
	RefinementStatusReadyBlocked = "paid_intent_refinement_ready_blocked"
	RefinementPath               = "data/editorial/batch_paid_intent_refinements.jsonl"
)

type Record struct {
	RefinementID             string `json:"refinement_id"`
	UniqueIntentID           string `json:"unique_intent_id"`
	BatchID                  string `json:"batch_id"`
	GateScope                string `json:"gate_scope"`
	SourceGateID             string `json:"source_gate_id"`
	Term                     string `json:"term"`
	CandidatePath            string `json:"candidate_path"`
	ChangedField             string `json:"changed_field"`
	RefinementStrategy       string `json:"refinement_strategy"`
	BodyPaidSignal           string `json:"body_paid_signal"`
	OriginalPaidIntentStatus string `json:"original_paid_intent_status"`
	AfterPaidIntentStatus    string `json:"after_paid_intent_status"`
	BeforePaidBusinessScore  int    `json:"before_paid_business_score"`
	AfterPaidBusinessScore   int    `json:"after_paid_business_score"`
	AfterHumanScore          int    `json:"after_human_score"`
	AfterAILikeScore         int    `json:"after_ai_like_score"`
	IndexPolicy              string `json:"index_policy"`
	RenderAllowed            bool   `json:"render_allowed"`
	SitemapAllowed           bool   `json:"sitemap_allowed"`
	PublicationAllowed       bool   `json:"publication_allowed"`
	PublicPath               string `json:"public_path"`
	CheckedAt                string `json:"checked_at"`
	RefinementStatus         string `json:"refinement_status"`
}

type Entry struct {
	Line   int
	Record Record
}

type Plan struct {
	Records []Record
}

type Result struct {
	Refinements    []Record
	ArchiveRefined int
	FinalRefined   int
}

type Issue struct {
	Code    string
	Message string
}

type Report struct {
	Issues []Issue
}

type refinementText struct {
	Signal   string
	Strategy string
	Sentence string
}

func BuildPlan(root string) (Plan, Report) {
	gates, gateReport := paidintent.LoadRecords(root)
	finalEntries, finalReport := batchfinaldrafts.LoadRecords(root)
	archiveEntries, archiveReport := batchdraftarchive.LoadRecords(root)
	issues := append(convertPaidIssues(gateReport), convertFinalIssues(finalReport)...)
	issues = append(issues, convertArchiveIssues(archiveReport)...)
	if len(issues) > 0 {
		return Plan{}, Report{Issues: issues}
	}

	finalByIntent := make(map[string]batchfinaldrafts.Record)
	for _, entry := range finalEntries {
		finalByIntent[entry.Record.UniqueIntentID] = entry.Record
	}
	archiveByIntent := make(map[string]batchdrafts.Record)
	for _, entry := range archiveEntries {
		archiveByIntent[entry.Record.UniqueIntentID] = entry.Record
	}

	records := make([]Record, 0)
	for _, entry := range gates {
		gate := entry.Record
		if !needsRefinement(gate.PaidIntentStatus) {
			continue
		}
		var record Record
		var report Report
		switch gate.GateScope {
		case paidintent.FinalDraftGateScope:
			draft, ok := finalByIntent[gate.UniqueIntentID]
			if !ok {
				issues = append(issues, Issue{Code: "paid_intent_refinement_missing_final_draft", Message: gate.UniqueIntentID})
				continue
			}
			record, report = planFinalDraft(gate, draft)
		case paidintent.ExpansionReadinessGateScope:
			draft, ok := archiveByIntent[gate.UniqueIntentID]
			if !ok {
				issues = append(issues, Issue{Code: "paid_intent_refinement_missing_archive_draft", Message: gate.UniqueIntentID})
				continue
			}
			record, report = planArchiveDraft(gate, draft)
		default:
			issues = append(issues, Issue{Code: "paid_intent_refinement_invalid_scope", Message: gate.UniqueIntentID + ":" + gate.GateScope})
			continue
		}
		issues = append(issues, report.Issues...)
		if report.Passed() {
			records = append(records, record)
		}
	}
	if len(issues) > 0 {
		return Plan{Records: records}, Report{Issues: issues}
	}
	sort.Slice(records, func(left int, right int) bool {
		if records[left].BatchID != records[right].BatchID {
			return records[left].BatchID < records[right].BatchID
		}
		return records[left].UniqueIntentID < records[right].UniqueIntentID
	})
	return Plan{Records: records}, Report{}
}

func RefineRepository(root string) (Result, Report) {
	gates, gateReport := paidintent.LoadRecords(root)
	finalEntries, finalReport := batchfinaldrafts.LoadRecords(root)
	archiveEntries, archiveReport := batchdraftarchive.LoadRecords(root)
	issues := append(convertPaidIssues(gateReport), convertFinalIssues(finalReport)...)
	issues = append(issues, convertArchiveIssues(archiveReport)...)
	if len(issues) > 0 {
		return Result{}, Report{Issues: issues}
	}

	finalIndexByIntent := make(map[string]int)
	finalIntentIDs := make(map[string]bool)
	for index, entry := range finalEntries {
		finalIndexByIntent[entry.Record.UniqueIntentID] = index
		finalIntentIDs[entry.Record.UniqueIntentID] = true
	}
	expansionGateByIntent := make(map[string]paidintent.Record)
	for _, entry := range gates {
		gate := entry.Record
		if gate.GateScope == paidintent.ExpansionReadinessGateScope {
			expansionGateByIntent[gate.UniqueIntentID] = gate
		}
	}

	result := Result{Refinements: make([]Record, 0)}
	for index := range archiveEntries {
		draft := archiveEntries[index].Record
		gate := expansionGateByIntent[draft.UniqueIntentID]
		if gate.UniqueIntentID == "" {
			gate = archiveGateFromDraft(draft)
		}
		refined, record, changed, shouldRecord, report := refreshArchiveDraft(gate, draft)
		issues = append(issues, report.Issues...)
		if report.Passed() && changed {
			archiveEntries[index].Record = refined
			result.ArchiveRefined++
			if shouldRecord && !finalIntentIDs[draft.UniqueIntentID] {
				result.Refinements = append(result.Refinements, record)
			}
		}
	}

	for _, entry := range gates {
		gate := entry.Record
		if gate.GateScope != paidintent.FinalDraftGateScope {
			continue
		}
		index, ok := finalIndexByIntent[gate.UniqueIntentID]
		if !ok {
			issues = append(issues, Issue{Code: "paid_intent_refinement_missing_final_draft", Message: gate.UniqueIntentID})
			continue
		}
		currentGate := paidintent.EvaluateDraft(finalEntries[index].Record)
		if !needsRefinement(currentGate.PaidIntentStatus) {
			continue
		}
		currentGate.SourceMatrixID = gate.SourceMatrixID
		refined, record, report := refineFinalDraft(currentGate, finalEntries[index].Record)
		issues = append(issues, report.Issues...)
		if report.Passed() {
			finalEntries[index].Record = refined
			result.Refinements = append(result.Refinements, record)
			result.FinalRefined++
		}
	}
	if len(issues) > 0 {
		return result, Report{Issues: issues}
	}
	if len(result.Refinements) == 0 && result.ArchiveRefined == 0 && result.FinalRefined == 0 {
		return result, Report{}
	}
	sort.Slice(result.Refinements, func(left int, right int) bool {
		if result.Refinements[left].BatchID != result.Refinements[right].BatchID {
			return result.Refinements[left].BatchID < result.Refinements[right].BatchID
		}
		return result.Refinements[left].UniqueIntentID < result.Refinements[right].UniqueIntentID
	})
	if result.ArchiveRefined > 0 {
		if err := writeArchiveEntries(root, archiveEntries); err != nil {
			return result, Report{Issues: []Issue{{Code: "paid_intent_refinement_write_archive_failed", Message: err.Error()}}}
		}
	}
	if result.FinalRefined > 0 {
		if err := writeFinalEntries(root, finalEntries); err != nil {
			return result, Report{Issues: []Issue{{Code: "paid_intent_refinement_write_final_failed", Message: err.Error()}}}
		}
	}
	if len(result.Refinements) > 0 {
		records, mergeReport := mergeExistingRecords(root, result.Refinements)
		if !mergeReport.Passed() {
			return result, Report{Issues: mergeReport.Issues}
		}
		if err := WriteRecords(root, records); err != nil {
			return result, Report{Issues: []Issue{{Code: "paid_intent_refinement_write_records_failed", Message: err.Error()}}}
		}
	}
	return result, Report{}
}

func LoadRecords(root string) ([]Entry, Report) {
	projectRoot, err := findProjectRoot(root)
	if err != nil {
		return nil, Report{Issues: []Issue{{Code: "project_root_not_found", Message: err.Error()}}}
	}
	path := filepath.Join(projectRoot, RefinementPath)
	file, err := os.Open(path)
	if err != nil {
		return nil, Report{Issues: []Issue{{Code: "paid_intent_refinements_missing", Message: err.Error()}}}
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
			issues = append(issues, Issue{Code: "paid_intent_refinement_invalid_json", Message: fmt.Sprintf("line=%d %s", lineNumber, err.Error())})
			continue
		}
		entries = append(entries, Entry{Line: lineNumber, Record: record})
	}
	if err := scanner.Err(); err != nil {
		issues = append(issues, Issue{Code: "paid_intent_refinement_scan_failed", Message: err.Error()})
	}
	return entries, Report{Issues: issues}
}

func Validate(root string) Report {
	entries, report := LoadRecords(root)
	if !report.Passed() {
		return report
	}
	issues := make([]Issue, 0)
	if len(entries) == 0 {
		issues = append(issues, Issue{Code: "paid_intent_refinements_empty", Message: RefinementPath})
	}
	seen := make(map[string]int)
	for _, entry := range entries {
		recordReport := ValidateRecord(entry.Record)
		for _, issue := range recordReport.Issues {
			issue.Message = fmt.Sprintf("line=%d %s", entry.Line, issue.Message)
			issues = append(issues, issue)
		}
		if previous := seen[entry.Record.UniqueIntentID]; previous > 0 {
			issues = append(issues, Issue{Code: "paid_intent_refinement_duplicate_intent", Message: fmt.Sprintf("line=%d previous_line=%d id=%s", entry.Line, previous, entry.Record.UniqueIntentID)})
		}
		seen[entry.Record.UniqueIntentID] = entry.Line
	}
	return Report{Issues: issues}
}

func ValidateRecord(record Record) Report {
	issues := make([]Issue, 0)
	if record.RefinementID == "" || record.UniqueIntentID == "" || record.BatchID == "" || record.SourceGateID == "" || record.Term == "" {
		issues = append(issues, Issue{Code: "paid_intent_refinement_missing_identity", Message: record.UniqueIntentID})
	}
	if record.GateScope != paidintent.FinalDraftGateScope && record.GateScope != paidintent.ExpansionReadinessGateScope {
		issues = append(issues, Issue{Code: "paid_intent_refinement_invalid_scope", Message: record.GateScope})
	}
	if !needsRefinement(record.OriginalPaidIntentStatus) {
		issues = append(issues, Issue{Code: "paid_intent_refinement_invalid_original_status", Message: record.OriginalPaidIntentStatus})
	}
	if record.AfterPaidIntentStatus != paidintent.PassedBlockedStatus {
		issues = append(issues, Issue{Code: "paid_intent_refinement_not_paid_ready", Message: record.AfterPaidIntentStatus})
	}
	if record.AfterPaidBusinessScore < 4 {
		issues = append(issues, Issue{Code: "paid_intent_refinement_low_business_score", Message: fmt.Sprintf("%s=%d", record.UniqueIntentID, record.AfterPaidBusinessScore)})
	}
	if record.AfterHumanScore < 85 || record.AfterAILikeScore > 20 {
		issues = append(issues, Issue{Code: "paid_intent_refinement_failed_human_score", Message: fmt.Sprintf("%s=%d/%d", record.UniqueIntentID, record.AfterHumanScore, record.AfterAILikeScore)})
	}
	if record.BodyPaidSignal == "" || record.RefinementStrategy == "" || record.ChangedField == "" {
		issues = append(issues, Issue{Code: "paid_intent_refinement_missing_strategy", Message: record.UniqueIntentID})
	}
	if record.IndexPolicy != "noindex" {
		issues = append(issues, Issue{Code: "paid_intent_refinement_invalid_index_policy", Message: record.IndexPolicy})
	}
	if record.RenderAllowed || record.SitemapAllowed || record.PublicationAllowed || record.PublicPath != "" {
		issues = append(issues, Issue{Code: "paid_intent_refinement_public_flag", Message: record.UniqueIntentID})
	}
	if record.CheckedAt == "" {
		issues = append(issues, Issue{Code: "paid_intent_refinement_without_checked_at", Message: record.UniqueIntentID})
	}
	if record.RefinementStatus != RefinementStatusReadyBlocked {
		issues = append(issues, Issue{Code: "paid_intent_refinement_invalid_status", Message: record.RefinementStatus})
	}
	return Report{Issues: issues}
}

func planArchiveDraft(gate paidintent.Record, draft batchdrafts.Record) (Record, Report) {
	_, record, report := refineArchiveDraft(gate, draft)
	return record, report
}

func planFinalDraft(gate paidintent.Record, draft batchfinaldrafts.Record) (Record, Report) {
	_, record, report := refineFinalDraft(gate, draft)
	return record, report
}

func refineArchiveDraft(gate paidintent.Record, draft batchdrafts.Record) (batchdrafts.Record, Record, Report) {
	var lastRecord Record
	var lastReport Report
	var lastDraft batchdrafts.Record
	for _, refinement := range orderedRefinements(draft.UniqueIntentID, draft.LegalArea) {
		refined := draft
		refined.DigitalAction = appendSentence(refined.DigitalAction, refinement.Sentence)
		score := humanscore.ScoreText(refined.FullText())
		refined.HumanScore = score.HumanScore
		refined.AILikeScore = score.AILikeScore
		evaluated := paidintent.EvaluateArchiveDraft(refined)
		record := buildRecord(gate, "digital_action", refinement, evaluated, score)
		report := validatePlanResult(gate, evaluated, score)
		if report.Passed() {
			return refined, record, report
		}
		lastDraft = refined
		lastRecord = record
		lastReport = report
	}
	return lastDraft, lastRecord, lastReport
}

func refreshArchiveDraft(gate paidintent.Record, draft batchdrafts.Record) (batchdrafts.Record, Record, bool, bool, Report) {
	cleaned := draft
	cleaned.DigitalAction = removeKnownRefinementSentences(cleaned.DigitalAction)
	score := humanscore.ScoreText(cleaned.FullText())
	cleaned.HumanScore = score.HumanScore
	cleaned.AILikeScore = score.AILikeScore
	currentGate := paidintent.EvaluateArchiveDraft(cleaned)
	if gate.CheckedAt != "" {
		currentGate.CheckedAt = gate.CheckedAt
	}
	if !needsRefinement(currentGate.PaidIntentStatus) {
		return cleaned, Record{}, archiveDraftChanged(draft, cleaned), false, Report{}
	}
	refined, record, report := refineArchiveDraft(currentGate, cleaned)
	return refined, record, archiveDraftChanged(draft, refined), true, report
}

func archiveGateFromDraft(draft batchdrafts.Record) paidintent.Record {
	return paidintent.EvaluateArchiveDraft(draft)
}

func refineFinalDraft(gate paidintent.Record, draft batchfinaldrafts.Record) (batchfinaldrafts.Record, Record, Report) {
	var lastRecord Record
	var lastReport Report
	var lastDraft batchfinaldrafts.Record
	for _, refinement := range orderedRefinements(draft.UniqueIntentID, "") {
		refined := draft
		refined.DigitalTriage = appendSentence(refined.DigitalTriage, refinement.Sentence)
		score := humanscore.ScoreText(refined.FullText())
		refined.HumanScore = score.HumanScore
		refined.AILikeScore = score.AILikeScore
		evaluated := paidintent.EvaluateDraft(refined)
		record := buildRecord(gate, "digital_triage", refinement, evaluated, score)
		report := validatePlanResult(gate, evaluated, score)
		if report.Passed() {
			return refined, record, report
		}
		lastDraft = refined
		lastRecord = record
		lastReport = report
	}
	return lastDraft, lastRecord, lastReport
}

func buildRecord(gate paidintent.Record, changedField string, refinement refinementText, evaluated paidintent.Record, score humanscore.Score) Record {
	return Record{
		RefinementID:             "paid-intent-refinement-" + gate.UniqueIntentID,
		UniqueIntentID:           gate.UniqueIntentID,
		BatchID:                  gate.BatchID,
		GateScope:                gate.GateScope,
		SourceGateID:             gate.PaidIntentGateID,
		Term:                     gate.Term,
		CandidatePath:            gate.CandidatePath,
		ChangedField:             changedField,
		RefinementStrategy:       refinement.Strategy,
		BodyPaidSignal:           refinement.Signal,
		OriginalPaidIntentStatus: gate.PaidIntentStatus,
		AfterPaidIntentStatus:    evaluated.PaidIntentStatus,
		BeforePaidBusinessScore:  gate.PaidBusinessScore,
		AfterPaidBusinessScore:   evaluated.PaidBusinessScore,
		AfterHumanScore:          score.HumanScore,
		AfterAILikeScore:         score.AILikeScore,
		IndexPolicy:              "noindex",
		RenderAllowed:            false,
		SitemapAllowed:           false,
		PublicationAllowed:       false,
		PublicPath:               "",
		CheckedAt:                gate.CheckedAt,
		RefinementStatus:         RefinementStatusReadyBlocked,
	}
}

func validatePlanResult(gate paidintent.Record, evaluated paidintent.Record, score humanscore.Score) Report {
	issues := make([]Issue, 0)
	if evaluated.PaidIntentStatus != paidintent.PassedBlockedStatus {
		issues = append(issues, Issue{Code: "paid_intent_refinement_after_status_not_passed", Message: gate.UniqueIntentID + ":" + evaluated.PaidIntentStatus})
	}
	if score.HumanScore < 85 || score.AILikeScore > 20 || len(score.BlockingIssues) > 0 {
		issues = append(issues, Issue{Code: "paid_intent_refinement_after_human_score_failed", Message: gate.UniqueIntentID + ":" + strings.Join(score.Codes(), ",")})
	}
	return Report{Issues: issues}
}

func needsRefinement(status string) bool {
	return status == paidintent.MissingPaidSignalStatus || status == paidintent.CTAOnlyBlockedStatus || status == paidintent.LowBusinessScoreStatus
}

func appendSentence(value string, sentence string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return sentence
	}
	if strings.HasSuffix(value, ".") {
		return value + " " + sentence
	}
	return value + ". " + sentence
}

func removeKnownRefinementSentences(value string) string {
	cleaned := strings.TrimSpace(value)
	for _, sentence := range knownRefinementSentences() {
		cleaned = strings.ReplaceAll(cleaned, sentence, "")
	}
	cleaned = strings.Join(strings.Fields(cleaned), " ")
	for strings.Contains(cleaned, ". .") {
		cleaned = strings.ReplaceAll(cleaned, ". .", ".")
	}
	cleaned = strings.TrimSpace(cleaned)
	cleaned = strings.TrimPrefix(cleaned, ". ")
	cleaned = strings.TrimPrefix(cleaned, ".")
	return strings.TrimSpace(cleaned)
}

func archiveDraftChanged(left batchdrafts.Record, right batchdrafts.Record) bool {
	return left.DigitalAction != right.DigitalAction || left.HumanScore != right.HumanScore || left.AILikeScore != right.AILikeScore
}

func orderedRefinements(intentID string, legalArea string) []refinementText {
	templates := refinementTemplates()
	start := stableIndex(intentID+"-"+legalArea, len(templates))
	ordered := make([]refinementText, 0, len(templates))
	preferred := preferredRefinementStrategy(intentID)
	if preferred != "" {
		for _, template := range templates {
			if template.Strategy == preferred {
				ordered = append(ordered, template)
				break
			}
		}
	}
	for offset := 0; offset < len(templates); offset++ {
		template := templates[(start+offset)%len(templates)]
		if template.Strategy == preferred {
			continue
		}
		ordered = append(ordered, template)
	}
	return ordered
}

func preferredRefinementStrategy(intentID string) string {
	preferences := []struct {
		Token    string
		Strategy string
	}{
		{"fonte-primaria", "body_private_hiring_source_boundary"},
		{"linha-do-tempo", "body_private_lawyer_timeline"},
		{"documento-minimo", "body_fee_quote_document_boundary"},
		{"prova-de-negativa", "body_fee_scope_negative_proof"},
		{"parte-responsavel", "body_private_client_evidence_value"},
		{"risco-economico", "body_paid_service_scope"},
		{"competencia-digital", "body_private_hiring_source_boundary"},
		{"estado-do-processo", "body_paid_consultation_decision_point"},
		{"prova-medica-ou-tecnica", "body_paid_document_review"},
		{"conflito-de-versoes", "body_paid_triage_risk_window"},
		{"custo-de-inercia", "body_paid_service_scope"},
		{"evidencia-de-boa-fe", "body_private_client_evidence_value"},
		{"vulnerabilidade", "body_paid_consultation_scope"},
		{"negociacao-previa", "body_private_service_budget"},
		{"prazo-e-urgencia", "body_paid_triage_risk_window"},
		{"prova-digital", "body_hiring_intent_documents"},
	}
	for _, preference := range preferences {
		if strings.Contains(intentID, preference.Token) {
			return preference.Strategy
		}
	}
	return ""
}

func refinementTemplates() []refinementText {
	return []refinementText{
		{
			Signal:   "contratar advogado",
			Strategy: "body_hiring_intent_documents",
			Sentence: "O roteiro identifica quando o leitor quer contratar advogado particular online, revisar documentos e receber analise paga antes de qualquer medida.",
		},
		{
			Signal:   "atendimento particular",
			Strategy: "body_private_service_budget",
			Sentence: "Para atendimento particular declarado, a triagem separa contrato, valor envolvido e duvidas para orcamento de honorarios antes de avancar.",
		},
		{
			Signal:   "analise paga",
			Strategy: "body_paid_document_review",
			Sentence: "Cliente particular que busca analise paga precisa reunir comprovante, protocolo e resposta formal para definir custo e escopo.",
		},
		{
			Signal:   "servico juridico pago",
			Strategy: "body_paid_service_scope",
			Sentence: "Se houver interesse em servico juridico pago, a etapa digital organiza valor da causa, urgencia e prova minima antes da conversa.",
		},
		{
			Signal:   "consulta paga",
			Strategy: "body_paid_consultation_scope",
			Sentence: "Consulta paga, quando cabivel, fica limitada a documentos, riscos e viabilidade do caso, com escopo combinado previamente.",
		},
		{
			Signal:   "honorarios",
			Strategy: "body_fee_scope_negative_proof",
			Sentence: "A leitura de honorarios depende de negativa formal, comprovante de envio e documentos que mostrem responsabilidade antes da analise.",
		},
		{
			Signal:   "advogado particular",
			Strategy: "body_private_lawyer_timeline",
			Sentence: "Quem procura advogado particular precisa organizar data do fato, protocolo, documentos e valor pago para uma triagem objetiva.",
		},
		{
			Signal:   "triagem paga",
			Strategy: "body_paid_triage_risk_window",
			Sentence: "Triagem paga so faz sentido quando ha risco concreto, prazo identificado, documento principal e pergunta juridica delimitada.",
		},
		{
			Signal:   "orcamento de honorarios",
			Strategy: "body_fee_quote_document_boundary",
			Sentence: "Orcamento de honorarios exige fronteira clara entre documento conferido, providencia esperada e etapa digital possivel.",
		},
		{
			Signal:   "cliente particular",
			Strategy: "body_private_client_evidence_value",
			Sentence: "Cliente particular deve informar valor envolvido, comprovantes, tentativa anterior e responsavel indicado no documento.",
		},
		{
			Signal:   "contratacao particular",
			Strategy: "body_private_hiring_source_boundary",
			Sentence: "Contratacao particular exige fonte especifica, contrato ou protocolo e prova documental suficiente para estimar trabalho juridico.",
		},
		{
			Signal:   "consulta paga",
			Strategy: "body_paid_consultation_decision_point",
			Sentence: "Antes de consulta paga, o material separa decisao recebida, contrato, recibo e duvida que precisa de leitura tecnica.",
		},
	}
}

func knownRefinementSentences() []string {
	known := make([]string, 0)
	for _, refinement := range refinementTemplates() {
		known = append(known, refinement.Sentence)
	}
	known = append(known,
		"O roteiro tambem verifica se a pessoa quer contratar advogado particular online para revisar documentos e receber analise paga antes de qualquer medida.",
		"Para atendimento particular declarado, a triagem separa documentos, valor envolvido e duvidas para orcamento de honorarios antes de avancar.",
		"Cliente particular que busca analise paga dos documentos precisa ter custo e escopo definidos antes da conversa juridica.",
		"Se houver interesse em servico juridico pago, a etapa digital organiza valor discutido, urgencia e prova minima para evitar atendimento sem escopo.",
		"Consulta paga, quando cabivel, deve ficar limitada a documentos, riscos e viabilidade do caso, com escopo combinado previamente.",
		"Contratacao particular exige fonte especifica, limite de atuacao online e prova documental suficiente para estimar trabalho juridico.",
	)
	return known
}

func stableIndex(value string, length int) int {
	if length == 0 {
		return 0
	}
	hash := 0
	for _, r := range value {
		hash = (hash*33 + int(r)) & 0x7fffffff
	}
	return hash % length
}

func convertPaidIssues(report paidintent.Report) []Issue {
	issues := make([]Issue, 0, len(report.Issues))
	for _, issue := range report.Issues {
		issues = append(issues, Issue{Code: "paid_intent_refinement_paid_" + issue.Code, Message: issue.Message})
	}
	return issues
}

func convertFinalIssues(report batchfinaldrafts.Report) []Issue {
	issues := make([]Issue, 0, len(report.Issues))
	for _, issue := range report.Issues {
		issues = append(issues, Issue{Code: "paid_intent_refinement_final_" + issue.Code, Message: issue.Message})
	}
	return issues
}

func convertArchiveIssues(report batchdraftarchive.Report) []Issue {
	issues := make([]Issue, 0, len(report.Issues))
	for _, issue := range report.Issues {
		issues = append(issues, Issue{Code: "paid_intent_refinement_archive_" + issue.Code, Message: issue.Message})
	}
	return issues
}

func mergeExistingRecords(root string, updates []Record) ([]Record, Report) {
	entries, loadReport := LoadRecords(root)
	if !loadReport.Passed() {
		return updates, loadReport
	}
	byIntent := make(map[string]Record)
	for _, entry := range entries {
		byIntent[entry.Record.UniqueIntentID] = entry.Record
	}
	for _, update := range updates {
		byIntent[update.UniqueIntentID] = update
	}
	merged := make([]Record, 0, len(byIntent))
	for _, record := range byIntent {
		merged = append(merged, record)
	}
	sort.Slice(merged, func(left int, right int) bool {
		if merged[left].BatchID != merged[right].BatchID {
			return merged[left].BatchID < merged[right].BatchID
		}
		if merged[left].GateScope != merged[right].GateScope {
			return merged[left].GateScope < merged[right].GateScope
		}
		return merged[left].UniqueIntentID < merged[right].UniqueIntentID
	})
	return merged, Report{}
}

func WriteRecords(root string, records []Record) error {
	projectRoot, err := findProjectRoot(root)
	if err != nil {
		return err
	}
	path := filepath.Join(projectRoot, RefinementPath)
	return writeJSONLines(path, records, func(record Record) []string {
		return ValidateRecord(record).Messages()
	})
}

func writeArchiveEntries(root string, entries []batchdrafts.Entry) error {
	projectRoot, err := findProjectRoot(root)
	if err != nil {
		return err
	}
	records := make([]batchdrafts.Record, 0, len(entries))
	for _, entry := range entries {
		records = append(records, entry.Record)
	}
	path := filepath.Join(projectRoot, "data", "editorial", "batch_draft_expansion_archive.jsonl")
	return writeJSONLines(path, records, func(record batchdrafts.Record) []string {
		return batchdrafts.ValidateRecord(record).Messages()
	})
}

func writeFinalEntries(root string, entries []batchfinaldrafts.Entry) error {
	projectRoot, err := findProjectRoot(root)
	if err != nil {
		return err
	}
	manifestIndex, manifestReport := batchfinaldrafts.BuildManifestIndex(root)
	if !manifestReport.Passed() {
		return fmt.Errorf("final_manifest_index_failed=%s", strings.Join(manifestReport.Messages(), " | "))
	}
	records := make([]batchfinaldrafts.Record, 0, len(entries))
	for _, entry := range entries {
		records = append(records, entry.Record)
	}
	path := filepath.Join(projectRoot, "data", "editorial", "batch_final_authorial_drafts.jsonl")
	return writeJSONLines(path, records, func(record batchfinaldrafts.Record) []string {
		return batchfinaldrafts.ValidateRecordAgainstManifestIndex(record, manifestIndex).Messages()
	})
}

func writeJSONLines[T any](path string, records []T, validate func(T) []string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()
	for _, record := range records {
		if messages := validate(record); len(messages) > 0 {
			return fmt.Errorf("invalid_record=%s", strings.Join(messages, " | "))
		}
		data, err := json.Marshal(record)
		if err != nil {
			return err
		}
		if _, err := file.Write(append(data, '\n')); err != nil {
			return err
		}
	}
	return nil
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

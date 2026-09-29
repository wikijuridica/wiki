package paidintent

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode"

	"portaljuridico/internal/batchdraftarchive"
	"portaljuridico/internal/batchdrafts"
	"portaljuridico/internal/batchfinaldrafts"
)

type Mode string

const RequirePaidSignal Mode = "require_paid_signal"

const (
	minimumPaidBusinessScore          = 4
	FinalDraftGateScope               = "final_draft"
	ExpansionReadinessGateScope       = "expansion_readiness"
	PassedBlockedStatus               = "paid_intent_passed_blocked_publication"
	PublicAssistanceBlockedStatus     = "paid_intent_blocked_public_assistance_free_risk"
	AdminSelfServiceBlockedStatus     = "paid_intent_blocked_admin_self_service_risk"
	FreeServiceBlockedStatus          = "paid_intent_blocked_free_service_signal"
	ResearchOnlyBlockedStatus         = "paid_intent_blocked_research_only_signal"
	CTAOnlyBlockedStatus              = "paid_intent_blocked_cta_only_paid_signal"
	MissingPaidSignalStatus           = "paid_intent_blocked_missing_paid_signal"
	LowBusinessScoreStatus            = "paid_intent_blocked_low_business_score"
	PrevidenciarioInformationalStatus = "paid_intent_flexible_previdenciario_informational_blocked_publication"
)

type Record struct {
	PaidIntentGateID   string   `json:"paid_intent_gate_id"`
	GateScope          string   `json:"gate_scope"`
	DraftID            string   `json:"draft_id"`
	UniqueIntentID     string   `json:"unique_intent_id"`
	BatchID            string   `json:"batch_id"`
	SourceMatrixID     string   `json:"source_matrix_id"`
	Term               string   `json:"term"`
	CandidatePath      string   `json:"candidate_path"`
	PaidIntentStatus   string   `json:"paid_intent_status"`
	PaidBusinessScore  int      `json:"paid_business_score"`
	CorePaidScore      int      `json:"core_paid_business_score"`
	CTAPaidScore       int      `json:"cta_paid_business_score"`
	PaidSignals        []string `json:"paid_signals"`
	CTAPaidSignals     []string `json:"cta_paid_signals"`
	BusinessSignals    []string `json:"business_signals"`
	RiskSignals        []string `json:"risk_signals"`
	RoutingDecision    string   `json:"routing_decision"`
	IndexPolicy        string   `json:"index_policy"`
	RenderAllowed      bool     `json:"render_allowed"`
	SitemapAllowed     bool     `json:"sitemap_allowed"`
	PublicationAllowed bool     `json:"publication_allowed"`
	PublicPath         string   `json:"public_path"`
	CheckedAt          string   `json:"checked_at"`
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

type readinessRecord struct {
	BatchID                     string   `json:"batch_id"`
	TargetCandidateCount        int      `json:"target_candidate_count"`
	ExpansionCandidateSelector  string   `json:"expansion_candidate_selector"`
	ExpansionCandidateIntentIDs []string `json:"expansion_candidate_intent_ids"`
}

type textScore struct {
	paidSignals             []string
	businessSignals         []string
	freeSignals             []string
	researchSignals         []string
	publicAssistanceSignals []string
	adminSelfServiceSignals []string
	paidBusinessScore       int
}

var paidServiceSignals = []string{
	"contratacao particular",
	"contratar advogado",
	"contratar advogada",
	"advogado particular",
	"advogada particular",
	"atendimento particular",
	"consulta paga",
	"triagem paga",
	"analise paga",
	"servico juridico pago",
	"honorarios",
	"orcamento de honorarios",
	"cliente particular",
}

var businessReadinessSignals = []string{
	"valor envolvido",
	"valor da causa",
	"valor pago",
	"prejuizo",
	"patrimonio",
	"empresa",
	"contrato",
	"plano de saude",
	"banco",
	"inss",
	"beneficio negado",
	"negativa formal",
	"protocolo",
	"documentos",
	"comprovante",
	"nota fiscal",
	"recibo",
	"orcamento",
	"risco clinico",
	"urgente",
	"liminar",
}

var publicAssistanceRiskSignals = []string{
	"bpc loas",
	"cadunico",
	"beneficio assistencial",
	"renda familiar",
	"baixa renda",
	"miserabilidade",
	"defensoria publica",
	"justica gratuita",
}

var adminSelfServiceRiskSignals = []string{
	"cumprimento de exigencia",
	"anexar documentos",
	"reenviar arquivos",
	"autoatendimento",
}

var freeServiceSignals = []string{
	"advogado gratuito",
	"advogada gratuita",
	"consulta gratis",
	"consulta gratuita",
	"servico gratuito",
	"atendimento gratuito",
	"juridico gratuito",
	"defensoria publica",
	"justica gratuita",
	"sem pagar",
	"nao posso pagar",
	"de graca",
	"pro bono",
	"gratuidade",
}

var researchOnlySignals = []string{
	"apenas curiosidade",
	"so pesquisando",
	"somente pesquisando",
	"para estudar",
	"trabalho de faculdade",
	"tcc",
	"resumo para prova",
	"modelo pronto",
	"modelo de peticao",
	"peticao pronta",
	"pdf gratis",
	"jurisprudencia para estudo",
	"significado juridico",
}

func Validate(root string) Report {
	gates, loadReport := LoadRecords(root)
	issues := append([]Issue{}, loadReport.Issues...)
	if !loadReport.Passed() {
		return Report{Issues: issues}
	}
	drafts, draftReport := batchfinaldrafts.LoadRecords(root)
	issues = append(issues, convertDraftIssues(draftReport)...)
	archiveEntries, archiveReport := batchdraftarchive.LoadRecords(root)
	issues = append(issues, convertArchiveIssues(archiveReport)...)
	readinessTargets, readinessReport := loadExpansionReadinessTargets(root)
	issues = append(issues, readinessReport.Issues...)
	if len(gates) == 0 && loadReport.Passed() {
		issues = append(issues, Issue{Code: "paid_intent_gates_empty", Message: "data/editorial/batch_paid_intent_gates.jsonl"})
	}
	draftByID := make(map[string]batchfinaldrafts.Record)
	finalIntentIDs := make(map[string]bool)
	for _, entry := range drafts {
		draftByID[entry.Record.DraftID] = entry.Record
		finalIntentIDs[entry.Record.UniqueIntentID] = true
	}
	archiveByIntent := make(map[string]batchdrafts.Record)
	for _, entry := range archiveEntries {
		archiveByIntent[entry.Record.UniqueIntentID] = entry.Record
	}
	seen := make(map[string]int)
	for _, entry := range gates {
		report := ValidateRecord(entry.Record, draftByID, archiveByIntent)
		for _, issue := range report.Issues {
			issue.Message = fmt.Sprintf("line=%d %s", entry.Line, issue.Message)
			issues = append(issues, issue)
		}
		if previous := seen[entry.Record.UniqueIntentID]; previous > 0 {
			issues = append(issues, Issue{Code: "paid_intent_duplicate_intent", Message: fmt.Sprintf("line=%d previous_line=%d id=%s", entry.Line, previous, entry.Record.UniqueIntentID)})
		}
		seen[entry.Record.UniqueIntentID] = entry.Line
	}
	for intentID := range finalIntentIDs {
		if seen[intentID] == 0 {
			issues = append(issues, Issue{Code: "paid_intent_missing_final_draft_gate", Message: intentID})
		}
	}
	for intentID := range readinessTargets {
		if seen[intentID] == 0 {
			issues = append(issues, Issue{Code: "paid_intent_missing_expansion_gate", Message: intentID})
		}
	}
	return Report{Issues: issues}
}

func LoadRecords(root string) ([]Entry, Report) {
	projectRoot, err := findProjectRoot(root)
	if err != nil {
		return nil, Report{Issues: []Issue{{Code: "project_root_not_found", Message: err.Error()}}}
	}
	path := filepath.Join(projectRoot, "data", "editorial", "batch_paid_intent_gates.jsonl")
	file, err := os.Open(path)
	if err != nil {
		return nil, Report{Issues: []Issue{{Code: "paid_intent_gates_missing", Message: err.Error()}}}
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
			issues = append(issues, Issue{Code: "paid_intent_gate_invalid_json", Message: fmt.Sprintf("line=%d %s", lineNumber, err.Error())})
			continue
		}
		entries = append(entries, Entry{Line: lineNumber, Record: record})
	}
	if err := scanner.Err(); err != nil {
		issues = append(issues, Issue{Code: "paid_intent_gate_scan_failed", Message: err.Error()})
	}
	return entries, Report{Issues: issues}
}

func BuildRepositoryRecords(root string) ([]Record, Report) {
	drafts, draftReport := batchfinaldrafts.LoadRecords(root)
	archiveEntries, archiveReport := batchdraftarchive.LoadRecords(root)
	readinessTargets, readinessReport := loadExpansionReadinessTargets(root)
	issues := append(convertDraftIssues(draftReport), convertArchiveIssues(archiveReport)...)
	issues = append(issues, readinessReport.Issues...)
	if len(issues) > 0 {
		return nil, Report{Issues: issues}
	}

	archiveByIntent := make(map[string]batchdrafts.Record)
	for _, entry := range archiveEntries {
		archiveByIntent[entry.Record.UniqueIntentID] = entry.Record
	}

	recordsByIntent := make(map[string]Record)
	for _, entry := range drafts {
		record := EvaluateDraft(entry.Record)
		recordsByIntent[record.UniqueIntentID] = record
	}
	for intentID := range readinessTargets {
		if _, ok := recordsByIntent[intentID]; ok {
			continue
		}
		draft, ok := archiveByIntent[intentID]
		if !ok {
			issues = append(issues, Issue{Code: "paid_intent_build_missing_archive_draft", Message: intentID})
			continue
		}
		record := EvaluateArchiveDraft(draft)
		recordsByIntent[record.UniqueIntentID] = record
	}
	if len(issues) > 0 {
		return nil, Report{Issues: issues}
	}

	records := make([]Record, 0, len(recordsByIntent))
	for _, record := range recordsByIntent {
		records = append(records, record)
	}
	sort.Slice(records, func(left int, right int) bool {
		if records[left].BatchID != records[right].BatchID {
			return records[left].BatchID < records[right].BatchID
		}
		if records[left].GateScope != records[right].GateScope {
			return records[left].GateScope < records[right].GateScope
		}
		return records[left].UniqueIntentID < records[right].UniqueIntentID
	})
	return records, Report{}
}

func ValidateRecord(record Record, draftsByID map[string]batchfinaldrafts.Record, archiveByIntent map[string]batchdrafts.Record) Report {
	issues := make([]Issue, 0)
	if record.PaidIntentGateID == "" || record.GateScope == "" || record.UniqueIntentID == "" || record.BatchID == "" || record.SourceMatrixID == "" || record.Term == "" || record.CandidatePath == "" {
		issues = append(issues, Issue{Code: "paid_intent_gate_missing_identity", Message: record.PaidIntentGateID})
	}
	switch record.GateScope {
	case FinalDraftGateScope:
		if record.DraftID == "" {
			issues = append(issues, Issue{Code: "paid_intent_gate_missing_draft_id", Message: record.UniqueIntentID})
			break
		}
		draft, ok := draftsByID[record.DraftID]
		if !ok {
			issues = append(issues, Issue{Code: "paid_intent_gate_missing_draft", Message: record.DraftID})
			break
		}
		if record.UniqueIntentID != draft.UniqueIntentID || record.BatchID != draft.BatchID || record.SourceMatrixID != draft.SourceMatrixID || record.Term != draft.Term || record.CandidatePath != draft.CandidatePath {
			issues = append(issues, Issue{Code: "paid_intent_gate_draft_mismatch", Message: record.UniqueIntentID})
		}
		expected := EvaluateDraft(draft)
		issues = append(issues, compareGateToEvaluation(record, expected)...)
	case ExpansionReadinessGateScope:
		if record.DraftID != "" {
			issues = append(issues, Issue{Code: "paid_intent_expansion_gate_has_draft_id", Message: record.UniqueIntentID})
		}
		draft, ok := archiveByIntent[record.UniqueIntentID]
		if !ok {
			issues = append(issues, Issue{Code: "paid_intent_gate_missing_archive_draft", Message: record.UniqueIntentID})
			break
		}
		if record.BatchID != draft.BatchID || record.SourceMatrixID != draft.SourceMatrixID || record.Term != draft.Term || record.CandidatePath != candidatePathForIntent(draft.UniqueIntentID) {
			issues = append(issues, Issue{Code: "paid_intent_gate_archive_mismatch", Message: record.UniqueIntentID})
		}
		expected := EvaluateArchiveDraft(draft)
		issues = append(issues, compareGateToEvaluation(record, expected)...)
	default:
		issues = append(issues, Issue{Code: "paid_intent_gate_invalid_scope", Message: record.GateScope})
	}
	if !validStatus(record.PaidIntentStatus) {
		issues = append(issues, Issue{Code: "paid_intent_gate_invalid_status", Message: record.PaidIntentStatus})
	}
	if record.IndexPolicy != "noindex" {
		issues = append(issues, Issue{Code: "paid_intent_gate_invalid_index_policy", Message: record.IndexPolicy})
	}
	if record.RenderAllowed {
		issues = append(issues, Issue{Code: "paid_intent_gate_render_allowed", Message: record.UniqueIntentID})
	}
	if record.SitemapAllowed {
		issues = append(issues, Issue{Code: "paid_intent_gate_sitemap_allowed", Message: record.UniqueIntentID})
	}
	if record.PublicationAllowed {
		issues = append(issues, Issue{Code: "paid_intent_gate_publication_allowed", Message: record.UniqueIntentID})
	}
	if record.PublicPath != "" {
		issues = append(issues, Issue{Code: "paid_intent_gate_has_public_path", Message: record.PublicPath})
	}
	if record.CheckedAt == "" {
		issues = append(issues, Issue{Code: "paid_intent_gate_without_checked_at", Message: record.UniqueIntentID})
	}
	return Report{Issues: issues}
}

func EvaluateDraft(record batchfinaldrafts.Record) Record {
	core := strings.Join([]string{
		record.Term,
		record.CandidateTitle,
		record.CandidateMetaDescription,
		record.Opening,
		record.SourceUse,
		record.DocumentGuidance,
		record.DigitalTriage,
	}, " ")
	return evaluateParts(EvaluationInput{
		GateScope:      FinalDraftGateScope,
		DraftID:        record.DraftID,
		UniqueIntentID: record.UniqueIntentID,
		BatchID:        record.BatchID,
		SourceMatrixID: record.SourceMatrixID,
		Term:           record.Term,
		CandidatePath:  record.CandidatePath,
		CoreText:       core,
		CTAText:        record.CTAContextMessage,
		CheckedAt:      record.CheckedAt,
	})
}

func EvaluateArchiveDraft(record batchdrafts.Record) Record {
	core := strings.Join([]string{
		record.Term,
		record.ReaderProblem,
		record.SourceHook,
		record.DocumentContext,
		record.RiskContext,
		record.DigitalAction,
	}, " ")
	return evaluateParts(EvaluationInput{
		GateScope:      ExpansionReadinessGateScope,
		DraftID:        "",
		UniqueIntentID: record.UniqueIntentID,
		BatchID:        record.BatchID,
		SourceMatrixID: record.SourceMatrixID,
		Term:           record.Term,
		CandidatePath:  candidatePathForIntent(record.UniqueIntentID),
		CoreText:       core,
		CTAText:        record.CTAContext,
		CheckedAt:      record.CheckedAt,
	})
}

type EvaluationInput struct {
	GateScope      string
	DraftID        string
	UniqueIntentID string
	BatchID        string
	SourceMatrixID string
	Term           string
	CandidatePath  string
	CoreText       string
	CTAText        string
	CheckedAt      string
}

func evaluateParts(input EvaluationInput) Record {
	coreScore := scoreText(input.CoreText)
	ctaScore := scoreText(input.CTAText)
	combinedScore := scoreText(strings.TrimSpace(input.CoreText + " " + input.CTAText))
	status := PassedBlockedStatus
	routing := "paid_intent_candidate_blocked_publication"
	risks := collectRiskSignals(combinedScore)
	if len(combinedScore.publicAssistanceSignals) > 0 {
		status = PublicAssistanceBlockedStatus
		routing = "commercial_publication_blocked_public_assistance"
	} else if len(combinedScore.adminSelfServiceSignals) > 0 {
		status = AdminSelfServiceBlockedStatus
		routing = "commercial_publication_blocked_admin_self_service"
	} else if len(combinedScore.freeSignals) > 0 {
		status = FreeServiceBlockedStatus
		routing = "commercial_publication_blocked_free_service_signal"
	} else if len(combinedScore.researchSignals) > 0 {
		status = ResearchOnlyBlockedStatus
		routing = "commercial_publication_blocked_research_only_signal"
	} else if len(coreScore.paidSignals) == 0 && len(ctaScore.paidSignals) > 0 {
		status = CTAOnlyBlockedStatus
		routing = "commercial_publication_blocked_cta_only_paid_signal"
	} else if len(coreScore.paidSignals) == 0 {
		status = MissingPaidSignalStatus
		routing = "commercial_publication_blocked_missing_paid_signal"
	} else if coreScore.paidBusinessScore < minimumPaidBusinessScore {
		status = LowBusinessScoreStatus
		routing = "commercial_publication_blocked_low_business_score"
	}
	if isPrevidenciarioInformationalFlex(input, status) {
		status = PrevidenciarioInformationalStatus
		routing = "informational_previdenciario_blocked_publication"
	}
	return Record{
		PaidIntentGateID:   "paid-intent-" + input.UniqueIntentID,
		GateScope:          input.GateScope,
		DraftID:            input.DraftID,
		UniqueIntentID:     input.UniqueIntentID,
		BatchID:            input.BatchID,
		SourceMatrixID:     input.SourceMatrixID,
		Term:               input.Term,
		CandidatePath:      input.CandidatePath,
		PaidIntentStatus:   status,
		PaidBusinessScore:  coreScore.paidBusinessScore,
		CorePaidScore:      coreScore.paidBusinessScore,
		CTAPaidScore:       ctaScore.paidBusinessScore,
		PaidSignals:        coreScore.paidSignals,
		CTAPaidSignals:     ctaScore.paidSignals,
		BusinessSignals:    coreScore.businessSignals,
		RiskSignals:        uniqueStrings(risks),
		RoutingDecision:    routing,
		IndexPolicy:        "noindex",
		RenderAllowed:      false,
		SitemapAllowed:     false,
		PublicationAllowed: false,
		PublicPath:         "",
		CheckedAt:          input.CheckedAt,
	}
}

func ValidateText(text string, mode Mode) Report {
	return ValidateTextParts(text, "", mode)
}

func ValidateTextParts(coreText string, ctaText string, mode Mode) Report {
	coreScore := scoreText(coreText)
	ctaScore := scoreText(ctaText)
	combinedScore := scoreText(strings.TrimSpace(coreText + " " + ctaText))
	issues := make([]Issue, 0)
	if len(combinedScore.freeSignals) > 0 {
		issues = append(issues, Issue{
			Code:    "paid_intent_free_service_signal",
			Message: "texto induz gratuidade/nao-pagamento: " + strings.Join(combinedScore.freeSignals, ","),
		})
	}
	if len(combinedScore.researchSignals) > 0 {
		issues = append(issues, Issue{
			Code:    "paid_intent_research_only_signal",
			Message: "texto parece pesquisa academica/curiosidade, nao contratacao: " + strings.Join(combinedScore.researchSignals, ","),
		})
	}
	if len(combinedScore.publicAssistanceSignals) > 0 {
		issues = append(issues, Issue{
			Code:    "paid_intent_public_assistance_free_risk",
			Message: "texto indica assistencia publica/beneficio de baixa renda: " + strings.Join(combinedScore.publicAssistanceSignals, ","),
		})
	}
	if len(combinedScore.adminSelfServiceSignals) > 0 {
		issues = append(issues, Issue{
			Code:    "paid_intent_admin_self_service_risk",
			Message: "texto indica fluxo administrativo de autoatendimento: " + strings.Join(combinedScore.adminSelfServiceSignals, ","),
		})
	}
	if len(issues) > 0 {
		return Report{Issues: issues}
	}
	if mode == RequirePaidSignal && len(coreScore.paidSignals) == 0 {
		if len(ctaScore.paidSignals) > 0 {
			issues = append(issues, Issue{
				Code:    "paid_intent_cta_only_paid_signal",
				Message: "sinal de contratacao paga ficou apenas no CTA/WhatsApp; o corpo precisa conter intencao de contratacao particular sem apelo artificial",
			})
		} else {
			issues = append(issues, Issue{
				Code:    "paid_intent_missing_paid_signal",
				Message: "faltou sinal explicito de contratacao paga, honorarios, consulta paga ou atendimento particular no corpo do conteudo",
			})
		}
	}
	if mode == RequirePaidSignal && coreScore.paidBusinessScore < minimumPaidBusinessScore {
		issues = append(issues, Issue{
			Code:    "paid_intent_low_business_score",
			Message: fmt.Sprintf("score=%d minimo=%d paid=%s business=%s", coreScore.paidBusinessScore, minimumPaidBusinessScore, strings.Join(coreScore.paidSignals, ","), strings.Join(coreScore.businessSignals, ",")),
		})
	}
	return Report{Issues: issues}
}

func AllowsExpansion(record Record) bool {
	if record.RenderAllowed || record.SitemapAllowed || record.PublicationAllowed || record.PublicPath != "" {
		return false
	}
	if record.PaidIntentStatus == PassedBlockedStatus {
		return true
	}
	return record.BatchID == "batch-previdenciario-digital" && record.PaidIntentStatus == PrevidenciarioInformationalStatus
}

func scoreText(text string) textScore {
	normalized := normalize(text)
	paid := matchedSignals(normalized, paidServiceSignals)
	business := matchedSignals(normalized, businessReadinessSignals)
	free := matchedSignals(normalized, freeServiceSignals)
	research := matchedSignals(normalized, researchOnlySignals)
	publicAssistance := matchedSignals(normalized, publicAssistanceRiskSignals)
	adminSelfService := matchedSignals(normalized, adminSelfServiceRiskSignals)
	score := len(paid)*3 + min(len(business), 3)
	return textScore{
		paidSignals:             paid,
		businessSignals:         business,
		freeSignals:             free,
		researchSignals:         research,
		publicAssistanceSignals: publicAssistance,
		adminSelfServiceSignals: adminSelfService,
		paidBusinessScore:       score,
	}
}

func compareGateToEvaluation(record Record, expected Record) []Issue {
	issues := make([]Issue, 0)
	if record.PaidIntentGateID != expected.PaidIntentGateID {
		issues = append(issues, Issue{Code: "paid_intent_gate_id_mismatch", Message: record.UniqueIntentID})
	}
	if record.GateScope != expected.GateScope {
		issues = append(issues, Issue{Code: "paid_intent_gate_scope_mismatch", Message: fmt.Sprintf("%s record=%s expected=%s", record.UniqueIntentID, record.GateScope, expected.GateScope)})
	}
	if record.PaidIntentStatus != expected.PaidIntentStatus {
		issues = append(issues, Issue{Code: "paid_intent_gate_status_mismatch", Message: fmt.Sprintf("%s record=%s expected=%s", record.UniqueIntentID, record.PaidIntentStatus, expected.PaidIntentStatus)})
	}
	if record.RoutingDecision != expected.RoutingDecision {
		issues = append(issues, Issue{Code: "paid_intent_gate_routing_mismatch", Message: record.UniqueIntentID})
	}
	if record.PaidBusinessScore != expected.PaidBusinessScore {
		issues = append(issues, Issue{Code: "paid_intent_gate_score_mismatch", Message: fmt.Sprintf("%s record=%d expected=%d", record.UniqueIntentID, record.PaidBusinessScore, expected.PaidBusinessScore)})
	}
	if record.CorePaidScore != expected.CorePaidScore {
		issues = append(issues, Issue{Code: "paid_intent_gate_core_score_mismatch", Message: fmt.Sprintf("%s record=%d expected=%d", record.UniqueIntentID, record.CorePaidScore, expected.CorePaidScore)})
	}
	if record.CTAPaidScore != expected.CTAPaidScore {
		issues = append(issues, Issue{Code: "paid_intent_gate_cta_score_mismatch", Message: fmt.Sprintf("%s record=%d expected=%d", record.UniqueIntentID, record.CTAPaidScore, expected.CTAPaidScore)})
	}
	if !sameStringSet(record.PaidSignals, expected.PaidSignals) {
		issues = append(issues, Issue{Code: "paid_intent_gate_paid_signals_mismatch", Message: fmt.Sprintf("%s record=%s expected=%s", record.UniqueIntentID, strings.Join(record.PaidSignals, ","), strings.Join(expected.PaidSignals, ","))})
	}
	if !sameStringSet(record.CTAPaidSignals, expected.CTAPaidSignals) {
		issues = append(issues, Issue{Code: "paid_intent_gate_cta_paid_signals_mismatch", Message: fmt.Sprintf("%s record=%s expected=%s", record.UniqueIntentID, strings.Join(record.CTAPaidSignals, ","), strings.Join(expected.CTAPaidSignals, ","))})
	}
	if !sameStringSet(record.BusinessSignals, expected.BusinessSignals) {
		issues = append(issues, Issue{Code: "paid_intent_gate_business_signals_mismatch", Message: fmt.Sprintf("%s record=%s expected=%s", record.UniqueIntentID, strings.Join(record.BusinessSignals, ","), strings.Join(expected.BusinessSignals, ","))})
	}
	if !sameStringSet(record.RiskSignals, expected.RiskSignals) {
		issues = append(issues, Issue{Code: "paid_intent_gate_risk_signals_mismatch", Message: fmt.Sprintf("%s record=%s expected=%s", record.UniqueIntentID, strings.Join(record.RiskSignals, ","), strings.Join(expected.RiskSignals, ","))})
	}
	return issues
}

func validStatus(value string) bool {
	switch value {
	case PassedBlockedStatus,
		PrevidenciarioInformationalStatus,
		PublicAssistanceBlockedStatus,
		AdminSelfServiceBlockedStatus,
		FreeServiceBlockedStatus,
		ResearchOnlyBlockedStatus,
		CTAOnlyBlockedStatus,
		MissingPaidSignalStatus,
		LowBusinessScoreStatus:
		return true
	default:
		return false
	}
}

func isPrevidenciarioInformationalFlex(input EvaluationInput, status string) bool {
	if input.BatchID != "batch-previdenciario-digital" {
		return false
	}
	switch status {
	case PublicAssistanceBlockedStatus, AdminSelfServiceBlockedStatus, ResearchOnlyBlockedStatus, CTAOnlyBlockedStatus, MissingPaidSignalStatus, LowBusinessScoreStatus:
		return true
	default:
		return false
	}
}

func matchedSignals(text string, signals []string) []string {
	matches := make([]string, 0)
	for _, signal := range signals {
		if strings.Contains(text, signal) {
			matches = append(matches, signal)
		}
	}
	sort.Strings(matches)
	return matches
}

func normalize(text string) string {
	mapped := strings.Map(func(r rune) rune {
		switch r {
		case 'á', 'à', 'â', 'ã', 'ä', 'Á', 'À', 'Â', 'Ã', 'Ä':
			return 'a'
		case 'é', 'è', 'ê', 'ë', 'É', 'È', 'Ê', 'Ë':
			return 'e'
		case 'í', 'ì', 'î', 'ï', 'Í', 'Ì', 'Î', 'Ï':
			return 'i'
		case 'ó', 'ò', 'ô', 'õ', 'ö', 'Ó', 'Ò', 'Ô', 'Õ', 'Ö':
			return 'o'
		case 'ú', 'ù', 'û', 'ü', 'Ú', 'Ù', 'Û', 'Ü':
			return 'u'
		case 'ç', 'Ç':
			return 'c'
		default:
			if unicode.IsLetter(r) || unicode.IsDigit(r) {
				return unicode.ToLower(r)
			}
			return ' '
		}
	}, text)
	return strings.Join(strings.Fields(mapped), " ")
}

func min(left int, right int) int {
	if left < right {
		return left
	}
	return right
}

func uniqueStrings(values []string) []string {
	seen := make(map[string]bool)
	result := make([]string, 0, len(values))
	for _, value := range values {
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		result = append(result, value)
	}
	sort.Strings(result)
	return result
}

func collectRiskSignals(score textScore) []string {
	risks := append([]string{}, score.freeSignals...)
	risks = append(risks, score.researchSignals...)
	risks = append(risks, score.publicAssistanceSignals...)
	risks = append(risks, score.adminSelfServiceSignals...)
	return risks
}

func candidatePathForIntent(intentID string) string {
	return "/temas/" + intentID + "/"
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

func convertDraftIssues(report batchfinaldrafts.Report) []Issue {
	issues := make([]Issue, 0, len(report.Issues))
	for _, issue := range report.Issues {
		issues = append(issues, Issue{Code: "paid_intent_draft_" + issue.Code, Message: issue.Message})
	}
	return issues
}

func convertArchiveIssues(report batchdraftarchive.Report) []Issue {
	issues := make([]Issue, 0, len(report.Issues))
	for _, issue := range report.Issues {
		issues = append(issues, Issue{Code: "paid_intent_archive_" + issue.Code, Message: issue.Message})
	}
	return issues
}

func loadExpansionReadinessTargets(root string) (map[string]bool, Report) {
	projectRoot, err := findProjectRoot(root)
	if err != nil {
		return nil, Report{Issues: []Issue{{Code: "project_root_not_found", Message: err.Error()}}}
	}
	archiveEntries, archiveReport := batchdraftarchive.LoadRecords(root)
	if !archiveReport.Passed() {
		return nil, Report{Issues: convertArchiveIssues(archiveReport)}
	}
	archiveIntentIDsByBatch := make(map[string][]string)
	for _, entry := range archiveEntries {
		archiveIntentIDsByBatch[entry.Record.BatchID] = append(archiveIntentIDsByBatch[entry.Record.BatchID], entry.Record.UniqueIntentID)
	}
	path := filepath.Join(projectRoot, "data", "editorial", "batch_candidate_expansion_readiness.jsonl")
	file, err := os.Open(path)
	if err != nil {
		return nil, Report{Issues: []Issue{{Code: "paid_intent_expansion_readiness_missing", Message: err.Error()}}}
	}
	defer file.Close()

	targets := make(map[string]bool)
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
		var record readinessRecord
		if err := json.Unmarshal([]byte(line), &record); err != nil {
			issues = append(issues, Issue{Code: "paid_intent_expansion_readiness_invalid_json", Message: fmt.Sprintf("line=%d %s", lineNumber, err.Error())})
			continue
		}
		intentIDs := record.ExpansionCandidateIntentIDs
		if record.ExpansionCandidateSelector == "archive_prefix_by_batch" || len(intentIDs) == 0 {
			intentIDs = firstNStrings(archiveIntentIDsByBatch[record.BatchID], record.TargetCandidateCount)
		}
		for _, intentID := range intentIDs {
			targets[intentID] = true
		}
	}
	if err := scanner.Err(); err != nil {
		issues = append(issues, Issue{Code: "paid_intent_expansion_readiness_scan_failed", Message: err.Error()})
	}
	return targets, Report{Issues: issues}
}

func firstNStrings(values []string, limit int) []string {
	if limit <= 0 || len(values) == 0 {
		return nil
	}
	if limit > len(values) {
		limit = len(values)
	}
	return append([]string{}, values[:limit]...)
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

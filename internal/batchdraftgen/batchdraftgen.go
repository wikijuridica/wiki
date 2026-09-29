package batchdraftgen

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode"

	"portaljuridico/internal/batchdrafts"
	"portaljuridico/internal/batchsourcematrix"
	"portaljuridico/internal/humanscore"
	"portaljuridico/internal/scalablebatches"
	"portaljuridico/internal/storage"
)

type Options struct {
	SamplesPerBatch int
	CheckedAt       string
}

type Metric struct {
	BatchID                    string  `json:"batch_id"`
	GenerationStatus           string  `json:"generation_status"`
	GeneratedSamples           int     `json:"generated_samples"`
	PassedSamples              int     `json:"passed_samples"`
	RewrittenSamples           int     `json:"rewritten_samples"`
	MinimumHumanScore          int     `json:"minimum_human_score"`
	MaximumAILikeScore         int     `json:"maximum_ai_like_score"`
	MaximumSimilarity          float64 `json:"maximum_similarity"`
	SourceMatrixCoveredSamples int     `json:"source_matrix_covered_samples"`
	StructuralPatternRisk      float64 `json:"structural_pattern_risk"`
	LabEstimatedCPUUnits       int     `json:"lab_estimated_cpu_units"`
	RenderAllowed              bool    `json:"render_allowed"`
	SitemapAllowed             bool    `json:"sitemap_allowed"`
	PublicationAllowed         bool    `json:"publication_allowed"`
	PublicPath                 string  `json:"public_path"`
	CheckedAt                  string  `json:"checked_at"`
	NextValidationAction       string  `json:"next_validation_action"`
}

type MetricEntry struct {
	Line   int
	Metric Metric
}

type Result struct {
	Drafts  []batchdrafts.Record
	Metrics []Metric
}

type Issue struct {
	Code    string
	Message string
}

type Report struct {
	Issues []Issue
}

type scenario struct {
	IntentID        string
	Term            string
	ReaderProblem   string
	SourceHook      string
	DocumentContext string
	RiskContext     string
	DigitalAction   string
}

type semanticFacet struct {
	ID                 string
	TermContext        string
	ReaderFocus        string
	SourceFocus        string
	DocumentFocus      string
	RiskFocus          string
	DigitalFocus       string
	OperationalContext string
}

type cycleProfile struct {
	ID            string
	TermContext   string
	ReaderFocus   string
	SourceFocus   string
	DocumentFocus string
	RiskFocus     string
	DigitalFocus  string
}

func DefaultOptions() Options {
	return Options{SamplesPerBatch: 10, CheckedAt: "2026-06-09"}
}

func Generate(root string, options Options) (Result, Report) {
	if options.SamplesPerBatch < 3 {
		return Result{}, Report{Issues: []Issue{{Code: "generation_samples_too_low", Message: fmt.Sprintf("samples=%d", options.SamplesPerBatch)}}}
	}
	if options.CheckedAt == "" {
		return Result{}, Report{Issues: []Issue{{Code: "generation_checked_at_missing", Message: "checked_at vazio"}}}
	}

	entries, loadReport := scalablebatches.LoadRecords(root)
	if !loadReport.Passed() {
		return Result{}, convertBatchReport(loadReport)
	}
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Record.BatchID < entries[j].Record.BatchID
	})

	issues := make([]Issue, 0)
	drafts := make([]batchdrafts.Record, 0, len(entries)*options.SamplesPerBatch)
	for _, entry := range entries {
		if validation := scalablebatches.ValidateRecord(entry.Record); !validation.Passed() {
			for _, issue := range validation.Issues {
				issues = append(issues, Issue{Code: "generation_invalid_batch", Message: issue.Code + ":" + issue.Message})
			}
			continue
		}
		scenarios := expandScenarios(entry.Record.LegalArea, scenariosForArea(entry.Record.LegalArea), options.SamplesPerBatch)
		if len(scenarios) < options.SamplesPerBatch {
			issues = append(issues, Issue{Code: "generation_scenario_bank_too_small", Message: entry.Record.LegalArea})
			continue
		}
		for i := 0; i < options.SamplesPerBatch; i++ {
			drafts = append(drafts, buildDraft(entry.Record, scenarios[i], options.CheckedAt))
		}
	}
	if len(issues) > 0 {
		return Result{Drafts: drafts}, Report{Issues: issues}
	}

	result := Result{Drafts: drafts, Metrics: buildMetrics(drafts, options.CheckedAt)}
	return result, ValidateResultWithRoot(root, result)
}

func ValidateResult(result Result) Report {
	return validateResult(result, nil)
}

func ValidateResultWithRoot(root string, result Result) Report {
	matrixEntries, matrixReport := batchsourcematrix.LoadRecords(root)
	if !matrixReport.Passed() {
		return validateResult(result, nil)
	}
	return validateResult(result, matrixEntries)
}

func validateResult(result Result, matrixEntries []batchsourcematrix.Entry) Report {
	issues := make([]Issue, 0)
	if len(result.Drafts) < 30 {
		issues = append(issues, Issue{Code: "generation_too_few_drafts", Message: fmt.Sprintf("drafts=%d", len(result.Drafts))})
	}
	rewrittenCount := result.RewrittenCount()
	maxSimilarity := result.MaximumPairSimilarity()
	structuralRisk := result.StructuralPatternRisk()
	seen := make(map[string]bool)
	for _, draft := range result.Drafts {
		if seen[draft.UniqueIntentID] {
			issues = append(issues, Issue{Code: "generation_duplicate_intent", Message: draft.UniqueIntentID})
		}
		seen[draft.UniqueIntentID] = true
		validation := batchdrafts.ValidateRecord(draft)
		for _, issue := range validation.Issues {
			issues = append(issues, Issue{Code: "generation_invalid_draft", Message: draft.UniqueIntentID + ":" + issue.Code + ":" + issue.Message})
		}
		if !ContainsOrigin(draft.CTAContext, draft.UniqueIntentID) {
			issues = append(issues, Issue{Code: "generation_cta_without_origin", Message: draft.UniqueIntentID})
		}
	}
	if rewrittenCount < 6 {
		issues = append(issues, Issue{Code: "generation_too_few_rewrites", Message: fmt.Sprintf("rewritten=%d", rewrittenCount)})
	}
	if maxSimilarity > 0.64 {
		issues = append(issues, Issue{Code: "generation_similarity_too_high", Message: fmt.Sprintf("max=%.2f", maxSimilarity)})
	}
	if structuralRisk > 0.35 {
		issues = append(issues, Issue{Code: "generation_structural_pattern_risk_too_high", Message: fmt.Sprintf("risk=%.2f", structuralRisk)})
	}
	matrixEntriesForDiversity := make([]batchdrafts.Entry, 0, len(result.Drafts))
	for index, draft := range result.Drafts {
		matrixEntriesForDiversity = append(matrixEntriesForDiversity, batchdrafts.Entry{Line: index + 1, Record: draft})
	}
	matrixReport := batchdrafts.ValidateSourceMatrixDiversity(matrixEntriesForDiversity)
	for _, issue := range matrixReport.Issues {
		issues = append(issues, Issue{Code: "generation_" + issue.Code, Message: issue.Message})
	}
	if len(matrixEntries) > 0 {
		coverage := batchsourcematrix.ValidateDraftCoverage(matrixEntries, result.Drafts)
		for _, issue := range coverage.Issues {
			issues = append(issues, Issue{Code: "generation_source_matrix_gap", Message: issue.Code + ":" + issue.Message})
		}
	}
	metricReport := ValidateMetrics(result.Metrics)
	issues = append(issues, metricReport.Issues...)
	return Report{Issues: issues}
}

func ValidateMetric(metric Metric) Report {
	issues := make([]Issue, 0)
	if metric.BatchID == "" {
		issues = append(issues, Issue{Code: "generation_metric_missing_batch", Message: "batch_id vazio"})
	}
	if metric.GenerationStatus != "batch_generation_scored_blocked" {
		issues = append(issues, Issue{Code: "generation_metric_invalid_status", Message: metric.GenerationStatus})
	}
	if metric.GeneratedSamples < 5 {
		issues = append(issues, Issue{Code: "generation_metric_too_few_samples", Message: fmt.Sprintf("generated=%d", metric.GeneratedSamples)})
	}
	if metric.PassedSamples != metric.GeneratedSamples || metric.PassedSamples == 0 {
		issues = append(issues, Issue{Code: "generation_metric_unpassed_samples", Message: fmt.Sprintf("passed=%d generated=%d", metric.PassedSamples, metric.GeneratedSamples)})
	}
	if metric.MinimumHumanScore < 85 {
		issues = append(issues, Issue{Code: "generation_metric_human_score_too_low", Message: fmt.Sprintf("min=%d", metric.MinimumHumanScore)})
	}
	if metric.MaximumAILikeScore > 20 {
		issues = append(issues, Issue{Code: "generation_metric_ai_score_too_high", Message: fmt.Sprintf("max=%d", metric.MaximumAILikeScore)})
	}
	if metric.MaximumSimilarity <= 0 || metric.MaximumSimilarity > 0.64 {
		issues = append(issues, Issue{Code: "generation_metric_similarity_too_high", Message: fmt.Sprintf("max=%.2f", metric.MaximumSimilarity)})
	}
	if metric.SourceMatrixCoveredSamples != metric.GeneratedSamples {
		issues = append(issues, Issue{Code: "generation_metric_source_matrix_gap", Message: fmt.Sprintf("covered=%d generated=%d", metric.SourceMatrixCoveredSamples, metric.GeneratedSamples)})
	}
	if metric.StructuralPatternRisk > 0.35 {
		issues = append(issues, Issue{Code: "generation_metric_structural_risk_too_high", Message: fmt.Sprintf("risk=%.2f", metric.StructuralPatternRisk)})
	}
	if metric.LabEstimatedCPUUnits <= 0 {
		issues = append(issues, Issue{Code: "generation_metric_cpu_estimate_missing", Message: metric.BatchID})
	}
	if metric.RenderAllowed {
		issues = append(issues, Issue{Code: "generation_metric_render_allowed", Message: metric.BatchID})
	}
	if metric.SitemapAllowed {
		issues = append(issues, Issue{Code: "generation_metric_sitemap_allowed", Message: metric.BatchID})
	}
	if metric.PublicationAllowed {
		issues = append(issues, Issue{Code: "generation_metric_publication_allowed", Message: metric.BatchID})
	}
	if metric.PublicPath != "" {
		issues = append(issues, Issue{Code: "generation_metric_has_public_path", Message: metric.PublicPath})
	}
	if metric.CheckedAt == "" {
		issues = append(issues, Issue{Code: "generation_metric_without_checked_at", Message: metric.BatchID})
	}
	return Report{Issues: issues}
}

func ValidateMetrics(metrics []Metric) Report {
	issues := make([]Issue, 0)
	if len(metrics) < 6 {
		issues = append(issues, Issue{Code: "generation_metrics_too_few_batches", Message: fmt.Sprintf("metrics=%d", len(metrics))})
	}
	seen := make(map[string]bool)
	for _, metric := range metrics {
		if seen[metric.BatchID] {
			issues = append(issues, Issue{Code: "generation_metric_duplicate_batch", Message: metric.BatchID})
		}
		seen[metric.BatchID] = true
		report := ValidateMetric(metric)
		issues = append(issues, report.Issues...)
	}
	return Report{Issues: issues}
}

func ValidateStoredMetrics(root string) Report {
	entries, report := LoadMetrics(root)
	if !report.Passed() {
		return report
	}
	metrics := make([]Metric, 0, len(entries))
	for _, entry := range entries {
		metrics = append(metrics, entry.Metric)
	}
	return ValidateMetrics(metrics)
}

func LoadMetrics(root string) ([]MetricEntry, Report) {
	projectRoot, err := findProjectRoot(root)
	if err != nil {
		return nil, Report{Issues: []Issue{{Code: "project_root_not_found", Message: err.Error()}}}
	}
	path := filepath.Join(projectRoot, "data", "editorial", "batch_generation_metrics.jsonl")
	file, err := os.Open(path)
	if err != nil {
		return nil, Report{Issues: []Issue{{Code: "generation_metrics_missing", Message: err.Error()}}}
	}
	defer file.Close()

	entries := make([]MetricEntry, 0)
	issues := make([]Issue, 0)
	scanner := bufio.NewScanner(file)
	lineNumber := 0
	for scanner.Scan() {
		lineNumber++
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var metric Metric
		if err := json.Unmarshal([]byte(line), &metric); err != nil {
			issues = append(issues, Issue{Code: "generation_metric_invalid_json", Message: fmt.Sprintf("line=%d %s", lineNumber, err.Error())})
			continue
		}
		entries = append(entries, MetricEntry{Line: lineNumber, Metric: metric})
	}
	if err := scanner.Err(); err != nil {
		issues = append(issues, Issue{Code: "generation_metric_scan_failed", Message: err.Error()})
	}
	return entries, Report{Issues: issues}
}

func WriteMetrics(root string, metrics []Metric) error {
	contract, err := storage.LoadContract(root)
	if err != nil {
		return err
	}
	layer, ok := contract.LayerByName("batch_generation_metrics")
	if !ok {
		return fmt.Errorf("unknown_layer=batch_generation_metrics")
	}
	path := filepath.Join(contract.Root, layer.Path)
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	defer file.Close()
	for _, metric := range metrics {
		if report := ValidateMetric(metric); !report.Passed() {
			return fmt.Errorf("invalid_generation_metric=%s", strings.Join(report.Messages(), " | "))
		}
		data, err := json.Marshal(metric)
		if err != nil {
			return err
		}
		if len(data)+1 > layer.RecordMaxBytes {
			return fmt.Errorf("record_too_large=batch_generation_metrics bytes=%d max=%d", len(data)+1, layer.RecordMaxBytes)
		}
		if _, err := file.Write(append(data, '\n')); err != nil {
			return err
		}
	}
	return nil
}

func WriteSamples(root string, drafts []batchdrafts.Record) error {
	contract, err := storage.LoadContract(root)
	if err != nil {
		return err
	}
	layer, ok := contract.LayerByName("batch_drafts")
	if !ok {
		return fmt.Errorf("unknown_layer=batch_drafts")
	}
	payload, err := prepareSamplesPayload(drafts, layer.RecordMaxBytes)
	if err != nil {
		return err
	}
	path := filepath.Join(contract.Root, layer.Path)
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	tempFile, err := os.CreateTemp(filepath.Dir(path), ".batch_drafts-*.tmp")
	if err != nil {
		return err
	}
	tempPath := tempFile.Name()
	cleanup := true
	defer func() {
		if cleanup {
			_ = os.Remove(tempPath)
		}
	}()
	if _, err := tempFile.Write(payload); err != nil {
		_ = tempFile.Close()
		return err
	}
	if err := tempFile.Close(); err != nil {
		return err
	}
	if err := os.Rename(tempPath, path); err != nil {
		return err
	}
	cleanup = false
	return nil
}

func WriteArchive(root string, drafts []batchdrafts.Record) error {
	contract, err := storage.LoadContract(root)
	if err != nil {
		return err
	}
	layer, ok := contract.LayerByName("batch_draft_expansion_archive")
	if !ok {
		return fmt.Errorf("unknown_layer=batch_draft_expansion_archive")
	}
	payload, err := prepareArchivePayload(drafts, layer.RecordMaxBytes)
	if err != nil {
		return err
	}
	path := filepath.Join(contract.Root, layer.Path)
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	tempFile, err := os.CreateTemp(filepath.Dir(path), ".batch_draft_expansion_archive-*.tmp")
	if err != nil {
		return err
	}
	tempPath := tempFile.Name()
	cleanup := true
	defer func() {
		if cleanup {
			_ = os.Remove(tempPath)
		}
	}()
	if _, err := tempFile.Write(payload); err != nil {
		_ = tempFile.Close()
		return err
	}
	if err := tempFile.Close(); err != nil {
		return err
	}
	if err := os.Rename(tempPath, path); err != nil {
		return err
	}
	cleanup = false
	return nil
}

func prepareSamplesPayload(drafts []batchdrafts.Record, recordMaxBytes int) ([]byte, error) {
	if err := validateSampleDrafts(drafts); err != nil {
		return nil, err
	}
	var buffer bytes.Buffer
	for _, draft := range drafts {
		if report := batchdrafts.ValidateRecord(draft); !report.Passed() {
			return nil, fmt.Errorf("invalid_sample_draft=%s %s", draft.UniqueIntentID, strings.Join(report.Messages(), " | "))
		}
		data, err := json.Marshal(draft)
		if err != nil {
			return nil, err
		}
		if len(data)+1 > recordMaxBytes {
			return nil, fmt.Errorf("record_too_large=batch_drafts bytes=%d max=%d id=%s", len(data)+1, recordMaxBytes, draft.UniqueIntentID)
		}
		buffer.Write(data)
		buffer.WriteByte('\n')
	}
	return buffer.Bytes(), nil
}

func prepareArchivePayload(drafts []batchdrafts.Record, recordMaxBytes int) ([]byte, error) {
	if err := validateArchiveDrafts(drafts); err != nil {
		return nil, err
	}
	var buffer bytes.Buffer
	for _, draft := range drafts {
		if report := batchdrafts.ValidateRecord(draft); !report.Passed() {
			return nil, fmt.Errorf("invalid_archive_draft=%s %s", draft.UniqueIntentID, strings.Join(report.Messages(), " | "))
		}
		data, err := json.Marshal(draft)
		if err != nil {
			return nil, err
		}
		if len(data)+1 > recordMaxBytes {
			return nil, fmt.Errorf("record_too_large=batch_draft_expansion_archive bytes=%d max=%d id=%s", len(data)+1, recordMaxBytes, draft.UniqueIntentID)
		}
		buffer.Write(data)
		buffer.WriteByte('\n')
	}
	return buffer.Bytes(), nil
}

func validateSampleDrafts(drafts []batchdrafts.Record) error {
	if len(drafts) < 18 {
		return fmt.Errorf("sample_drafts_too_few=%d", len(drafts))
	}
	entries := make([]batchdrafts.Entry, 0, len(drafts))
	perBatch := make(map[string]int)
	seenIntent := make(map[string]int)
	rewritten := 0
	for index, draft := range drafts {
		if report := batchdrafts.ValidateRecord(draft); !report.Passed() {
			return fmt.Errorf("invalid_sample_draft=%s %s", draft.UniqueIntentID, strings.Join(report.Messages(), " | "))
		}
		if previous := seenIntent[draft.UniqueIntentID]; previous > 0 {
			return fmt.Errorf("sample_draft_duplicate_intent=line:%d previous_line:%d id:%s", index+1, previous, draft.UniqueIntentID)
		}
		seenIntent[draft.UniqueIntentID] = index + 1
		perBatch[draft.BatchID]++
		if draft.RewriteStatus == "rewritten_after_score_failure" {
			rewritten++
		}
		entries = append(entries, batchdrafts.Entry{Line: index + 1, Record: draft})
	}
	for batchID, count := range perBatch {
		if count < 3 {
			return fmt.Errorf("sample_drafts_too_few_per_batch=%s:%d", batchID, count)
		}
	}
	if rewritten < 3 {
		return fmt.Errorf("sample_drafts_too_few_rewrites=%d", rewritten)
	}
	if pair := batchdrafts.MaximumPairSimilarityDetail(entries); pair.Score > 0.64 {
		return fmt.Errorf("sample_drafts_similarity_too_high=%.4f left=%s right=%s", pair.Score, pair.LeftID, pair.RightID)
	}
	return nil
}

func validateArchiveDrafts(drafts []batchdrafts.Record) error {
	if len(drafts) < 600 {
		return fmt.Errorf("archive_drafts_too_few=%d", len(drafts))
	}
	entries := make([]batchdrafts.Entry, 0, len(drafts))
	perBatch := make(map[string]int)
	seenIntent := make(map[string]int)
	rewritten := 0
	for index, draft := range drafts {
		if report := batchdrafts.ValidateRecord(draft); !report.Passed() {
			return fmt.Errorf("invalid_archive_draft=%s %s", draft.UniqueIntentID, strings.Join(report.Messages(), " | "))
		}
		if previous := seenIntent[draft.UniqueIntentID]; previous > 0 {
			return fmt.Errorf("archive_draft_duplicate_intent=line:%d previous_line:%d id:%s", index+1, previous, draft.UniqueIntentID)
		}
		seenIntent[draft.UniqueIntentID] = index + 1
		if draft.SourceMatrixID == "" {
			return fmt.Errorf("archive_draft_without_source_matrix=%s", draft.UniqueIntentID)
		}
		perBatch[draft.BatchID]++
		if draft.RewriteStatus == "rewritten_after_score_failure" {
			rewritten++
		}
		entries = append(entries, batchdrafts.Entry{Line: index + 1, Record: draft})
	}
	for batchID, count := range perBatch {
		if count < 100 {
			return fmt.Errorf("archive_drafts_too_few_per_batch=%s:%d", batchID, count)
		}
	}
	if rewritten < 300 {
		return fmt.Errorf("archive_drafts_too_few_rewrites=%d", rewritten)
	}
	if pair := batchdrafts.MaximumPairSimilarityDetail(entries); pair.Score > 0.64 {
		return fmt.Errorf("archive_drafts_similarity_too_high=%.4f left=%s right=%s", pair.Score, pair.LeftID, pair.RightID)
	}
	matrixReport := batchdrafts.ValidateSourceMatrixDiversity(entries)
	if !matrixReport.Passed() {
		return fmt.Errorf("archive_drafts_source_matrix_diversity_failed=%s", strings.Join(matrixReport.Messages(), " | "))
	}
	return nil
}

func (r Result) DraftIDs() []string {
	ids := make([]string, 0, len(r.Drafts))
	for _, draft := range r.Drafts {
		ids = append(ids, draft.UniqueIntentID)
	}
	return ids
}

func (r Result) RewrittenCount() int {
	count := 0
	for _, draft := range r.Drafts {
		if draft.RewriteStatus == "rewritten_after_score_failure" {
			count++
		}
	}
	return count
}

func (r Result) MaximumPairSimilarity() float64 {
	entries := make([]batchdrafts.Entry, 0, len(r.Drafts))
	for i, draft := range r.Drafts {
		entries = append(entries, batchdrafts.Entry{Line: i + 1, Record: draft})
	}
	return batchdrafts.MaximumPairSimilarity(entries)
}

func (r Result) StructuralPatternRisk() float64 {
	return structuralPatternRisk(r.Drafts)
}

func ContainsOrigin(value string, uniqueIntentID string) bool {
	return strings.Contains(strings.ToLower(value), strings.ToLower(uniqueIntentID))
}

func buildDraft(batch scalablebatches.Record, scenario scenario, checkedAt string) batchdrafts.Record {
	record := batchdrafts.Record{
		BatchID:            batch.BatchID,
		UniqueIntentID:     batch.LegalArea + "-" + scenario.IntentID,
		SourceMatrixID:     batch.LegalArea + "-" + sourceMatrixID(scenario),
		LegalArea:          batch.LegalArea,
		DraftStatus:        "batch_draft_scored_blocked",
		Language:           "pt-BR",
		Term:               scenario.Term,
		ReaderProblem:      scenario.ReaderProblem,
		SourceHook:         scenario.SourceHook,
		DocumentContext:    scenario.DocumentContext,
		RiskContext:        scenario.RiskContext,
		DigitalAction:      scenario.DigitalAction,
		SourceFamilies:     append([]string{}, batch.SourceFamilies...),
		RewriteStatus:      "rewritten_after_score_failure",
		InitialIssueCodes:  initialIssueCodes(scenario.Term),
		RewriteAttempts:    1,
		RenderAllowed:      false,
		SitemapAllowed:     false,
		PublicationAllowed: false,
		PublicPath:         "",
		CheckedAt:          checkedAt,
	}
	record.DigitalAction = addEditorialDepth(record.DigitalAction, scenario, record.LegalArea)
	record.DigitalAction = addSpecificityRepairIfNeeded(record, scenario)
	sourceFamily := ""
	if len(batch.SourceFamilies) > 0 {
		sourceFamily = batch.SourceFamilies[0]
	}
	record.CTAContext = strings.ReplaceAll(batch.CTAContextTemplate, "{unique_intent_id}", record.UniqueIntentID)
	record.CTAContext = strings.ReplaceAll(record.CTAContext, "{source_family}", sourceFamily)
	score := humanscore.ScoreText(record.FullText())
	record.HumanScore = score.HumanScore
	record.AILikeScore = score.AILikeScore
	return record
}

func addSpecificityRepairIfNeeded(record batchdrafts.Record, scenario scenario) string {
	score := humanscore.ScoreText(record.FullText())
	if !score.HasIssue("low_specificity") {
		return record.DigitalAction
	}
	cue := contextCueFrom(scenario.DocumentContext+" "+scenario.SourceHook+" "+scenario.ReaderProblem, 10, scenarioTemplateIndex(scenario.IntentID)+29)
	return record.DigitalAction + " Complemento de especificidade: antes de escalar o lote, a pauta confere " + cue + " e exige " + areaSpecificityEvidence(record.LegalArea) + ", sem publicar suposicao."
}

func areaSpecificityEvidence(area string) string {
	switch area {
	case "consumidor-financeiro":
		return "contrato, extratos, protocolo, resposta do banco, comprovante e prazo de contestacao"
	case "familia":
		return "sentenca ou acordo, certidao, renda, despesas, mensagens, calendario e documentos familiares"
	case "previdenciario":
		return "CNIS, comunicado de decisao, laudos, atestados, protocolo, recurso e requerimento"
	case "saude-suplementar":
		return "relatorio medico, negativa, contrato, carteirinha, protocolo, prazo da ANS e exames"
	case "sucessorio":
		return "certidao de obito, herdeiros, matricula, extratos, dividas, imposto e testamento quando houver"
	case "trabalhista":
		return "contrato, holerites, ponto, escala, mensagens, TRCT, FGTS e comprovante de pagamento"
	default:
		return "contrato, protocolo, resposta formal, comprovante, prazo e documento principal"
	}
}

func addEditorialDepth(current string, scenario scenario, area string) string {
	facetID := scenarioFacetID(scenario.IntentID)
	lead := readerLead(facetID)
	template := (scenarioTemplateIndex(scenario.IntentID) + scenarioTemplateIndex(facetID)) % 5
	cue := contextCueFrom(scenario.Term+" "+scenario.ReaderProblem+" "+scenario.DocumentContext+" "+scenario.RiskContext, 5, template+17)
	switch template {
	case 0:
		return current + " Revisao informativa de " + lead + " sobre " + cue + " separa fato, prova e duvida juridica para consulta online, sem promessa de resultado."
	case 1:
		return current + " Contexto editorial de " + lead + " ligado a " + cue + ": documentos, fonte e risco economico precisam conversar antes de contratar atendimento digital para " + areaContext(area) + "."
	case 2:
		return current + " Filtro juridico de " + lead + " com " + cue + ": o leitor entende limite da fonte, utilidade dos arquivos e diferenca entre orientacao geral e consulta particular."
	case 3:
		return current + " Validacao de " + lead + " em " + cue + ": o texto bloqueado cruza relato, documento e fonte antes de rota publica, mantendo CTA apenas como triagem contextual."
	default:
		return current + " Utilidade de " + lead + " para " + cue + ": a pauta organiza documentos, fase do problema e pergunta juridica para conversa objetiva com advogado."
	}
}

func scenarioFacetID(intentID string) string {
	for _, facet := range allSemanticFacets() {
		if strings.Contains(intentID, "-"+facet.ID) {
			return facet.ID
		}
	}
	return "contexto-juridico"
}

func initialIssueCodes(term string) []string {
	text := "Este conteúdo explica " + term + " de forma geral. Documentos necessários. Quando procurar advogado. " + strings.Repeat(term+" ", 8)
	score := humanscore.ScoreText(text)
	codes := score.Codes()
	if len(codes) == 0 {
		return []string{"ai_like_generic_markers"}
	}
	return codes
}

func buildMetrics(drafts []batchdrafts.Record, checkedAt string) []Metric {
	byBatch := make(map[string][]batchdrafts.Record)
	order := make([]string, 0)
	for _, draft := range drafts {
		if _, ok := byBatch[draft.BatchID]; !ok {
			order = append(order, draft.BatchID)
		}
		byBatch[draft.BatchID] = append(byBatch[draft.BatchID], draft)
	}
	sort.Strings(order)
	metrics := make([]Metric, 0, len(order))
	for _, batchID := range order {
		items := byBatch[batchID]
		minHuman := 100
		maxAI := 0
		passed := 0
		rewritten := 0
		entries := make([]batchdrafts.Entry, 0, len(items))
		for i, draft := range items {
			if draft.HumanScore < minHuman {
				minHuman = draft.HumanScore
			}
			if draft.AILikeScore > maxAI {
				maxAI = draft.AILikeScore
			}
			if batchdrafts.ValidateRecord(draft).Passed() {
				passed++
			}
			if draft.RewriteStatus == "rewritten_after_score_failure" {
				rewritten++
			}
			entries = append(entries, batchdrafts.Entry{Line: i + 1, Record: draft})
		}
		metrics = append(metrics, Metric{
			BatchID:                    batchID,
			GenerationStatus:           "batch_generation_scored_blocked",
			GeneratedSamples:           len(items),
			PassedSamples:              passed,
			RewrittenSamples:           rewritten,
			MinimumHumanScore:          minHuman,
			MaximumAILikeScore:         maxAI,
			MaximumSimilarity:          batchdrafts.MaximumPairSimilarity(entries),
			SourceMatrixCoveredSamples: sourceMatrixCoveredSamples(items),
			StructuralPatternRisk:      structuralPatternRisk(items),
			LabEstimatedCPUUnits:       len(items) * 12,
			RenderAllowed:              false,
			SitemapAllowed:             false,
			PublicationAllowed:         false,
			PublicPath:                 "",
			CheckedAt:                  checkedAt,
			NextValidationAction:       "aumentar lote com semantica por subtema, matriz de fonte especifica, similaridade por familia e publicacao bloqueada ate revisao e SEO passarem",
		})
	}
	return metrics
}

func sourceMatrixID(scenario scenario) string {
	for _, suffix := range []string{"-documentos-prazo", "-revisao-fonte", "-triagem-risco"} {
		if strings.HasSuffix(scenario.IntentID, suffix) {
			return strings.TrimSuffix(scenario.IntentID, suffix)
		}
	}
	for _, facet := range allSemanticFacets() {
		suffix := "-" + facet.ID
		if strings.HasSuffix(scenario.IntentID, suffix) {
			return strings.TrimSuffix(scenario.IntentID, suffix)
		}
		profiledMarker := suffix + "-"
		if index := strings.LastIndex(scenario.IntentID, profiledMarker); index > 0 && strings.Contains(scenario.IntentID[index+len(profiledMarker):], "rodada-") {
			return scenario.IntentID[:index]
		}
	}
	return scenario.IntentID
}

func sourceMatrixCoveredSamples(items []batchdrafts.Record) int {
	count := 0
	for _, item := range items {
		if item.SourceMatrixID != "" {
			count++
		}
	}
	return count
}

func structuralPatternRisk(items []batchdrafts.Record) float64 {
	if len(items) == 0 {
		return 0
	}
	counts := make(map[string]int)
	max := 0
	for _, item := range items {
		signature := patternSignature(item.ReaderProblem)
		counts[signature]++
		if counts[signature] > max {
			max = counts[signature]
		}
	}
	return float64(max) / float64(len(items))
}

func patternSignature(value string) string {
	words := strings.Fields(strings.Map(func(r rune) rune {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			return unicode.ToLower(r)
		}
		return ' '
	}, value))
	signals := make([]string, 0, 3)
	stop := map[string]bool{"o": true, "a": true, "os": true, "as": true, "um": true, "uma": true, "de": true, "do": true, "da": true, "e": true, "mas": true, "com": true, "para": true, "precisa": true}
	for _, word := range words {
		if !stop[word] {
			signals = append(signals, word)
		}
		if len(signals) == 3 {
			break
		}
	}
	if len(signals) == 0 {
		return "empty"
	}
	return strings.Join(signals, "-")
}

func expandScenarios(area string, base []scenario, count int) []scenario {
	if count <= len(base) {
		return base[:count]
	}
	expanded := append([]scenario{}, base...)
	variant := 1
	for len(expanded) < count {
		for index, item := range base {
			if len(expanded) >= count {
				break
			}
			expanded = append(expanded, scenarioVariant(area, item, variant+index, len(expanded)))
		}
		variant++
	}
	return expanded
}

func scenarioVariant(area string, base scenario, variant int, ordinal int) scenario {
	facets := semanticFacetsFor(area)
	facet := facets[(variant-1)%len(facets)]
	cycle := (variant - 1) / len(facets)
	profile := cycleProfile{}
	if ordinal >= 100 {
		profile = scenarioCycleProfile(area, cycle+ordinal-100)
	}
	facetID := facet.ID
	if cycle > 0 {
		facetID = fmt.Sprintf("%s-rodada-%02d", facet.ID, cycle+1)
		if profile.ID != "" {
			facetID = fmt.Sprintf("%s-%s-rodada-%02d", facet.ID, profile.ID, cycle+1)
		}
	}
	template := scenarioTemplateIndex(base.IntentID + "-" + facet.ID + "-" + profile.ID)
	term := variantTerm(template, base.Term, facet, profile)
	readerFocus := facet.ReaderFocus
	if profile.ReaderFocus != "" {
		readerFocus = readerFocus + "; " + profile.ReaderFocus
	}
	sourceFocus := facet.SourceFocus
	if profile.SourceFocus != "" {
		sourceFocus = sourceFocus + " " + profile.SourceFocus
	}
	documentFocus := facet.DocumentFocus
	if profile.DocumentFocus != "" {
		documentFocus = uppercaseFirst(profileBlendPrefix(facet.DocumentFocus, profile.DocumentFocus, template+ordinal)) + ". " + joinDistinctContext(documentFocus, profile.DocumentFocus)
	}
	riskFocus := facet.RiskFocus
	if profile.RiskFocus != "" {
		riskFocus = uppercaseFirst(profileBlendPrefix(facet.RiskFocus, profile.RiskFocus, template+ordinal+7)) + ". " + riskFocus + " " + profile.RiskFocus
	}
	digitalFocus := facet.DigitalFocus
	if profile.DigitalFocus != "" {
		digitalFocus = uppercaseFirst(profileBlendPrefix(facet.DigitalFocus, profile.DigitalFocus, template+ordinal+13)) + ". " + digitalFocus + " " + profile.DigitalFocus
	}
	return scenario{
		IntentID:        base.IntentID + "-" + facetID,
		Term:            term,
		ReaderProblem:   variantReaderProblem(template, facetID, area, readerFocus, base, profile),
		SourceHook:      variantSourceHook(template, facetID, sourceFocus, base, profile),
		DocumentContext: variantDocumentContext(template, facetID, documentFocus, base, profile),
		RiskContext:     variantRiskContext(template, facetID, riskFocus, base, profile),
		DigitalAction:   variantDigitalAction(template, facetID, digitalFocus, base, profile),
	}
}

func variantTerm(template int, baseTerm string, facet semanticFacet, profile cycleProfile) string {
	focus := facet.OperationalContext
	if profile.TermContext != "" {
		focus = profile.TermContext + " em " + facet.OperationalContext
	}
	switch template {
	case 0:
		return uppercaseFirst(focus) + " no caso de " + baseTerm
	case 1:
		return baseTerm + ": " + focus
	case 2:
		return baseTerm + " quando há " + focus
	case 3:
		return "Como analisar " + baseTerm + " com " + focus
	default:
		return "Documentos e risco em " + baseTerm + " com " + focus
	}
}

func profileBlendPrefix(primary string, secondary string, offset int) string {
	primaryCue := contextCueFrom(primary, 2, offset)
	secondaryCue := contextCueFrom(secondary, 2, offset+3)
	return primaryCue + " com " + secondaryCue
}

func subthemeCue(term string) string {
	words := make([]string, 0, 2)
	for _, word := range normalizedWords(term) {
		if len(word) > 3 && !localStopWord(word) {
			words = append(words, word)
		}
		if len(words) == 2 {
			break
		}
	}
	if len(words) == 0 {
		return "subtema especifico"
	}
	return strings.Join(words, " ")
}

func contextCue(value string, limit int) string {
	return contextCueFrom(value, limit, 0)
}

func contextCueFrom(value string, limit int, offset int) string {
	candidates := make([]string, 0, limit)
	seen := make(map[string]bool)
	for _, word := range normalizedWords(value) {
		if len(word) <= 3 || localStopWord(word) || seen[word] {
			continue
		}
		seen[word] = true
		candidates = append(candidates, word)
	}
	if len(candidates) == 0 {
		return "pista especifica pendente"
	}
	words := make([]string, 0, limit)
	start := offset % len(candidates)
	if start < 0 {
		start = 0
	}
	for index := 0; index < len(candidates); index++ {
		words = append(words, candidates[(start+index)%len(candidates)])
		if len(words) == limit {
			break
		}
	}
	if len(words) == 0 {
		return "pista especifica pendente"
	}
	return strings.Join(words, " ")
}

func matrixFieldCue(base scenario, facetID string, profile cycleProfile, limit int, salt int, fieldParts ...string) string {
	if len(fieldParts) == 0 {
		fieldParts = []string{base.ReaderProblem, base.SourceHook, base.DocumentContext, base.RiskContext, base.DigitalAction}
	}
	seed := scenarioTemplateIndex(base.IntentID + "-" + facetID + "-" + profile.ID)
	return contextCueFrom(strings.Join(fieldParts, " "), limit, seed+salt)
}

func joinDistinctContext(parts ...string) string {
	accepted := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		current := strings.Join(accepted, " ")
		if current != "" && sharesSignalNGram(current, part, 3) {
			continue
		}
		accepted = append(accepted, part)
	}
	return strings.Join(accepted, ". ")
}

func sharesSignalNGram(left string, right string, size int) bool {
	leftSet := signalNGrams(left, size)
	if len(leftSet) == 0 {
		return false
	}
	for ngram := range signalNGrams(right, size) {
		if leftSet[ngram] {
			return true
		}
	}
	return false
}

func signalNGrams(value string, size int) map[string]bool {
	words := make([]string, 0)
	for _, word := range normalizedWords(value) {
		if len(word) > 3 && !localStopWord(word) {
			words = append(words, word)
		}
	}
	ngrams := make(map[string]bool)
	for i := 0; i+size <= len(words); i++ {
		ngrams[strings.Join(words[i:i+size], " ")] = true
	}
	return ngrams
}

func localStopWord(word string) bool {
	return localStopWords[word]
}

func normalizedWords(value string) []string {
	return strings.Fields(strings.Map(func(r rune) rune {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			return unicode.ToLower(r)
		}
		return ' '
	}, value))
}

func uppercaseFirst(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return value
	}
	runes := []rune(value)
	runes[0] = unicode.ToUpper(runes[0])
	return string(runes)
}

func scenarioTemplateIndex(intentID string) int {
	sum := 0
	for _, r := range intentID {
		sum += int(r)
	}
	return sum % 5
}

func variantReaderProblem(template int, facetID string, area string, readerFocus string, base scenario, profile cycleProfile) string {
	lead := readerLead(facetID)
	context := areaContext(area)
	facetOffset := scenarioTemplateIndex(facetID)
	style := (template + facetOffset) % 5
	baseProblem := matrixFieldCue(base, facetID, profile, 5, template+facetOffset, base.ReaderProblem)
	if profile.ID != "" {
		cue := matrixFieldCue(base, facetID, profile, 5, template+facetOffset, base.ReaderProblem)
		switch style {
		case 0:
			return lead + " em " + context + " trata " + readerFocus + ". A rodada nova muda o eixo para " + profile.TermContext + " e preserva pista concreta: " + cue + "."
		case 1:
			return lead + " parte da pergunta do usuario e testa " + readerFocus + ". O recorte de " + profile.TermContext + " usa estes sinais do caso matriz: " + cue + "."
		case 2:
			return "Neste subtema, " + lead + " nao basta como etiqueta. A leitura cruza " + readerFocus + " com pista material do caso: " + cue + "."
		case 3:
			return lead + " organiza a duvida antes do CTA. O perfil " + profile.TermContext + " so avanca quando conversa com sinais concretos: " + cue + "."
		default:
			return lead + " em " + context + " separa relato e prova. A expansao usa " + profile.TermContext + " com pista editorial especifica: " + cue + "."
		}
	}
	switch style {
	case 0:
		return lead + " no contexto de " + context + ": o ponto central e " + readerFocus + ", conectado ao problema original do leitor: " + baseProblem
	case 1:
		return lead + " em " + context + " parte do fato narrado pelo usuario e confere " + readerFocus + ". Base concreta: " + baseProblem
	case 2:
		return lead + " nao pode virar frase generica; neste subtema, a leitura exige " + readerFocus + " e compara o relato com a situacao documentada: " + baseProblem
	case 3:
		return lead + " torna o recorte de " + context + " util quando mostra " + readerFocus + " e separa o que ja esta provado do que ainda falta: " + baseProblem
	default:
		return lead + " antes de qualquer CTA precisa explicar " + readerFocus + " em linguagem direta para quem vive este problema: " + baseProblem
	}
}

func variantSourceHook(template int, facetID string, sourceFocus string, base scenario, profile cycleProfile) string {
	facetOffset := scenarioTemplateIndex(facetID)
	style := (template + facetOffset) % 5
	if profile.ID != "" {
		cue := matrixFieldCue(base, facetID, profile, 5, template+facetOffset+11, base.SourceHook)
		switch style {
		case 0:
			return sourceFocus + " Fonte do subtema: " + cue + "."
		case 1:
			return "Autoridade juridica do recorte: " + cue + ". " + sourceFocus
		case 2:
			return sourceFocus + " Sinais de autoridade: " + cue + "."
		case 3:
			return sourceFocus + " Recorte de fonte exigido: " + cue + "."
		default:
			return sourceFocus + " Matriz oficial ligada a: " + cue + "."
		}
	}
	switch style {
	case 0:
		return sourceFocus + " Referencia bloqueada: " + matrixFieldCue(base, facetID, profile, 5, template+facetOffset+11, base.SourceHook) + "."
	case 1:
		return "Autoridade do rascunho: " + sourceFocus + " Pista: " + matrixFieldCue(base, facetID, profile, 5, template+facetOffset+11, base.SourceHook) + "."
	case 2:
		return sourceFocus + " Uso editorial proprio, sem espelhar fonte. Pista: " + matrixFieldCue(base, facetID, profile, 5, template+facetOffset+11, base.SourceHook) + "."
	case 3:
		return sourceFocus + " Auditoria bloqueada com pista: " + matrixFieldCue(base, facetID, profile, 5, template+facetOffset+11, base.SourceHook) + "."
	default:
		return sourceFocus + " Rascunho proprio noindex; pista: " + matrixFieldCue(base, facetID, profile, 5, template+facetOffset+11, base.SourceHook) + "."
	}
}

func variantDocumentContext(template int, facetID string, documentFocus string, base scenario, profile cycleProfile) string {
	facetOffset := scenarioTemplateIndex(facetID)
	style := (template + facetOffset) % 5
	baseDocument := matrixFieldCue(base, facetID, profile, 5, template+facetOffset+23, base.DocumentContext)
	if profile.ID != "" {
		cue := matrixFieldCue(base, facetID, profile, 5, template+facetOffset+23, base.DocumentContext)
		switch style {
		case 0:
			return documentFocus + " Pista documental propria: " + cue + "."
		case 1:
			return documentFocus + " Arquivo central da rodada: " + cue + "."
		case 2:
			return documentFocus + " Sinais documentais: " + cue + "."
		case 3:
			return documentFocus + " Pista de fase e arquivo: " + cue + "."
		default:
			return documentFocus + " Documento que altera analise: " + cue + "."
		}
	}
	if sharesSignalNGram(documentFocus, baseDocument, 3) {
		baseDocument = "O caso matriz e resumido por fase, periodo e lacuna documental, sem repetir a mesma sequencia de arquivos."
	}
	switch style {
	case 0:
		return documentFocus + " Recorte documental: " + baseDocument + "."
	case 1:
		return documentFocus + " Arquivos relevantes: " + baseDocument + "."
	case 2:
		return documentFocus + " Diferença pratica: " + baseDocument + "."
	case 3:
		return documentFocus + " Referencia documental: " + baseDocument + "."
	default:
		return documentFocus + " Analise documental parte de: " + baseDocument + "."
	}
}

func variantRiskContext(template int, facetID string, riskFocus string, base scenario, profile cycleProfile) string {
	facetOffset := scenarioTemplateIndex(facetID)
	style := (template + facetOffset) % 5
	baseRisk := matrixFieldCue(base, facetID, profile, 5, template+facetOffset+37, base.RiskContext)
	if profile.ID != "" {
		cue := matrixFieldCue(base, facetID, profile, 5, template+facetOffset+37, base.RiskContext)
		switch style {
		case 0:
			return riskFocus + " Pista concreta: " + cue + "."
		case 1:
			return riskFocus + " Conferencia exigida: " + cue + "."
		case 2:
			return riskFocus + " Sinais de prejuizo ou suposicao: " + cue + "."
		case 3:
			return riskFocus + " Diferenca juridica: " + cue + "."
		default:
			return riskFocus + " Pista de decisao informada: " + cue + "."
		}
	}
	switch style {
	case 0:
		return riskFocus + " Risco concreto: " + baseRisk + "."
	case 1:
		return riskFocus + " Cautela do caso: " + baseRisk + "."
	case 2:
		return riskFocus + " Prejuizo ou suposicao: " + baseRisk + "."
	case 3:
		return riskFocus + " Limite juridico: " + baseRisk + "."
	default:
		return riskFocus + " Decisao informada considera: " + baseRisk + "."
	}
}

func variantDigitalAction(template int, facetID string, digitalFocus string, base scenario, profile cycleProfile) string {
	facetOffset := scenarioTemplateIndex(facetID)
	style := (template + facetOffset) % 5
	if profile.ID != "" {
		cue := matrixFieldCue(base, facetID, profile, 5, template+facetOffset+51, base.DigitalAction)
		switch style {
		case 0:
			return digitalFocus + " WhatsApp contextual: " + cue + "."
		case 1:
			return digitalFocus + " Contexto remoto: " + cue + "."
		case 2:
			return digitalFocus + " CTA separado preserva: " + cue + "."
		case 3:
			return digitalFocus + " Triagem usa: " + cue + "."
		default:
			return digitalFocus + " Contexto rastreavel: " + cue + "."
		}
	}
	switch style {
	case 0:
		return digitalFocus + " WhatsApp contextual: " + matrixFieldCue(base, facetID, profile, 5, template+facetOffset+51, base.DigitalAction) + "."
	case 1:
		return digitalFocus + " Atendimento remoto recebe: " + matrixFieldCue(base, facetID, profile, 5, template+facetOffset+51, base.DigitalAction) + "."
	case 2:
		return digitalFocus + " CTA separado com pista: " + matrixFieldCue(base, facetID, profile, 5, template+facetOffset+51, base.DigitalAction) + "."
	case 3:
		return digitalFocus + " Triagem digital usa: " + matrixFieldCue(base, facetID, profile, 5, template+facetOffset+51, base.DigitalAction) + "."
	default:
		return digitalFocus + " Origem rastreavel: " + matrixFieldCue(base, facetID, profile, 5, template+facetOffset+51, base.DigitalAction) + "."
	}
}

func scenarioCycleProfile(area string, cycle int) cycleProfile {
	if cycle <= 0 {
		return cycleProfile{}
	}
	profiles := []cycleProfile{
		{
			ID:            "complemento-posterior-anexo-corrigido",
			TermContext:   "documento complementar posterior",
			ReaderFocus:   "comparar arquivo novo, resposta posterior e documento inicial",
			SourceFocus:   "Etapa posterior precisa de apoio proprio, nao apenas do direito em abstrato.",
			DocumentFocus: "Comprovante de reenvio, anexo corrigido e comunicacao mais recente entram como camada propria.",
			RiskFocus:     "Repetir pedido sem corrigir lacuna ja apontada no protocolo aumenta o risco.",
			DigitalFocus:  "Triagem remota separa arquivo inicial, complemento enviado depois e resposta recebida.",
		},
		{
			ID:            "recurso-decisao-recorrida-prova-nova",
			TermContext:   "fase recursal documentada",
			ReaderFocus:   "distinguir pedido inicial, recurso, revisao e resposta complementar",
			SourceFocus:   "Rito, prazo ou autoridade da etapa recursal precisam aparecer na fonte.",
			DocumentFocus: "Razoes do recurso, decisao recorrida, protocolo e prova nova ficam em trilha separada.",
			RiskFocus:     "Tratar recurso como simples novo pedido pode fazer a pessoa perder prazo.",
			DigitalFocus:  "Fluxo remoto marca fase recursal, prazo, documento novo e pergunta juridica objetiva.",
		},
		{
			ID:            "divergencia-cadastral-base-antiga",
			TermContext:   "erro cadastral verificavel",
			ReaderFocus:   "separar erro de cadastro, documento faltante e divergencia de versoes",
			SourceFocus:   "Cadastro, registro, contrato ou base administrativa precisam orientar a fonte oficial.",
			DocumentFocus: "Historico cadastral, comprovante antigo e dado divergente explicam o recorte.",
			RiskFocus:     "Pedir providencia errada quando o problema e acerto de base atrasa a solucao.",
			DigitalFocus:  "Conferencia digital lista campo divergente, prova do dado correto e canal ja usado.",
		},
		{
			ID:            "recorrencia-prejuizo-continuado-valores-repetidos",
			TermContext:   "impacto continuado comprovado",
			ReaderFocus:   "medir recorrencia, continuidade do prejuizo e evento que tornou o caso urgente",
			SourceFocus:   "Prazo, dever de resposta ou criterio objetivo de continuidade delimitam a fonte.",
			DocumentFocus: "Extratos, historico de pagamentos, comunicados e datas repetidas mostram continuidade.",
			RiskFocus:     "Urgencia generica sem prejuizo continuado demonstrado enfraquece a analise.",
			DigitalFocus:  "Roteiro online organiza recorrencia, valores, datas e documento que muda a prioridade.",
		},
	}
	profile := profiles[(cycle-1)%len(profiles)]
	if area == "previdenciario" && profile.ID == "divergencia-cadastral-base-antiga" {
		profile.TermContext = "divergencia no CNIS ou cadastro previdenciario"
		profile.ReaderFocus = "separar erro de CNIS, vinculo faltante, laudo incompleto e fase do requerimento"
		profile.SourceFocus = "Cadastro, recurso ou exigencia concreta precisam aparecer na fonte do INSS ou Previdencia."
		profile.DocumentFocus = "CNIS antigo, carteira, PPP, guia, comunicado de decisao e protocolo mostram a divergencia."
		profile.RiskFocus = "Protocolar beneficio com base cadastral incompleta pode gerar indeferimento evitavel."
		profile.DigitalFocus = "Conferencia remota separa acerto de CNIS, recurso, novo pedido e possivel analise judicial."
	}
	return profile
}

func areaContext(area string) string {
	switch area {
	case "saude-suplementar":
		return "saude suplementar digital"
	case "trabalhista":
		return "rotina trabalhista online"
	case "familia":
		return "familia e acordos remotos"
	case "previdenciario":
		return "beneficio previdenciario digital"
	case "consumidor-financeiro":
		return "consumo financeiro online"
	case "sucessorio":
		return "inventario e sucessao digital"
	default:
		return "servico juridico digital"
	}
}

func areaSourceContext(area string) string {
	switch area {
	case "saude-suplementar":
		return "ANS, contrato do plano e lei de saude suplementar ficam separados como autoridade, sem promessa de cobertura."
	case "trabalhista":
		return "CLT, prova de jornada e comunicacoes do emprego sustentam a leitura antes de qualquer calculo."
	case "familia":
		return "Codigo Civil, CNJ e documentos familiares orientam consenso, guarda, alimentos ou partilha."
	case "previdenciario":
		return "INSS, CNIS, laudos e comunicados administrativos definem a fase do pedido."
	case "consumidor-financeiro":
		return "Banco Central, consumidor.gov.br, contrato e extratos ajudam a separar fraude, erro e cobranca."
	case "sucessorio":
		return "Codigo Civil, CNJ, cartorio, certidoes e bens do espolio delimitam a via possivel."
	default:
		return "Fonte oficial, documento do usuario e trilha de auditoria delimitam a pauta."
	}
}

func areaDocumentContext(area string) string {
	switch area {
	case "saude-suplementar":
		return "Relatorio medico, negativa da operadora, carteirinha, contrato e protocolo entram conforme urgencia."
	case "trabalhista":
		return "Holerites, ponto, escala, mensagens e termo rescisorio organizam a prova laboral."
	case "familia":
		return "Certidoes, renda, despesas, calendario combinado e minuta familiar mostram o impacto concreto."
	case "previdenciario":
		return "CNIS, comunicado de decisao, laudos, atestados e protocolo indicam a etapa administrativa."
	case "consumidor-financeiro":
		return "Extratos, contrato, comprovante Pix, fatura, protocolo e resposta do banco apontam responsabilidade."
	case "sucessorio":
		return "Obito, herdeiros, matricula, extratos, imposto, testamento ou divida definem o caminho."
	default:
		return "Documento pessoal, protocolo, decisao e comprovantes sustentam o recorte."
	}
}

func areaRiskContext(area string) string {
	switch area {
	case "saude-suplementar":
		return "Erro de urgencia ou cobertura pode prejudicar prova clinica e estrategia."
	case "trabalhista":
		return "Narrativa sem periodo, funcao ou prova digital enfraquece a analise."
	case "familia":
		return "Acordo aparente pode esconder conflito sobre filhos, renda ou bens."
	case "previdenciario":
		return "Prazo, qualidade de segurado e conexao entre laudo e trabalho precisam ser conferidos."
	case "consumidor-financeiro":
		return "Fraude, contratacao valida e erro cadastral exigem separacao documental."
	case "sucessorio":
		return "Bens, dividas, consenso e testamento mudam a via e nao podem ser presumidos."
	default:
		return "Fonte generica ou prova incompleta mantem o lote bloqueado."
	}
}

func areaDigitalContext(area string) string {
	switch area {
	case "saude-suplementar":
		return "Triagem online prioriza arquivos medicos, negativa e prazo assistencial."
	case "trabalhista":
		return "Atendimento remoto organiza periodo, empregador, verbas e prints preservados."
	case "familia":
		return "Fluxo digital separa consenso, documento dos filhos, renda e pontos pendentes da minuta."
	case "previdenciario":
		return "Analise remota classifica requerimento, recurso, exigencia ou novo pedido."
	case "consumidor-financeiro":
		return "Consulta online registra instituicao, valor, data, protocolo e impacto financeiro."
	case "sucessorio":
		return "Triagem sucessoria digital lista herdeiros, bens, dividas e documento faltante."
	default:
		return "Atendimento digital organiza fonte, arquivos e etapa juridica."
	}
}

func compactContext(value string) string {
	value = strings.TrimSpace(value)
	if len(value) <= 150 {
		return value
	}
	cut := strings.LastIndex(value[:150], " ")
	if cut < 80 {
		cut = 150
	}
	return strings.TrimSpace(value[:cut]) + "."
}

func readerLead(facetID string) string {
	leads := map[string]string{
		"prova-digital":            "Prova digital preservada",
		"prazo-e-urgencia":         "Prazo e urgencia documentados",
		"fonte-primaria":           "Fonte primaria conferida",
		"documento-minimo":         "Documento minimo para analise",
		"negociacao-previa":        "Historico de tentativa previa",
		"risco-economico":          "Impacto financeiro mensuravel",
		"vulnerabilidade":          "Vulnerabilidade concreta",
		"competencia-digital":      "Caminho digital definido",
		"linha-do-tempo":           "Linha do tempo organizada",
		"prova-de-negativa":        "Negativa formal analisada",
		"parte-responsavel":        "Responsavel juridico identificado",
		"estado-do-processo":       "Situacao atual delimitada",
		"prova-medica-ou-tecnica":  "Prova tecnica contextual",
		"conflito-de-versoes":      "Conflito de versoes mapeado",
		"custo-de-inercia":         "Consequencia de inercia explicada",
		"rota-administrativa":      "Rota administrativa separada",
		"prova-patrimonial":        "Prova patrimonial objetiva",
		"impacto-familiar":         "Impacto familiar delimitado",
		"evidencia-de-boa-fe":      "Boa-fe documentada",
		"lacuna-de-fonte":          "Lacuna de fonte bloqueante",
		"risco-clinico":            "Risco clinico comprovado",
		"rotina-de-trabalho":       "Rotina de trabalho demonstrada",
		"historico-previdenciario": "Historico previdenciario organizado",
	}
	if lead := leads[facetID]; lead != "" {
		return lead
	}
	for _, facet := range allSemanticFacets() {
		if strings.HasPrefix(facetID, facet.ID+"-") {
			if lead := leads[facet.ID]; lead != "" {
				return lead
			}
		}
	}
	if index := strings.LastIndex(facetID, "-rodada-"); index > 0 {
		if lead := leads[facetID[:index]]; lead != "" {
			return lead
		}
	}
	return "Contexto juridico especifico"
}

var localStopWords = map[string]bool{
	"para": true, "pela": true, "pelo": true, "pelas": true, "pelos": true, "quando": true, "sobre": true,
	"documento": true, "documentos": true, "fonte": true, "fontes": true, "oficial": true, "oficiais": true,
	"juridico": true, "jurídico": true, "juridica": true, "jurídica": true, "digital": true, "online": true,
	"triagem": true, "contexto": true, "caso": true, "subtema": true, "leitor": true, "rascunho": true,
	"prova": true, "provas": true, "data": true, "datas": true, "resposta": true, "protocolo": true,
}

func allSemanticFacets() []semanticFacet {
	facets := append([]semanticFacet{}, semanticFacetsFor("")...)
	facets = append(facets,
		semanticFacet{ID: "risco-clinico"},
		semanticFacet{ID: "rotina-de-trabalho"},
		semanticFacet{ID: "historico-previdenciario"},
	)
	return facets
}

func semanticFacetsFor(area string) []semanticFacet {
	common := []semanticFacet{
		{"prova-digital", "com prova digital preservada", "separar prints, protocolos e arquivos enviados em aplicativos", "A checagem prioriza documento oficial que confirme o canal e a data do evento.", "prints, e-mails, protocolo, comprovante de envio e resposta formal precisam mostrar continuidade", "a narrativa ficar forte no relato e fraca na prova verificavel", "o sistema agrupa arquivos por data, identifica lacunas e nao libera publicacao se a origem estiver incompleta", "prova"},
		{"prazo-e-urgencia", "com prazo e urgencia documentados", "diferenciar urgencia real, prazo administrativo e demora comum", "O apoio oficial deve explicar prazo, competencia ou rito antes de qualquer interpretacao editorial.", "datas de pedido, resposta, vencimento, agenda e comunicacao indicam se ha urgencia concreta", "perder prazo ou tratar todo atraso como ilegal sem fonte e cronologia", "a triagem calcula janela temporal, registra alerta e preserva pergunta objetiva para advogado online", "prazo"},
		{"fonte-primaria", "com fonte primaria conferida", "confirmar se a fonte primaria cobre exatamente o subtema", "A verificacao separa lei, regulador, orgao publico e tribunal, sem copiar texto de nenhum deles.", "identificacao da norma, protocolo, decisao, contrato e documento do usuario sustenta a diferenca do tema", "usar fonte generica para criar pagina parecida com outras do lote", "o laboratorio bloqueia rascunho sem matriz oficial e sem vinculo de fonte URL-a-URL", "fonte"},
		{"documento-minimo", "com documento minimo para analise remota", "entender quais documentos mudam a conclusao juridica", "A fonte oficial funciona como ancora de autoridade, mas o texto precisa explicar o problema do usuario.", "contrato, comprovante, mensagem, resposta, identificacao das partes e documento publico formam o conjunto minimo", "publicar orientacao sem documento que confirme fato, data ou responsabilidade", "o fluxo pede os arquivos essenciais e gera mensagem de WhatsApp com origem rastreavel", "documento"},
		{"negociacao-previa", "com historico de tentativa previa", "mostrar o que foi pedido antes de contratar advogado", "A leitura considera canais administrativos e registros formais antes de falar em medida judicial.", "protocolos, respostas, recusas, datas e comprovantes de tentativa revelam se falta etapa util", "pular tentativa relevante ou prometer processo sem maturidade documental", "a triagem separa tentativa administrativa, urgencia e prova para uma avaliacao digital proporcional", "negociacao"},
		{"risco-economico", "com impacto financeiro mensuravel", "identificar valor, recorrencia, desconto, custo ou perda pratica", "A matriz precisa apontar fonte que ajude a qualificar o impacto, nao apenas o nome do direito.", "extratos, notas, boletos, recibos, holerites ou comprovantes indicam o tamanho do problema", "gerar texto abstrato sem explicar o prejuizo documentado", "o atendimento remoto registra valores e recorrencia sem fazer promessa comercial", "valor"},
		{"vulnerabilidade", "com vulnerabilidade concreta descrita", "avaliar idade, saude, renda, dependencia ou urgencia familiar quando isso altera a prioridade", "A fonte oficial deve sustentar o criterio objetivo ligado ao subtema.", "laudos, comprovantes de renda, despesas, certidoes e registros familiares precisam aparecer quando forem relevantes", "usar tom emocional sem prova ou transformar todo caso em urgencia", "o sistema marca prioridade documental e exige revisao antes de qualquer rota publica", "prioridade"},
		{"competencia-digital", "com caminho digital definido", "entender se o caso pode ser conduzido de forma totalmente online", "A verificacao observa orgao, canal digital, documento aceito e limite do atendimento remoto.", "assinaturas, documentos de identidade, protocolos eletronicos e comprovantes digitais sustentam o fluxo", "prometer solucao presencial ou cartorial quando a pauta e 100 por cento digital", "o CTA recebe contexto do canal digital, arquivos esperados e motivo da consulta", "digital"},
		{"linha-do-tempo", "com linha do tempo organizada", "montar sequencia de fatos antes de escolher a providencia", "A fonte valida prazos, competencias ou conceitos usados na leitura dos eventos.", "evento inicial, pedido, resposta, agravamento, tentativa e documento final precisam ficar em ordem", "confundir fato antigo, resposta recente e prazo ainda aberto", "a triagem transforma datas em roteiro de avaliacao sem publicar conteudo bruto", "cronologia"},
		{"prova-de-negativa", "com negativa formal analisada", "distinguir recusa expressa, silencio, exigencia e resposta incompleta", "A fonte de apoio precisa permitir leitura da negativa dentro do subtema.", "carta, e-mail, protocolo, print, decisao ou comunicacao da empresa mostra o tipo de negativa", "tratar qualquer silencio como recusa definitiva sem documento", "o atendimento digital classifica a negativa e prepara pergunta contextual para WhatsApp", "negativa"},
		{"parte-responsavel", "com responsavel identificado", "saber quem deve responder pelo problema antes de sugerir caminho juridico", "A matriz de fonte ajuda a separar orgao publico, empresa, banco, operadora, empregador ou familia.", "contrato, cadastro, identificacao, comunicacao e comprovante ligam o fato a quem respondeu", "acusar parte errada ou criar texto generico sem sujeito juridico", "o fluxo registra responsavel, origem e fonte antes de qualquer avaliacao remota", "responsavel"},
		{"estado-do-processo", "com situacao atual delimitada", "separar consulta inicial, recurso, revisao, cumprimento, acordo ou urgencia", "A fonte oficial deve servir ao estado real do caso e nao apenas ao tema amplo.", "decisao, despacho, resposta administrativa, acordo, contrato ou protocolo mostram a fase", "misturar fases e criar pagina que compete com outro subtema", "a triagem classifica a etapa e bloqueia publicacao quando houver conflito de intencao", "fase"},
		{"prova-medica-ou-tecnica", "com prova tecnica contextual", "conectar laudo, relatorio, exame, pericia, parecer ou documento tecnico ao direito discutido", "A fonte e usada para conferir se o documento tecnico conversa com regra, prazo ou cobertura.", "laudo, exame, relatorio, receita, parecer, vistoria ou demonstrativo precisa indicar fato verificavel", "transformar opiniao tecnica isolada em conclusao juridica pronta", "o atendimento remoto pede arquivo tecnico e registra lacuna antes de recomendar proximo passo", "tecnica"},
		{"conflito-de-versoes", "com conflito de versoes mapeado", "identificar divergencia entre relato do usuario, documento da outra parte e fonte oficial", "A pesquisa de fonte deve ajudar a explicar o ponto que esta em disputa.", "mensagens, resposta escrita, contrato, comprovantes e historico mostram onde as versoes divergem", "publicar narrativa unilateral sem indicar prova minima", "a triagem aponta a divergencia principal e mantem o lote bloqueado ate revisao", "conflito"},
		{"custo-de-inercia", "com consequencia de inercia explicada", "mostrar o que pode piorar se o usuario nao organizar a prova agora", "A fonte oficial delimita prazo ou criterio objetivo que torna a inercia relevante.", "datas, cobrancas, comunicados, agenda, vencimentos e protocolos demonstram risco de demora", "usar medo generico como CTA em vez de informacao util", "o WhatsApp recebe motivo contextual, sem promessa e com foco em documento verificavel", "inercia"},
		{"rota-administrativa", "com rota administrativa separada", "distinguir reclamacao administrativa, recurso, pedido novo e medida judicial", "A fonte prioriza canais oficiais e limites da via administrativa.", "protocolo, canal usado, resposta, comprovante de envio e pendencia indicam se a etapa existe", "ajuizar mentalmente toda demanda sem checar caminho simples e documentado", "a triagem marca etapa administrativa e identifica quando a consulta juridica online faz sentido", "administrativo"},
		{"prova-patrimonial", "com prova patrimonial objetiva", "mapear bens, valores, renda, descontos, dividas ou patrimonio envolvido", "A fonte de apoio precisa conversar com registro, contrato, extrato ou documento economico.", "matricula, extrato, holerite, imposto, contrato, fatura ou comprovante financeiro sustentam o tema", "confundir suspeita patrimonial com prova aproveitavel", "o atendimento digital organiza valores e documentos antes de qualquer roteiro juridico", "patrimonio"},
		{"impacto-familiar", "com impacto familiar delimitado", "entender quando filhos, dependentes, herdeiros ou familiares mudam a leitura do caso", "A matriz oficial deve sustentar o recorte familiar sem generalizar.", "certidoes, despesas, calendario, residencia, renda e comunicacoes mostram a situacao concreta", "criar texto emocional sem criterio juridico e documental", "o fluxo remoto registra pessoas afetadas e documentos antes de CTA contextual", "familia"},
		{"evidencia-de-boa-fe", "com boa-fe documentada", "provar tentativa correta, comunicacao transparente e preservacao de comprovantes", "A fonte ajuda a separar conduta documentada de relato incompleto.", "mensagens, protocolos, recibos, comprovantes e respostas demonstram cooperação ou resistencia", "ignorar documento que mostra tentativa previa ou aceitar print solto como prova total", "a triagem identifica boa-fe e lacunas para avaliacao juridica remota", "boafe"},
		{"lacuna-de-fonte", "com lacuna de fonte bloqueante", "reconhecer quando o tema ainda nao tem fonte especifica suficiente para pagina publica", "A validacao URL-a-URL decide se a matriz esta pronta apenas como referencia bloqueada.", "sem fonte especifica, documentos e contexto ficam em laboratorio, nunca em sitemap", "preencher vazio com texto bonito e sem autoridade juridica", "o gerador marca bloqueio e exige nova pesquisa antes de escalar", "bloqueio"},
	}
	if area == "saude-suplementar" {
		return append(common, semanticFacet{"risco-clinico", "com risco clinico comprovado", "ligar urgencia medica a negativa, prazo e documento assistencial", "A fonte da ANS precisa conversar com cobertura, prazo ou resposta da operadora.", "relatorio medico, pedido, exame, carteirinha, contrato e negativa formam o eixo clinico", "tratar ansiedade como urgencia clinica sem relatorio", "a triagem destaca risco clinico e prova medica para consulta remota", "clinico"})
	}
	if area == "trabalhista" {
		return append(common, semanticFacet{"rotina-de-trabalho", "com rotina laboral demonstrada", "provar jornada, comando, salario, funcao ou ruptura do contrato", "A CLT e a fonte trabalhista orientam o recorte, mas a rotina documentada decide a utilidade.", "ponto, escala, holerite, mensagem, advertencia e contrato mostram a pratica", "confundir desconforto comum com violacao provada", "o atendimento organiza rotina e documentos para analise trabalhista online", "rotina"})
	}
	if area == "previdenciario" {
		return append(common, semanticFacet{"historico-previdenciario", "com historico previdenciario organizado", "ligar CNIS, pericia, exigencia, laudo ou beneficio ao momento certo", "A fonte do INSS orienta fase administrativa e documento esperado.", "CNIS, comunicado, protocolo, laudo, atestado e carteira mostram continuidade", "perder fase administrativa ou discutir incapacidade sem historico", "a triagem separa requerimento, recurso, exigencia e acao possivel", "previdenciario"})
	}
	return common
}

func scenariosForArea(area string) []scenario {
	switch area {
	case "saude-suplementar":
		return []scenario{
			{"procedimento-urgente-negado", "procedimento urgente negado pelo plano de saúde", "O leitor recebeu resposta negativa da operadora para cirurgia indicada como urgente e precisa separar contrato, relatório médico, protocolo, segmentação do plano, prazo da ANS e risco clínico antes de agir.", "A pauta usa Lei 9656, Rol da ANS, regra de cobertura assistencial, canais oficiais de reclamação e o motivo escrito pela operadora.", "A triagem pede pedido médico, relatório clínico, resposta formal, protocolo, contrato, carteirinha, exames recentes e mensagens do aplicativo.", "O risco é a demora agravar o quadro ou o usuário aceitar justificativa incompleta sem preservar prova da urgência e da negativa.", "O atendimento digital organiza arquivos, confirma datas, identifica fonte aplicável e prepara perguntas objetivas para avaliação jurídica remota."},
			{"medicamento-prescrito-recusado", "medicamento prescrito recusado pelo plano", "A família tem prescrição atual, recebeu recusa do plano e precisa entender indicação médica, cobertura, Rol da ANS, orçamento, contrato e justificativa técnica da negativa.", "A análise parte das normas da ANS, legislação dos planos, atualização do Rol, protocolo administrativo e resposta emitida pela operadora.", "São esperados prescrição, relatório do especialista, CID quando constar, orçamento, e-mail de recusa, protocolo e cópia do contrato.", "A cautela é não prometer fornecimento imediato e diferenciar urgência documentada, escolha terapêutica e exclusão contratual alegada.", "A triagem online confere a documentação, monta linha do tempo e orienta se falta reclamação administrativa, nova prova médica ou análise judicial."},
			{"reembolso-exame-fora-rede", "reembolso de exame fora da rede credenciada", "O usuário pagou exame porque não conseguiu agenda na rede e precisa verificar prazo de atendimento, nota fiscal, pedido médico, contrato e resposta da operadora.", "A pauta cruza canais da ANS, contrato assistencial, regra de rede credenciada, comprovante de indisponibilidade e registro de atendimento.", "A conferência exige nota fiscal, comprovante de pagamento, pedido médico, protocolo, justificativa da rede indisponível e data do exame.", "O risco é confundir escolha particular com indisponibilidade real da rede ou perder documento que comprove tentativa de atendimento.", "A análise remota recebe os arquivos, compara datas, identifica lacunas e registra origem do caso para WhatsApp com contexto."},
			{"home-care-reduzido", "home care reduzido ou suspenso pelo plano", "A família recebeu redução de assistência domiciliar e precisa organizar relatório clínico, contrato, prescrição de enfermagem, resposta da operadora e evolução do paciente.", "A fonte considera regras da ANS, legislação de planos, contrato de cobertura, relatório médico e histórico de autorização anterior.", "Devem ser enviados relatório clínico, prescrição, autorizações antigas, comunicado de redução, protocolos, contrato e exames que mostrem dependência.", "A cautela é diferenciar ajuste técnico de suspensão abusiva e evitar promessa de manutenção automática sem prova médica robusta.", "A triagem digital monta sequência de eventos, confere urgência e prepara leitura jurídica antes de medida administrativa ou judicial."},
			{"prazo-consulta-especialista", "prazo excessivo para consulta com especialista", "O beneficiário tenta agendar especialista, recebe data distante e precisa provar tentativas, protocolo, rede disponível, urgência e prazo máximo aplicável.", "A análise usa prazos da ANS, canais de reclamação, contrato do plano, rede credenciada e documentos médicos que indicam prioridade.", "A triagem pede protocolos, prints de agenda, pedido médico, carteirinha, contrato, resposta da central e eventual relatório clínico.", "O risco é reclamar sem demonstrar tentativa real de agendamento ou sem identificar se o caso é urgência, retorno ou consulta eletiva.", "O fluxo online organiza evidências, registra datas e prepara mensagem contextual de WhatsApp para avaliar providência proporcional."},
		}
	case "trabalhista":
		return []scenario{
			{"rescisao-indireta-assedio-salario", "rescisão indireta por assédio e atraso salarial", "O empregado relata cobrança humilhante, atraso de salário e medo de pedir demissão sem avaliar contrato, holerites, mensagens, testemunhas e datas relevantes.", "A pauta usa CLT, orientação trabalhista, documentos rescisórios, controle de jornada, prova digital e cautela sobre permanência no emprego.", "A triagem pede contrato, holerites, mensagens, advertências, comprovantes de atraso, escala, ponto e nomes de testemunhas.", "O risco é romper o vínculo sem prova suficiente ou perder verbas por uma narrativa sem datas e documentos concretos.", "O atendimento online organiza linha do tempo, preserva prints e indica perguntas para análise de reclamação trabalhista digital."},
			{"horas-extras-banco-irregular", "horas extras e compensação irregular de jornada", "O trabalhador percebe jornada maior que a registrada e precisa comparar CLT, cartões de ponto, escala, mensagens, banco de horas, holerites, contrato e períodos prescricionais.", "A análise parte da CLT, informações oficiais sobre jornada, regras de compensação, testemunhas e documentos que mostram rotina efetiva.", "São conferidos ponto, escala, mensagens, e-mails, holerites, contrato, recibos de compensação e prints preservados.", "A cautela é separar ocorrência eventual de jornada habitual e não calcular verbas sem base documental completa.", "A triagem remota recebe arquivos, agrupa períodos, revisa testemunhas e identifica se há perguntas objetivas para ação trabalhista."},
			{"justa-causa-prova-fragil", "contestação de justa causa com prova frágil", "A pessoa foi dispensada por justa causa e precisa avaliar comunicado, proporcionalidade, advertências anteriores, holerites, testemunhas e verbas retidas.", "A pauta considera CLT, documentos rescisórios, histórico disciplinar, prova empresarial e critérios de proporcionalidade.", "Devem ser enviados termo de rescisão, comunicado da falta, advertências, mensagens, controle de ponto, holerites e prova apresentada pela empresa.", "O risco é assinar documento sem ressalva, perder prazo ou ignorar prova que confirme a falta alegada.", "O fluxo digital revisa datas, arquivos e perguntas essenciais antes de orientar contestação ou negociação."},
			{"acidente-trabalho-estabilidade", "acidente de trabalho e estabilidade após afastamento", "O empregado voltou de afastamento, teme dispensa e precisa organizar CAT, laudos, atestados, benefício do INSS, contrato e comunicação da empresa.", "A análise combina CLT, documentos previdenciários, prova médica, comunicação de acidente e histórico de retorno ao trabalho.", "A triagem pede CAT, laudos, exames, atestados, comunicado de decisão do INSS, holerites, contrato e mensagens com a empresa.", "A cautela é diferenciar afastamento comum, acidente laboral e doença ocupacional sem criar promessa de reintegração automática.", "O atendimento online monta cronologia, confere benefício e registra origem para análise trabalhista e previdenciária conectada."},
			{"verbas-rescisorias-nao-pagas", "verbas rescisórias não pagas no prazo", "O ex-empregado saiu da empresa, não recebeu tudo no prazo e precisa conferir TRCT, chave do FGTS, holerites, aviso prévio e comprovantes.", "A pauta usa CLT, documentos rescisórios, guias, prazos de pagamento e prova de comunicação entre empresa e trabalhador.", "São esperados TRCT, termo de homologação quando houver, comprovante de pagamento, extrato de FGTS, holerites e mensagens.", "O risco é aceitar cálculo incompleto ou acionar sem separar atraso, diferença de valor e verba ainda controvertida.", "A triagem digital recebe documentos, organiza valores por tipo e prepara roteiro para análise jurídica remota."},
		}
	case "familia":
		return []scenario{
			{"divorcio-consensual-filhos-bens", "divórcio consensual online com filhos e bens", "O casal concorda com o divórcio, mas precisa tratar guarda, convivência, pensão, partilha de bens, renda e documentos dos filhos sem confundir cartório com processo judicial.", "A orientação considera Código Civil, regras processuais, CNJ, cartório quando cabível e diferença entre consenso formal e conflito pendente.", "A triagem pede certidão de casamento, documentos dos filhos, comprovantes de renda, acordo sobre bens, endereço das partes e despesas da criança.", "O risco é tentar via inadequada quando há filhos menores, desacordo oculto ou partilha sem documentação do patrimônio.", "O fluxo online organiza pontos de consenso, documentos e pendências para análise de advogado antes da minuta."},
			{"pensao-revisao-desemprego", "revisão de pensão alimentícia após desemprego", "Quem paga pensão perdeu renda e precisa demonstrar mudança real, sentença anterior, acordo, despesas da criança, extratos e capacidade financeira atual.", "A pauta usa legislação de família, decisões sobre alimentos, prova de renda e cautela para não estimular interrupção unilateral.", "São esperados sentença ou acordo, comprovantes de desemprego, renda atual, despesas do filho, recibos escolares e gastos médicos.", "A cautela é não parar pagamento por conta própria e diferenciar dificuldade temporária de alteração relevante para revisão.", "A triagem remota monta quadro de renda, despesas, datas e documentos para avaliar revisão, negociação ou cumprimento."},
			{"guarda-distancia-escola", "guarda e convivência quando os pais moram longe", "Os pais vivem em cidades diferentes e precisam organizar residência, escola, férias, custos de viagem, calendário de convivência e comunicação com a criança.", "A análise considera legislação de família, melhor interesse da criança, CNJ, prova da rotina familiar e acordos anteriores.", "A triagem pede certidão, comprovante de residência, calendário escolar, mensagens, despesas de deslocamento e proposta de convivência.", "O risco é transformar disputa logística em conflito maior sem demonstrar rotina, cooperação e impacto real para a criança.", "O atendimento online recebe documentos, organiza calendário e identifica pontos de acordo antes de ação ou homologação."},
			{"alimentos-avoengos", "pensão avoenga quando os pais não conseguem pagar", "A família cogita pedir alimentos aos avós e precisa avaliar renda dos pais, necessidade do filho, despesas, tentativa anterior e situação dos avós.", "A pauta parte do Código Civil, entendimento sobre obrigação complementar e prova financeira de todos os envolvidos.", "São conferidos comprovantes de renda, despesas da criança, decisão anterior, mensagens, documentos dos avós e histórico de pagamento.", "A cautela é não tratar avós como primeira opção automática nem ignorar capacidade dos pais.", "A triagem digital organiza documentos, identifica lacunas e prepara perguntas para análise de família sem promessa de valor."},
			{"partilha-bens-conta-digital", "partilha de bens com conta digital e investimentos", "No fim do casamento, uma parte suspeita de patrimônio oculto em conta digital e precisa organizar extratos, imposto de renda, bens, contrato e datas de aquisição.", "A análise considera regime de bens, Código Civil, documentos financeiros, declaração fiscal e prova de movimentação patrimonial.", "A triagem pede certidão, pacto quando houver, extratos, informes de rendimento, imposto de renda, matrícula de imóvel e comprovantes.", "O risco é alegar ocultação sem prova mínima ou deixar fora bens adquiridos no período correto.", "O atendimento online organiza ativos, datas e documentos para orientar estratégia de divórcio ou sobrepartilha."},
		}
	case "previdenciario":
		return []scenario{
			{"auxilio-incapacidade-pericia", "auxílio por incapacidade negado após perícia", "O segurado recebeu indeferimento depois da perícia e precisa comparar laudos, exames, atestados, CNIS, qualidade de segurado e atividade profissional.", "A pauta usa canais do INSS, Previdência, legislação de benefícios, comunicado de decisão e exigências documentais.", "A triagem pede decisão, laudos, exames, atestados, CNIS, carteira de trabalho, protocolos e histórico médico organizado por data.", "O risco é perder prazo de recurso ou discutir incapacidade sem demonstrar relação entre doença e trabalho exercido.", "O atendimento digital confere documentos, datas e lacunas para avaliar recurso administrativo ou ação judicial."},
			{"bpc-loas-cadunico-renda", "BPC LOAS negado por renda familiar", "A família recebeu negativa do BPC e precisa revisar CadÚnico, composição familiar, laudos, despesas de saúde, renda real e documentos da casa.", "A análise parte de regras assistenciais, INSS, Previdência, cadastro social, critérios administrativos e prova de vulnerabilidade.", "São esperados comunicado de decisão, CadÚnico, laudos, comprovantes de renda, despesas médicas, residência e documentos familiares.", "A cautela é discutir apenas renda bruta e esquecer impedimentos, gastos essenciais ou composição familiar correta.", "A triagem remota organiza documentos e perguntas para recurso, nova solicitação ou medida judicial proporcional."},
			{"cumprimento-exigencia-parado", "cumprimento de exigência do INSS sem resposta", "O segurado enviou documentos pelo Meu INSS, o pedido ficou parado e precisa comprovar protocolo, carta de exigência, envio, CNIS e histórico do benefício.", "A pauta usa canais oficiais do INSS, protocolo administrativo, prazos de análise e documento exigido no processo.", "A triagem pede print do protocolo, carta de exigência, comprovante de envio, CNIS, documentos pessoais e número do requerimento.", "O risco é tratar atraso como indeferimento ou ajuizar sem confirmar se a exigência foi realmente cumprida.", "O fluxo online revisa arquivo por arquivo, identifica pendência e orienta cobrança administrativa ou providência jurídica."},
			{"aposentadoria-cnis-incompleto", "CNIS incompleto antes de pedir aposentadoria", "O trabalhador percebe vínculos faltando no CNIS e precisa reunir carteira, contracheques, PPP, carnês, guias e documentos de empresa antiga.", "A análise parte do INSS, Previdência, cadastro de vínculos, prova de contribuição e exigências para acerto cadastral.", "São conferidos CNIS, carteira de trabalho, holerites, guias, contratos, PPP, certidões e protocolos anteriores.", "A cautela é protocolar aposentadoria com tempo incompleto e receber indeferimento evitável por falta de acerto prévio.", "A triagem digital organiza vínculos, identifica lacunas e prepara lista documental para correção administrativa."},
			{"beneficio-cessado-laudo", "benefício cessado mesmo com laudo atualizado", "O beneficiário teve auxílio cessado, ainda possui laudo recente e precisa entender perícia, decisão, exames, atestados e atividade laboral.", "A pauta considera INSS, Previdência, comunicação de cessação, prova médica e diferença entre incapacidade parcial, temporária e permanente.", "A triagem pede comunicado, laudos, exames, atestados, receitas, CNIS, função exercida e protocolos de recurso.", "O risco é perder prazo ou apresentar laudo sem conexão clara com limitações no trabalho.", "O atendimento online monta linha do tempo e avalia se cabe recurso, novo pedido ou processo."},
		}
	case "consumidor-financeiro":
		return []scenario{
			{"negativacao-divida-desconhecida", "negativação indevida por dívida desconhecida", "O consumidor descobriu restrição no cadastro e precisa guardar consulta, contrato inexistente ou contestado, protocolos, extratos e resposta do credor.", "A pauta usa defesa do consumidor, Banco Central, canais oficiais, prova documental e histórico de cobrança.", "A triagem pede consulta do cadastro, faturas, contrato se apresentado, protocolos, e-mails, boletim quando houver fraude e extratos.", "O risco é prometer indenização sem separar fraude, erro cadastral, dívida prescrita ou contratação válida.", "O atendimento digital organiza credor, datas, documentos e urgência comercial para análise jurídica remota."},
			{"consignado-nao-reconhecido", "empréstimo consignado não reconhecido", "A pessoa encontrou desconto em benefício ou salário e precisa verificar contrato, depósito, autorização, extratos, protocolos e contestação administrativa.", "A análise cruza Banco Central, INSS quando houver benefício, regras de consumidor e prova bancária do desconto.", "São esperados extrato, contrato apresentado pelo banco, comprovante de depósito, benefício, protocolos e reclamação feita.", "A cautela é diferenciar fraude, contratação eletrônica contestada, refinanciamento e valor efetivamente recebido.", "A triagem remota confere documentos, banco envolvido e datas antes de avaliar cancelamento, devolução ou demanda."},
			{"pix-fraude-resposta-banco", "fraude via Pix e resposta insuficiente do banco", "O cliente sofreu golpe, comunicou o banco e precisa organizar horários, comprovante Pix, protocolo, boletim, resposta da instituição e linha do tempo.", "A pauta considera Banco Central, regras de segurança, comunicação rápida, boletim de ocorrência e prova digital preservada.", "A triagem pede comprovante Pix, extrato, protocolo, conversa com fraudador, boletim, resposta do banco e horário dos contatos.", "O risco é prometer ressarcimento sem avaliar rapidez da comunicação, falha de segurança e conduta do usuário.", "O atendimento online monta cronologia, confere documentos e avalia reclamação administrativa ou medida jurídica cabível."},
			{"cartao-cobranca-nao-reconhecida", "cobrança não reconhecida no cartão", "O consumidor viu compras desconhecidas na fatura e precisa preservar cartão, contestação, protocolos, extratos, bloqueio e resposta da administradora.", "A análise usa Banco Central, defesa do consumidor, regras de contestação, prova de comunicação e documentos da fatura.", "São conferidos fatura, extrato, protocolo, boletim quando houver, e-mails, comprovante de bloqueio e resposta da instituição.", "A cautela é separar fraude, compra recorrente esquecida, contestação fora do prazo e falha de segurança.", "A triagem digital organiza eventos e documentos para avaliar providência administrativa ou judicial."},
			{"tarifa-bancaria-indevida", "tarifa bancária cobrada sem contratação clara", "O cliente identifica tarifa recorrente e precisa conferir contrato, cesta de serviços, extratos, protocolos, histórico de contratação e resposta do banco.", "A pauta usa Banco Central, canais oficiais, regras de relacionamento bancário e prova documental da cobrança.", "A triagem pede extratos, contrato, prints do aplicativo, protocolos, resposta do banco e histórico de alteração de pacote.", "O risco é transformar cobrança contratada em disputa sem prova ou perder repetição mensal relevante.", "O atendimento online recebe documentos, calcula período e prepara perguntas para análise de direito do consumidor."},
		}
	case "sucessorio":
		return []scenario{
			{"inventario-extrajudicial-consenso", "inventário extrajudicial com herdeiros concordes", "A família quer resolver inventário em cartório e precisa confirmar consenso, certidão de óbito, herdeiros, bens, dívidas, imposto e eventual testamento.", "A análise usa Código Civil, regras notariais, CNJ, e-Notariado quando cabível e limites do inventário extrajudicial.", "A triagem pede certidão de óbito, documentos dos herdeiros, matrícula de imóvel, extratos, dívidas, imposto e dados do cartório.", "O risco é prometer prazo curto quando há conflito, incapaz, testamento, dívida complexa ou documento faltante.", "O atendimento digital organiza bens, herdeiros e pendências para avaliar caminho cartorial ou judicial."},
			{"imovel-financiado-partilha", "partilha de imóvel financiado no inventário", "Os herdeiros precisam dividir imóvel financiado e entender saldo devedor, contrato bancário, matrícula, imposto, espólio e acordo de partilha.", "A pauta considera legislação sucessória, registro de imóveis, contrato de financiamento, regras notariais e documentos do bem.", "São esperados matrícula, contrato bancário, saldo devedor, certidão de óbito, documentos dos herdeiros e comprovantes de imposto.", "A cautela é ignorar garantia, dívida ou divergência entre herdeiros e criar partilha inviável.", "A triagem remota revisa documentos, organiza bens e dívidas e indica perguntas para inventário judicial ou extrajudicial."},
			{"alvara-valores-bancarios", "alvará para levantar valores bancários de falecido", "A família encontrou saldo pequeno e precisa saber se alvará pode substituir inventário completo, considerando herdeiros, dependentes, bens e dívidas.", "A análise parte de legislação sucessória, documentos bancários, certidão de óbito, orientações judiciais e limites do levantamento.", "A triagem pede certidão de óbito, extrato bancário, documentos dos herdeiros, declaração de dependentes e informação sobre outros bens.", "O risco é simplificar caso que tem imóvel, disputa, testamento ou dívida relevante.", "O atendimento online confere valores, bens conhecidos e documentos para avaliar alvará, inventário ou outra medida."},
			{"testamento-duvida-validade", "testamento encontrado após a morte", "Os familiares encontraram testamento e precisam entender validade formal, herdeiros necessários, bens, cartório, certidões e possível inventário judicial.", "A pauta usa Código Civil, regras sucessórias, documentos notariais, CNJ e diferença entre testamento público, cerrado ou particular.", "São conferidos testamento, certidão de óbito, documentos dos herdeiros, matrícula de bens, extratos e informações de cartório.", "A cautela é partilhar bens sem confirmar testamento ou presumir invalidade sem análise formal.", "A triagem digital organiza documentos e perguntas para orientar abertura, registro ou discussão no inventário."},
			{"divida-espolio-cobranca", "dívida do espólio cobrada durante inventário", "Os herdeiros receberam cobrança contra o falecido e precisam separar dívida legítima, contrato, extratos, bens do espólio, imposto e responsabilidade patrimonial.", "A análise considera Código Civil, documentos de cobrança, inventário, bens do espólio e limites de responsabilidade dos herdeiros.", "A triagem pede contrato, cobrança, extratos, certidão de óbito, relação de bens, dívidas conhecidas e documentos dos herdeiros.", "O risco é pagar dívida sem verificar origem ou ignorar cobrança que afeta partilha e imposto.", "O atendimento online organiza credores, valores, bens e documentos para análise sucessória remota."},
		}
	default:
		return nil
	}
}

func convertBatchReport(report scalablebatches.Report) Report {
	issues := make([]Issue, 0, len(report.Issues))
	for _, issue := range report.Issues {
		issues = append(issues, Issue{Code: "generation_batch_load_failed", Message: issue.Code + ":" + issue.Message})
	}
	return Report{Issues: issues}
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

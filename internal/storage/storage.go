package storage

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const validationScannerMaxBytes = 1024 * 1024

type Contract struct {
	Format                             string  `json:"format"`
	DependencyPolicy                   string  `json:"dependency_policy"`
	ProductionCPUPolicy                string  `json:"production_cpu_policy"`
	SeparatesSourceAuditFromEditorial  bool    `json:"separates_source_audit_from_editorial"`
	SeparatesTermSeedFromPublicContent bool    `json:"separates_term_seed_from_public_content"`
	ContentStartPolicy                 string  `json:"content_start_policy"`
	Layers                             []Layer `json:"layers"`
	Root                               string  `json:"-"`
}

type Layer struct {
	Name                     string `json:"name"`
	Path                     string `json:"path"`
	RecordType               string `json:"record_type"`
	RecordMaxBytes           int    `json:"record_max_bytes"`
	PublicIndexable          bool   `json:"public_indexable"`
	AllowsRawOfficialText    bool   `json:"allows_raw_official_text"`
	AllowsEditorialContent   bool   `json:"allows_editorial_content"`
	RequiresSourceProvenance bool   `json:"requires_source_provenance"`
	RequiresQualityState     bool   `json:"requires_quality_state"`
	Description              string `json:"description"`
}

type Issue struct {
	Code    string
	Message string
}

type Report struct {
	Issues []Issue
}

func LoadContract(root string) (Contract, error) {
	projectRoot, err := findProjectRoot(root)
	if err != nil {
		return Contract{}, err
	}
	data, err := os.ReadFile(filepath.Join(projectRoot, "content", "storage_contract.json"))
	if err != nil {
		return Contract{}, err
	}
	var contract Contract
	if err := json.Unmarshal(data, &contract); err != nil {
		return Contract{}, err
	}
	contract.Root = projectRoot
	return contract, nil
}

func Validate(root string) Report {
	contract, err := LoadContract(root)
	if err != nil {
		return Report{Issues: []Issue{{Code: "storage_contract_load_failed", Message: err.Error()}}}
	}
	return contract.Validate()
}

func (c Contract) LayerByName(name string) (Layer, bool) {
	for _, layer := range c.Layers {
		if layer.Name == name {
			return layer, true
		}
	}
	return Layer{}, false
}

func AppendJSONL(root string, layerName string, record any) error {
	contract, err := LoadContract(root)
	if err != nil {
		return err
	}
	layer, ok := contract.LayerByName(layerName)
	if !ok {
		return fmt.Errorf("unknown_layer=%s", layerName)
	}
	data, err := json.Marshal(record)
	if err != nil {
		return err
	}
	if layer.RecordMaxBytes <= 0 {
		return fmt.Errorf("invalid_record_budget=%s", layerName)
	}
	if len(data)+1 > layer.RecordMaxBytes {
		return fmt.Errorf("record_too_large=%s bytes=%d max=%d", layerName, len(data)+1, layer.RecordMaxBytes)
	}
	path := filepath.Join(contract.Root, layer.Path)
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return err
	}
	defer file.Close()
	if _, err := file.Write(append(data, '\n')); err != nil {
		return err
	}
	return nil
}

func (c Contract) Validate() Report {
	issues := make([]Issue, 0)
	if c.Format != "jsonl_file_store_v1" {
		issues = append(issues, Issue{Code: "invalid_format", Message: "banco leve deve usar jsonl_file_store_v1"})
	}
	if c.DependencyPolicy != "stdlib_only_no_external_database_dependency" {
		issues = append(issues, Issue{Code: "invalid_dependency_policy", Message: "banco leve nao pode depender de banco externo, SDK ou dependencia externa"})
	}
	if !c.SeparatesSourceAuditFromEditorial {
		issues = append(issues, Issue{Code: "source_audit_not_separated", Message: "auditoria de fonte deve ficar separada de texto editorial"})
	}
	if !c.SeparatesTermSeedFromPublicContent {
		issues = append(issues, Issue{Code: "term_seed_not_separated", Message: "ingestao de termos deve ficar separada de conteudo publico"})
	}
	if !strings.Contains(c.ContentStartPolicy, "draft_only") {
		issues = append(issues, Issue{Code: "content_start_policy_not_draft_only", Message: "termos so podem iniciar rascunhos bloqueados para indexacao"})
	}

	required := []string{"term_seeds", "term_intent_candidates", "manual_keyword_research", "source_audits", "batch_source_url_audits", "source_snapshots", "editorial_drafts", "content_briefs", "authorial_content_drafts", "source_specificity_blockers", "source_specificity_resolutions", "prepublication_gates", "legal_editorial_reviews", "human_content_score", "scalable_content_batches", "batch_drafts", "batch_draft_expansion_archive", "batch_candidate_expansion_readiness", "batch_expansion_strategy", "batch_candidate_gates", "batch_candidate_reviews", "batch_prepublication_gates", "batch_source_specificity_resolutions", "batch_public_manifest_gates", "batch_final_authorial_drafts", "batch_paid_intent_gates", "batch_generation_metrics", "batch_source_matrix", "published_manifest"}
	paths := make(map[string]string)
	for _, name := range required {
		layer, ok := c.LayerByName(name)
		if !ok {
			issues = append(issues, Issue{Code: "missing_layer", Message: name})
			continue
		}
		issues = append(issues, c.validateLayer(layer, paths)...)
	}
	return Report{Issues: issues}
}

func (c Contract) validateLayer(layer Layer, paths map[string]string) []Issue {
	issues := make([]Issue, 0)
	if layer.Name == "" || layer.RecordType == "" || layer.Description == "" {
		issues = append(issues, Issue{Code: "incomplete_layer", Message: layer.Name})
	}
	if layer.Path == "" || !strings.HasPrefix(layer.Path, "data/") {
		issues = append(issues, Issue{Code: "invalid_layer_path", Message: layer.Name + ":" + layer.Path})
	} else {
		if previous := paths[layer.Path]; previous != "" {
			issues = append(issues, Issue{Code: "shared_layer_path", Message: previous + ":" + layer.Name + ":" + layer.Path})
		}
		paths[layer.Path] = layer.Name
		if _, err := os.Stat(filepath.Join(c.Root, layer.Path)); err != nil {
			issues = append(issues, Issue{Code: "missing_layer_file", Message: layer.Name + ":" + layer.Path})
		} else {
			issues = append(issues, validateJSONLFile(filepath.Join(c.Root, layer.Path), layer)...)
		}
	}
	if layer.RecordMaxBytes <= 0 || layer.RecordMaxBytes > 32768 {
		issues = append(issues, Issue{Code: "invalid_record_budget", Message: layer.Name})
	}
	if layer.PublicIndexable {
		issues = append(issues, Issue{Code: "storage_layer_directly_indexable", Message: layer.Name})
	}
	if layer.Name == "term_seeds" {
		if layer.AllowsRawOfficialText || layer.AllowsEditorialContent {
			issues = append(issues, Issue{Code: "term_seed_payload_too_broad", Message: "term_seeds nao armazena texto oficial bruto nem editorial"})
		}
		if !layer.RequiresSourceProvenance || !layer.RequiresQualityState {
			issues = append(issues, Issue{Code: "term_seed_missing_guards", Message: "term_seeds exige proveniencia e estado de qualidade"})
		}
	}
	if layer.Name == "term_intent_candidates" {
		if layer.AllowsRawOfficialText || layer.AllowsEditorialContent {
			issues = append(issues, Issue{Code: "term_intent_payload_too_broad", Message: "term_intent_candidates nao armazena texto oficial bruto nem editorial"})
		}
		if !layer.RequiresSourceProvenance || !layer.RequiresQualityState {
			issues = append(issues, Issue{Code: "term_intent_missing_guards", Message: "term_intent_candidates exige proveniencia e estado de qualidade"})
		}
	}
	if layer.Name == "manual_keyword_research" {
		if layer.AllowsRawOfficialText || layer.AllowsEditorialContent {
			issues = append(issues, Issue{Code: "manual_research_payload_too_broad", Message: "manual_keyword_research guarda pesquisa, nao texto oficial bruto nem conteudo publico"})
		}
	}
	if layer.Name == "editorial_drafts" {
		if !layer.AllowsEditorialContent || layer.AllowsRawOfficialText {
			issues = append(issues, Issue{Code: "editorial_draft_layer_invalid", Message: "drafts guardam texto editorial proprio, nao payload oficial bruto"})
		}
	}
	if layer.Name == "content_briefs" {
		if !layer.AllowsEditorialContent || layer.AllowsRawOfficialText {
			issues = append(issues, Issue{Code: "content_brief_layer_invalid", Message: "briefs guardam conteudo editorial inicial, nao payload oficial bruto"})
		}
	}
	if layer.Name == "authorial_content_drafts" {
		if !layer.AllowsEditorialContent || layer.AllowsRawOfficialText {
			issues = append(issues, Issue{Code: "authorial_draft_layer_invalid", Message: "rascunhos autorais guardam texto proprio em PT-BR, nao payload oficial bruto"})
		}
		if !layer.RequiresSourceProvenance || !layer.RequiresQualityState {
			issues = append(issues, Issue{Code: "authorial_draft_missing_guards", Message: "rascunhos autorais exigem proveniencia, estado de qualidade e bloqueio antes de publicar"})
		}
	}
	if layer.Name == "source_specificity_resolutions" {
		if layer.AllowsRawOfficialText || layer.AllowsEditorialContent {
			issues = append(issues, Issue{Code: "source_resolution_layer_invalid", Message: "resolucoes de fonte guardam metadados e proveniencia, nao texto oficial bruto nem texto editorial"})
		}
		if !layer.RequiresSourceProvenance || !layer.RequiresQualityState {
			issues = append(issues, Issue{Code: "source_resolution_missing_guards", Message: "resolucoes de fonte exigem proveniencia e estado de qualidade"})
		}
	}
	if layer.Name == "prepublication_gates" {
		if layer.AllowsRawOfficialText || layer.AllowsEditorialContent {
			issues = append(issues, Issue{Code: "prepublication_gate_layer_invalid", Message: "prepublication_gates guardam metadados de SEO/crawl, nao texto oficial bruto nem texto editorial"})
		}
		if !layer.RequiresSourceProvenance || !layer.RequiresQualityState {
			issues = append(issues, Issue{Code: "prepublication_gate_missing_guards", Message: "prepublication_gates exigem fonte resolvida e estado de qualidade"})
		}
	}
	if layer.Name == "legal_editorial_reviews" {
		if layer.AllowsRawOfficialText || layer.AllowsEditorialContent {
			issues = append(issues, Issue{Code: "legal_review_layer_invalid", Message: "legal_editorial_reviews guardam metadados de revisao e CTA rascunho, nao texto oficial bruto nem pagina editorial"})
		}
		if !layer.RequiresSourceProvenance || !layer.RequiresQualityState {
			issues = append(issues, Issue{Code: "legal_review_missing_guards", Message: "legal_editorial_reviews exigem fonte resolvida e estado de qualidade"})
		}
	}
	if layer.Name == "human_content_score" {
		if layer.AllowsRawOfficialText || layer.AllowsEditorialContent {
			issues = append(issues, Issue{Code: "human_score_layer_invalid", Message: "human_content_score guarda score e metadados, nao texto bruto nem pagina editorial"})
		}
		if !layer.RequiresSourceProvenance || !layer.RequiresQualityState {
			issues = append(issues, Issue{Code: "human_score_missing_guards", Message: "human_content_score exige fonte/proveniencia e estado de qualidade"})
		}
	}
	if layer.Name == "scalable_content_batches" {
		if layer.AllowsRawOfficialText || layer.AllowsEditorialContent {
			issues = append(issues, Issue{Code: "batch_layer_invalid", Message: "scalable_content_batches guarda manifesto de lote, nao texto bruto nem pagina editorial"})
		}
		if !layer.RequiresSourceProvenance || !layer.RequiresQualityState {
			issues = append(issues, Issue{Code: "batch_layer_missing_guards", Message: "scalable_content_batches exige fonte/proveniencia e estado de qualidade"})
		}
	}
	if layer.Name == "batch_drafts" {
		if !layer.AllowsEditorialContent || layer.AllowsRawOfficialText {
			issues = append(issues, Issue{Code: "batch_draft_layer_invalid", Message: "batch_drafts guarda rascunho editorial proprio, nao texto oficial bruto"})
		}
		if !layer.RequiresSourceProvenance || !layer.RequiresQualityState {
			issues = append(issues, Issue{Code: "batch_draft_layer_missing_guards", Message: "batch_drafts exige fonte/proveniencia e estado de qualidade"})
		}
	}
	if layer.Name == "batch_draft_expansion_archive" {
		if !layer.AllowsEditorialContent || layer.AllowsRawOfficialText {
			issues = append(issues, Issue{Code: "batch_draft_archive_layer_invalid", Message: "batch_draft_expansion_archive guarda rascunhos autorais bloqueados, nao texto oficial bruto"})
		}
		if !layer.RequiresSourceProvenance || !layer.RequiresQualityState {
			issues = append(issues, Issue{Code: "batch_draft_archive_layer_missing_guards", Message: "batch_draft_expansion_archive exige fonte/proveniencia e estado de qualidade"})
		}
	}
	if layer.Name == "batch_candidate_gates" {
		if layer.AllowsRawOfficialText || layer.AllowsEditorialContent {
			issues = append(issues, Issue{Code: "batch_candidate_gate_layer_invalid", Message: "batch_candidate_gates guarda gate de selecao, nao texto oficial bruto nem conteudo editorial"})
		}
		if !layer.RequiresSourceProvenance || !layer.RequiresQualityState {
			issues = append(issues, Issue{Code: "batch_candidate_gate_layer_missing_guards", Message: "batch_candidate_gates exige fonte/proveniencia e estado de qualidade"})
		}
	}
	if layer.Name == "batch_candidate_expansion_readiness" {
		if layer.AllowsRawOfficialText || layer.AllowsEditorialContent {
			issues = append(issues, Issue{Code: "batch_candidate_expansion_layer_invalid", Message: "batch_candidate_expansion_readiness guarda metadados de prontidao, nao texto oficial bruto nem conteudo editorial"})
		}
		if !layer.RequiresSourceProvenance || !layer.RequiresQualityState {
			issues = append(issues, Issue{Code: "batch_candidate_expansion_layer_missing_guards", Message: "batch_candidate_expansion_readiness exige fonte/proveniencia e estado de qualidade"})
		}
	}
	if layer.Name == "batch_expansion_strategy" {
		if layer.AllowsRawOfficialText || layer.AllowsEditorialContent {
			issues = append(issues, Issue{Code: "batch_expansion_strategy_layer_invalid", Message: "batch_expansion_strategy guarda plano de escala e validacoes, nao texto oficial bruto nem conteudo editorial"})
		}
		if !layer.RequiresSourceProvenance || !layer.RequiresQualityState {
			issues = append(issues, Issue{Code: "batch_expansion_strategy_layer_missing_guards", Message: "batch_expansion_strategy exige fonte/proveniencia e estado de qualidade"})
		}
	}
	if layer.Name == "batch_candidate_reviews" {
		if layer.AllowsRawOfficialText || layer.AllowsEditorialContent {
			issues = append(issues, Issue{Code: "batch_candidate_review_layer_invalid", Message: "batch_candidate_reviews guarda metadados de revisao e CTA rascunho, nao texto oficial bruto nem pagina editorial"})
		}
		if !layer.RequiresSourceProvenance || !layer.RequiresQualityState {
			issues = append(issues, Issue{Code: "batch_candidate_review_layer_missing_guards", Message: "batch_candidate_reviews exige fonte/proveniencia e estado de qualidade"})
		}
	}
	if layer.Name == "batch_prepublication_gates" {
		if layer.AllowsRawOfficialText || layer.AllowsEditorialContent {
			issues = append(issues, Issue{Code: "batch_prepublication_gate_layer_invalid", Message: "batch_prepublication_gates guarda metadados de canonical/noindex, nao texto oficial bruto nem pagina editorial"})
		}
		if !layer.RequiresSourceProvenance || !layer.RequiresQualityState {
			issues = append(issues, Issue{Code: "batch_prepublication_gate_layer_missing_guards", Message: "batch_prepublication_gates exige fonte/proveniencia e estado de qualidade"})
		}
	}
	if layer.Name == "batch_source_specificity_resolutions" {
		if layer.AllowsRawOfficialText || layer.AllowsEditorialContent {
			issues = append(issues, Issue{Code: "batch_source_specificity_layer_invalid", Message: "batch_source_specificity_resolutions guarda metadados de fonte e bloqueio, nao texto oficial bruto nem pagina editorial"})
		}
		if !layer.RequiresSourceProvenance || !layer.RequiresQualityState {
			issues = append(issues, Issue{Code: "batch_source_specificity_layer_missing_guards", Message: "batch_source_specificity_resolutions exige fonte/proveniencia e estado de qualidade"})
		}
	}
	if layer.Name == "batch_public_manifest_gates" {
		if layer.AllowsRawOfficialText || layer.AllowsEditorialContent {
			issues = append(issues, Issue{Code: "batch_public_manifest_layer_invalid", Message: "batch_public_manifest_gates guarda metadados de manifesto bloqueado, nao texto oficial bruto nem pagina editorial"})
		}
		if !layer.RequiresSourceProvenance || !layer.RequiresQualityState {
			issues = append(issues, Issue{Code: "batch_public_manifest_layer_missing_guards", Message: "batch_public_manifest_gates exige fonte/proveniencia e estado de qualidade"})
		}
	}
	if layer.Name == "batch_final_authorial_drafts" {
		if !layer.AllowsEditorialContent || layer.AllowsRawOfficialText {
			issues = append(issues, Issue{Code: "batch_final_draft_layer_invalid", Message: "batch_final_authorial_drafts guarda texto autoral proprio, nao texto oficial bruto"})
		}
		if !layer.RequiresSourceProvenance || !layer.RequiresQualityState {
			issues = append(issues, Issue{Code: "batch_final_draft_layer_missing_guards", Message: "batch_final_authorial_drafts exige fonte/proveniencia e estado de qualidade"})
		}
	}
	if layer.Name == "batch_paid_intent_gates" {
		if layer.AllowsRawOfficialText || layer.AllowsEditorialContent {
			issues = append(issues, Issue{Code: "batch_paid_intent_gate_layer_invalid", Message: "batch_paid_intent_gates guarda metadados de intencao comercial, nao texto oficial bruto nem conteudo editorial"})
		}
		if !layer.RequiresSourceProvenance || !layer.RequiresQualityState {
			issues = append(issues, Issue{Code: "batch_paid_intent_gate_layer_missing_guards", Message: "batch_paid_intent_gates exige fonte/proveniencia e estado de qualidade"})
		}
	}
	if layer.Name == "batch_generation_metrics" {
		if layer.AllowsRawOfficialText || layer.AllowsEditorialContent {
			issues = append(issues, Issue{Code: "batch_generation_metric_layer_invalid", Message: "batch_generation_metrics guarda metricas, nao texto oficial bruto nem conteudo editorial"})
		}
		if !layer.RequiresSourceProvenance || !layer.RequiresQualityState {
			issues = append(issues, Issue{Code: "batch_generation_metric_layer_missing_guards", Message: "batch_generation_metrics exige fonte/proveniencia e estado de qualidade"})
		}
	}
	if layer.Name == "batch_source_matrix" {
		if layer.AllowsRawOfficialText || layer.AllowsEditorialContent {
			issues = append(issues, Issue{Code: "batch_source_matrix_layer_invalid", Message: "batch_source_matrix guarda metadados e proveniencia, nao texto oficial bruto nem conteudo editorial"})
		}
		if !layer.RequiresSourceProvenance || !layer.RequiresQualityState {
			issues = append(issues, Issue{Code: "batch_source_matrix_layer_missing_guards", Message: "batch_source_matrix exige fonte/proveniencia e estado de qualidade"})
		}
	}
	if layer.Name == "batch_source_url_audits" {
		if layer.AllowsRawOfficialText || layer.AllowsEditorialContent {
			issues = append(issues, Issue{Code: "batch_source_url_audit_layer_invalid", Message: "batch_source_url_audits guarda auditoria URL-level, nao texto oficial bruto nem conteudo editorial"})
		}
		if !layer.RequiresSourceProvenance || !layer.RequiresQualityState {
			issues = append(issues, Issue{Code: "batch_source_url_audit_layer_missing_guards", Message: "batch_source_url_audits exige proveniencia e estado de qualidade"})
		}
	}
	return issues
}

func validateJSONLFile(path string, layer Layer) []Issue {
	file, err := os.Open(path)
	if err != nil {
		return []Issue{{Code: "open_layer_file_failed", Message: layer.Name + ":" + err.Error()}}
	}
	defer file.Close()

	issues := make([]Issue, 0)
	scanner := bufio.NewScanner(file)
	scannerMaxBytes := layer.RecordMaxBytes + 1
	if scannerMaxBytes < validationScannerMaxBytes {
		scannerMaxBytes = validationScannerMaxBytes
	}
	scanner.Buffer(make([]byte, 0, 4096), scannerMaxBytes)
	lineNumber := 0
	for scanner.Scan() {
		lineNumber++
		line := scanner.Bytes()
		if len(strings.TrimSpace(string(line))) == 0 {
			continue
		}
		if len(line)+1 > layer.RecordMaxBytes {
			issues = append(issues, Issue{Code: "jsonl_record_too_large", Message: fmt.Sprintf("%s:%d", layer.Name, lineNumber)})
			continue
		}
		var payload map[string]any
		if err := json.Unmarshal(line, &payload); err != nil {
			issues = append(issues, Issue{Code: "invalid_jsonl_record", Message: fmt.Sprintf("%s:%d:%s", layer.Name, lineNumber, err.Error())})
		}
	}
	if err := scanner.Err(); err != nil {
		issues = append(issues, Issue{Code: "scan_layer_file_failed", Message: layer.Name + ":" + err.Error()})
	}
	return issues
}

func (r Report) Passed() bool {
	return len(r.Issues) == 0
}

func (r Report) Messages() []string {
	messages := make([]string, 0, len(r.Issues))
	for _, issue := range r.Issues {
		messages = append(messages, fmt.Sprintf("%s: %s", issue.Code, issue.Message))
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

package batchcandidatepipeline

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode"

	"portaljuridico/internal/batchcandidatereviews"
	"portaljuridico/internal/batchdrafts"
	"portaljuridico/internal/batchfinaldrafts"
	"portaljuridico/internal/batchprepublication"
	"portaljuridico/internal/batchpublicmanifest"
	"portaljuridico/internal/batchsourcematrix"
	"portaljuridico/internal/batchsourcespecificity"
	"portaljuridico/internal/content"
	"portaljuridico/internal/humanscore"
	"portaljuridico/internal/paidintent"
	"portaljuridico/internal/quality"
	"portaljuridico/internal/seo"
)

type Result struct {
	Reviews            int
	Prepublication     int
	SourceSpecificity  int
	PublicManifest     int
	FinalDrafts        int
	SourceLocked       int
	SourceBlocked      int
	SelectedCandidates int
}

type Issue struct {
	Code    string
	Message string
}

type Report struct {
	Issues []Issue
}

type selectedCandidate struct {
	GateID         string
	BatchID        string
	UniqueIntentID string
	CandidatePath  string
	SourceMatrixID string
	Draft          batchdrafts.Record
}

func Refresh(root string) (Result, Report) {
	projectRoot, err := findProjectRoot(root)
	if err != nil {
		return Result{}, Report{Issues: []Issue{{Code: "project_root_not_found", Message: err.Error()}}}
	}

	candidateIndex, candidateReport := batchcandidatereviews.BuildCandidateIndex(projectRoot)
	matrixEntries, matrixReport := batchsourcematrix.LoadRecords(projectRoot)
	specificSourceURLsByMatrix, specificSourceReport := batchsourcespecificity.BuildSpecificSourceURLsByMatrix(projectRoot)
	finalEntries, finalReport := batchfinaldrafts.LoadRecords(projectRoot)
	prepublicationEntries, prepublicationReport := batchprepublication.LoadRecords(projectRoot)
	paidEntries, paidReport := paidintent.LoadRecords(projectRoot)
	repo, repoErr := content.LoadRepository(projectRoot)

	issues := convertReviewIssues(candidateReport)
	issues = append(issues, convertMatrixIssues(matrixReport)...)
	issues = append(issues, convertSourceSpecificityIssues(specificSourceReport)...)
	issues = append(issues, convertFinalIssues(finalReport)...)
	issues = append(issues, convertPrepublicationIssues(prepublicationReport)...)
	issues = append(issues, convertPaidIssues(paidReport)...)
	if repoErr != nil {
		issues = append(issues, Issue{Code: "batch_candidate_pipeline_site_config_unavailable", Message: repoErr.Error()})
	}
	if len(issues) > 0 {
		return Result{}, Report{Issues: issues}
	}

	baseURL := "https://wikijuridica.com.br"
	if repo.BaseURL != "" {
		baseURL = repo.BaseURL
	}
	baseURL = strings.TrimRight(baseURL, "/")

	sourceURLsByMatrix := make(map[string][]string)
	for _, entry := range matrixEntries {
		sourceURLsByMatrix[entry.Record.MatrixID] = append([]string{}, entry.Record.SourceURLs...)
	}

	oldSEOByIntent := make(map[string]batchprepublication.Record)
	for _, entry := range prepublicationEntries {
		oldSEOByIntent[entry.Record.UniqueIntentID] = entry.Record
	}

	finalByIntent := make(map[string]batchfinaldrafts.Record)
	for _, entry := range finalEntries {
		finalByIntent[entry.Record.UniqueIntentID] = entry.Record
	}

	paidByIntent := make(map[string]paidintent.Record)
	for _, entry := range paidEntries {
		paidByIntent[entry.Record.UniqueIntentID] = entry.Record
	}

	selected := selectedCandidates(candidateIndex)
	sourceLockedByIntent := make(map[string]bool)
	for _, candidate := range selected {
		gate, ok := paidByIntent[candidate.UniqueIntentID]
		if !ok {
			continue
		}
		if paidintent.AllowsExpansion(gate) && len(specificSourceURLsByMatrix[candidate.SourceMatrixID]) > 0 {
			sourceLockedByIntent[candidate.UniqueIntentID] = true
		}
	}

	reviews := make([]batchcandidatereviews.Record, 0, len(selected))
	prepublications := make([]batchprepublication.Record, 0, len(selected))
	sources := make([]batchsourcespecificity.Record, 0, len(selected))
	manifests := make([]batchpublicmanifest.Record, 0, len(selected))
	manifestByIntent := make(map[string]batchpublicmanifest.Record)
	finalDrafts := make([]batchfinaldrafts.Record, 0, len(sourceLockedByIntent))

	result := Result{SelectedCandidates: len(selected)}
	checkedAt := "2026-06-09"
	for _, candidate := range selected {
		if candidate.Draft.CheckedAt != "" {
			checkedAt = candidate.Draft.CheckedAt
		}
		review := buildReview(candidate, candidateIndex.BaseURLMode, candidateIndex.OfficialURLLocked, checkedAt)
		prepublication := buildPrepublication(candidate, review, baseURL, oldSEOByIntent[candidate.UniqueIntentID], checkedAt)
		sourceURLs := sourceURLsByMatrix[candidate.SourceMatrixID]
		if sourceLockedByIntent[candidate.UniqueIntentID] {
			sourceURLs = specificSourceURLsByMatrix[candidate.SourceMatrixID]
		}
		if len(sourceURLs) == 0 {
			issues = append(issues, Issue{Code: "batch_candidate_pipeline_missing_source_urls", Message: candidate.SourceMatrixID})
			continue
		}
		source := buildSourceSpecificity(candidate, prepublication, sourceURLs, sourceLockedByIntent[candidate.UniqueIntentID], checkedAt)
		manifest := buildManifest(prepublication, source, checkedAt)

		reviews = append(reviews, review)
		prepublications = append(prepublications, prepublication)
		sources = append(sources, source)
		manifests = append(manifests, manifest)
		manifestByIntent[candidate.UniqueIntentID] = manifest
		if source.SourceSpecificityStatus == batchsourcespecificity.LockedStatus {
			result.SourceLocked++
		}
		if source.SourceSpecificityStatus == batchsourcespecificity.BlockedStatus {
			result.SourceBlocked++
		}
	}
	if len(issues) > 0 {
		return result, Report{Issues: issues}
	}

	for _, candidate := range selected {
		if !sourceLockedByIntent[candidate.UniqueIntentID] {
			continue
		}
		manifest := manifestByIntent[candidate.UniqueIntentID]
		if old, ok := finalByIntent[candidate.UniqueIntentID]; ok && reusableFinalDraft(old, manifest) {
			finalDrafts = append(finalDrafts, old)
			continue
		}
		finalDrafts = append(finalDrafts, buildFinalDraft(candidate, manifest, checkedAt))
	}
	sort.Slice(finalDrafts, func(left int, right int) bool {
		if finalDrafts[left].BatchID != finalDrafts[right].BatchID {
			return finalDrafts[left].BatchID < finalDrafts[right].BatchID
		}
		return finalDrafts[left].UniqueIntentID < finalDrafts[right].UniqueIntentID
	})

	if err := writeRecords(projectRoot, "data/editorial/batch_candidate_reviews.jsonl", reviews); err != nil {
		issues = append(issues, Issue{Code: "batch_candidate_pipeline_write_reviews_failed", Message: err.Error()})
	}
	if err := writeRecords(projectRoot, "data/editorial/batch_prepublication_gates.jsonl", prepublications); err != nil {
		issues = append(issues, Issue{Code: "batch_candidate_pipeline_write_prepublication_failed", Message: err.Error()})
	}
	if err := writeRecords(projectRoot, "data/editorial/batch_source_specificity_resolutions.jsonl", sources); err != nil {
		issues = append(issues, Issue{Code: "batch_candidate_pipeline_write_source_specificity_failed", Message: err.Error()})
	}
	if err := writeRecords(projectRoot, "data/editorial/batch_public_manifest_gates.jsonl", manifests); err != nil {
		issues = append(issues, Issue{Code: "batch_candidate_pipeline_write_manifest_failed", Message: err.Error()})
	}
	if err := writeRecords(projectRoot, "data/editorial/batch_final_authorial_drafts.jsonl", finalDrafts); err != nil {
		issues = append(issues, Issue{Code: "batch_candidate_pipeline_write_final_drafts_failed", Message: err.Error()})
	}
	if len(issues) > 0 {
		return result, Report{Issues: issues}
	}

	result.Reviews = len(reviews)
	result.Prepublication = len(prepublications)
	result.SourceSpecificity = len(sources)
	result.PublicManifest = len(manifests)
	result.FinalDrafts = len(finalDrafts)
	return result, Report{}
}

func selectedCandidates(index batchcandidatereviews.CandidateIndex) []selectedCandidate {
	selected := make([]selectedCandidate, 0, len(index.SelectedByIntent))
	for _, candidate := range index.SelectedByIntent {
		selected = append(selected, selectedCandidate{
			GateID:         candidate.GateID,
			BatchID:        candidate.BatchID,
			UniqueIntentID: candidate.UniqueIntentID,
			CandidatePath:  candidate.CandidatePath,
			SourceMatrixID: candidate.SourceMatrixID,
			Draft:          candidate.Draft,
		})
	}
	sort.Slice(selected, func(left int, right int) bool {
		if selected[left].BatchID != selected[right].BatchID {
			return selected[left].BatchID < selected[right].BatchID
		}
		return selected[left].UniqueIntentID < selected[right].UniqueIntentID
	})
	return selected
}

func buildReview(candidate selectedCandidate, baseURLMode string, officialURLLocked bool, checkedAt string) batchcandidatereviews.Record {
	term := strings.TrimSpace(candidate.Draft.Term)
	return batchcandidatereviews.Record{
		ReviewID:          "review-" + candidate.UniqueIntentID,
		GateID:            candidate.GateID,
		BatchID:           candidate.BatchID,
		UniqueIntentID:    candidate.UniqueIntentID,
		CandidatePath:     candidate.CandidatePath,
		BaseURLMode:       baseURLMode,
		OfficialURLLocked: officialURLLocked,
		ReviewStatus:      "batch_candidate_review_blocked",
		Language:          "pt-BR",
		LegalArea:         candidate.Draft.LegalArea,
		Term:              term,
		SourceMatrixID:    candidate.SourceMatrixID,
		ReviewerRole:      "juridico_editorial_lab",
		LegalReviewNotes: []string{
			"Conferir " + term + " a partir de documentos, datas e fonte oficial antes de qualquer orientação pública.",
			"A revisão deve separar problema do leitor, risco jurídico, prova digital e limites do atendimento remoto.",
			"O CTA fica contextual ao WhatsApp para triagem paga, sem promessa de resultado ou publicação automática.",
		},
		RequiredFixes: []string{
			"Resolver fonte específica final, revisão SEO e manifesto público antes de qualquer renderização.",
		},
		CTADraft:               "Para triagem jurídica online pelo WhatsApp, envie documentos principais, datas, protocolos, contratos, comprovantes e dúvida objetiva sobre " + term + " antes da análise.",
		CTAContextMessage:      "Origem: " + candidate.CandidatePath + " Gate: " + candidate.GateID + " Intent: " + candidate.UniqueIntentID + " Tema: " + term + ". WhatsApp recebe documentos e contexto para triagem jurídica digital paga, sem promessa de resultado.",
		CTAStatus:              "draft_contextual_not_public",
		PublicationBlockReason: "P0 bloqueado: revisão em massa exige fonte específica, SEO final, manifesto público finito e aprovação explícita antes de publicar.",
		RenderAllowed:          false,
		SitemapAllowed:         false,
		PublicationAllowed:     false,
		PublicPath:             "",
		CheckedAt:              checkedAt,
	}
}

func buildPrepublication(candidate selectedCandidate, review batchcandidatereviews.Record, baseURL string, old batchprepublication.Record, checkedAt string) batchprepublication.Record {
	title := candidateTitle(candidate.Draft.Term, old.CandidateTitle)
	meta := candidateMeta(candidate.Draft.Term, old.CandidateMetaDescription)
	return batchprepublication.Record{
		PrepublicationID:         "prepub-" + candidate.UniqueIntentID,
		ReviewID:                 review.ReviewID,
		GateID:                   candidate.GateID,
		BatchID:                  candidate.BatchID,
		UniqueIntentID:           candidate.UniqueIntentID,
		SourceMatrixID:           candidate.SourceMatrixID,
		Term:                     candidate.Draft.Term,
		CandidatePath:            candidate.CandidatePath,
		CandidateCanonicalURL:    strings.TrimRight(baseURL, "/") + candidate.CandidatePath,
		CandidateRobots:          "noindex,follow",
		CandidateTitle:           title,
		CandidateMetaDescription: meta,
		SourceSpecificityStatus:  "matrix_audited_final_source_pending",
		RemainingGates: []string{
			"fonte específica final do candidato",
			"revisão SEO final",
			"manifesto público finito",
			"aprovação para render e sitemap",
		},
		RenderAllowed:      false,
		SitemapAllowed:     false,
		PublicationAllowed: false,
		PublicPath:         "",
		CheckedAt:          checkedAt,
	}
}

func buildSourceSpecificity(candidate selectedCandidate, prepublication batchprepublication.Record, sourceURLs []string, locked bool, checkedAt string) batchsourcespecificity.Record {
	status := batchsourcespecificity.BlockedStatus
	blockingReason := "fonte oficial da matriz ainda não resolve o recorte específico de " + candidate.Draft.Term + " com precisão suficiente para render público em escala."
	neededSourceDetail := "auditar URL oficial específica, termo de uso e aplicabilidade ao subtema antes de liberar manifesto, HTML ou sitemap."
	if locked {
		status = batchsourcespecificity.LockedStatus
		blockingReason = ""
		neededSourceDetail = ""
	}
	return batchsourcespecificity.Record{
		ResolutionID:            "source-specificity-" + candidate.UniqueIntentID,
		PrepublicationID:        prepublication.PrepublicationID,
		ReviewID:                prepublication.ReviewID,
		BatchID:                 candidate.BatchID,
		UniqueIntentID:          candidate.UniqueIntentID,
		SourceMatrixID:          candidate.SourceMatrixID,
		Term:                    candidate.Draft.Term,
		CandidatePath:           candidate.CandidatePath,
		CandidateCanonicalURL:   prepublication.CandidateCanonicalURL,
		CandidateRobots:         prepublication.CandidateRobots,
		SourceSpecificityStatus: status,
		SelectedSourceURLs:      append([]string{}, sourceURLs...),
		BlockingReason:          blockingReason,
		NeededSourceDetail:      neededSourceDetail,
		UsePolicy:               batchsourcespecificity.UsePolicy,
		ScrapingAllowed:         false,
		IngestionAllowed:        false,
		RenderAllowed:           false,
		SitemapAllowed:          false,
		PublicationAllowed:      false,
		PublicPath:              "",
		CheckedAt:               checkedAt,
	}
}

func buildManifest(prepublication batchprepublication.Record, source batchsourcespecificity.Record, checkedAt string) batchpublicmanifest.Record {
	status := batchpublicmanifest.SourceBlockedStatus
	seoReviewRequired := false
	contentDraftRequired := false
	if source.SourceSpecificityStatus == batchsourcespecificity.LockedStatus {
		status = batchpublicmanifest.SEOReviewPendingStatus
		seoReviewRequired = true
		contentDraftRequired = true
	}
	return batchpublicmanifest.Record{
		ManifestGateID:           "public-manifest-" + source.UniqueIntentID,
		SourceResolutionID:       source.ResolutionID,
		PrepublicationID:         source.PrepublicationID,
		ReviewID:                 source.ReviewID,
		BatchID:                  source.BatchID,
		UniqueIntentID:           source.UniqueIntentID,
		SourceMatrixID:           source.SourceMatrixID,
		Term:                     source.Term,
		CandidatePath:            source.CandidatePath,
		CandidateCanonicalURL:    source.CandidateCanonicalURL,
		CandidateRobots:          source.CandidateRobots,
		CandidateTitle:           prepublication.CandidateTitle,
		CandidateMetaDescription: prepublication.CandidateMetaDescription,
		SourceSpecificityStatus:  source.SourceSpecificityStatus,
		ManifestGateStatus:       status,
		SelectedSourceURLs:       append([]string{}, source.SelectedSourceURLs...),
		BlockingReason:           source.BlockingReason,
		NeededSourceDetail:       source.NeededSourceDetail,
		IndexPolicy:              "noindex",
		UsePolicy:                batchpublicmanifest.UsePolicy,
		SEOReviewRequired:        seoReviewRequired,
		ContentDraftRequired:     contentDraftRequired,
		ManifestAllowed:          false,
		RenderAllowed:            false,
		SitemapAllowed:           false,
		PublicationAllowed:       false,
		PublicPath:               "",
		CheckedAt:                checkedAt,
	}
}

func reusableFinalDraft(record batchfinaldrafts.Record, manifest batchpublicmanifest.Record) bool {
	if record.BatchID == "batch-previdenciario-digital" && hasLegacyPrevidenciarioFinalDraft(record) {
		return false
	}
	if record.ManifestGateID != manifest.ManifestGateID {
		return false
	}
	if record.BatchID != manifest.BatchID || record.SourceMatrixID != manifest.SourceMatrixID {
		return false
	}
	if record.Term != manifest.Term || record.CandidatePath != manifest.CandidatePath {
		return false
	}
	if record.CandidateCanonicalURL != manifest.CandidateCanonicalURL {
		return false
	}
	if record.CandidateTitle != manifest.CandidateTitle || record.CandidateMetaDescription != manifest.CandidateMetaDescription {
		return false
	}
	if !sameStringSet(record.SelectedSourceURLs, manifest.SelectedSourceURLs) {
		return false
	}
	score := humanscore.ScoreText(record.FullText())
	if score.HumanScore < 85 || score.AILikeScore > 20 || len(score.BlockingIssues) > 0 {
		return false
	}
	return quality.AnalyzeText(record.FullText()).Passed()
}

func hasLegacyPrevidenciarioFinalDraft(record batchfinaldrafts.Record) bool {
	return strings.HasPrefix(record.Opening, "Caso em direito previdenciário exige recorte concreto para triagem online.") ||
		strings.HasPrefix(record.DocumentGuidance, "Documentos de trabalho: decisão do INSS, laudos, exames, CNIS, protocolo do Meu INSS") ||
		strings.HasPrefix(record.DigitalTriage, "Triagem digital organiza cronologia, autoridade consultada, parte contrária, prejuízo")
}

func buildFinalDraft(candidate selectedCandidate, manifest batchpublicmanifest.Record, checkedAt string) batchfinaldrafts.Record {
	term := strings.TrimSpace(candidate.Draft.Term)
	if term == "" {
		term = manifest.Term
	}
	record := batchfinaldrafts.Record{
		DraftID:                  "final-draft-" + candidate.UniqueIntentID,
		ManifestGateID:           manifest.ManifestGateID,
		UniqueIntentID:           candidate.UniqueIntentID,
		BatchID:                  candidate.BatchID,
		SourceMatrixID:           candidate.SourceMatrixID,
		Term:                     term,
		CandidatePath:            manifest.CandidatePath,
		CandidateCanonicalURL:    manifest.CandidateCanonicalURL,
		CandidateTitle:           manifest.CandidateTitle,
		CandidateMetaDescription: manifest.CandidateMetaDescription,
		SelectedSourceURLs:       append([]string{}, manifest.SelectedSourceURLs...),
		DraftStatus:              batchfinaldrafts.DraftStatus,
		Language:                 "pt-BR",
		Opening:                  finalOpening(candidate),
		SourceUse:                finalSourceUse(candidate, manifest.SelectedSourceURLs),
		DocumentGuidance:         finalDocumentGuidance(candidate),
		DigitalTriage:            finalDigitalTriage(candidate),
		CTAContextMessage:        finalCTA(candidate, manifest),
		InformationalNotice:      "Conteúdo informativo em preparação; não substitui consulta jurídica individual, análise de documentos, estratégia processual ou orientação profissional sobre caso concreto.",
		IndexPolicy:              "noindex",
		RenderAllowed:            false,
		SitemapAllowed:           false,
		PublicationAllowed:       false,
		PublicPath:               "",
		CheckedAt:                checkedAt,
	}
	score := humanscore.ScoreText(record.FullText())
	record.HumanScore = score.HumanScore
	record.AILikeScore = score.AILikeScore
	if record.HumanScore < 85 || record.AILikeScore > 20 || len(score.BlockingIssues) > 0 {
		record.Opening = record.Opening + " O recorte não deve virar orientação genérica: ele precisa preservar valor discutido, prazo, documento mínimo, fonte oficial e ato digital possível para avaliação editorial."
		record.DocumentGuidance = record.DocumentGuidance + " Conferir também contrato, comprovantes de pagamento, mensagens com a outra parte, protocolo administrativo, identificação das partes e uma linha do tempo curta com datas verificáveis."
		score = humanscore.ScoreText(record.FullText())
		record.HumanScore = score.HumanScore
		record.AILikeScore = score.AILikeScore
	}
	return record
}

func finalOpening(candidate selectedCandidate) string {
	if candidate.Draft.LegalArea == "previdenciario" {
		return finalPrevidenciarioOpening(candidate)
	}
	problem := "Leitura inicial cruza fatos, documentos, autoridade consultada e resposta da outra parte, sem transformar o tema em promessa automática."
	return "Caso em " + areaLabel(candidate) + " exige recorte concreto para triagem online. " + problem + " Faceta analisada: " + focusLabel(candidate) + ", com validação jurídico-editorial ainda bloqueada para evitar página pública sem fonte suficiente."
}

func finalSourceUse(candidate selectedCandidate, urls []string) string {
	sourceCount := fmt.Sprintf(" O lote carrega %d URL oficial específica auditada para este recorte.", len(urls))
	return "Fonte oficial entra como prova de autoridade, não como texto a copiar." + sourceCount + " Uso permitido: referência, conferência de contexto e vínculo de proveniência; scraping, ingestão, renderização e sitemap continuam bloqueados."
}

func finalDocumentGuidance(candidate selectedCandidate) string {
	if candidate.Draft.LegalArea == "previdenciario" {
		return finalPrevidenciarioDocumentGuidance(candidate)
	}
	return "Documentos de trabalho: " + documentChecklist(candidate) + ". Revisão deve separar prova essencial, complemento útil, lacuna que impede conclusão e dado que muda urgência, valor discutido ou prazo de resposta."
}

func finalDigitalTriage(candidate selectedCandidate) string {
	if candidate.Draft.LegalArea == "previdenciario" {
		return finalPrevidenciarioDigitalTriage(candidate)
	}
	return "Triagem digital organiza cronologia, autoridade consultada, parte contrária, prejuízo, tentativa de solução e arquivo mínimo. Risco principal: " + riskLabel(candidate) + ". Serviço jurídico pago só avança com orçamento de honorários, escopo remoto e documentos suficientes para análise particular."
}

func finalPrevidenciarioOpening(candidate selectedCandidate) string {
	term := strings.TrimSpace(candidate.Draft.Term)
	if term == "" {
		term = "tema previdenciário"
	}
	baseTerm := cleanFinalTerm(term)
	problem := polishFinalSnippet(firstSentence(candidate.Draft.ReaderProblem))
	if problem == "" {
		problem = "O caso precisa ser separado por benefício, fase administrativa, documento principal e resposta do INSS."
	}
	return "Análise previdenciária de " + baseTerm + " parte de " + focusLabel(candidate) + " sem antecipar conclusão sobre benefício. " + problem + " Material informativo permanece bloqueado, sem promessa de concessão, revisão ou prazo."
}

func finalPrevidenciarioDocumentGuidance(candidate selectedCandidate) string {
	documentFocus := polishFinalSnippet(firstSentence(candidate.Draft.DocumentContext))
	if documentFocus == "" {
		documentFocus = "Documento previdenciário útil precisa ligar decisão, CNIS, laudo, protocolo e fase do pedido."
	}
	return "Na prova documental, " + lowerFirst(trimSentenceEnd(documentFocus)) + ". Checklist contextual: " + documentChecklist(candidate) + ". Revisão deve separar prova essencial, complemento útil, lacuna que impede conclusão e dado que muda prazo, renda, fase administrativa ou necessidade de consulta jurídica."
}

func finalPrevidenciarioDigitalTriage(candidate selectedCandidate) string {
	action := polishFinalSnippet(firstSentence(candidate.Draft.DigitalAction))
	if action == "" {
		action = "A triagem online organiza requerimento, recurso, exigência, perícia, CNIS e documento médico por data."
	}
	risk := polishFinalSnippet(firstSentence(candidate.Draft.RiskContext))
	if risk == "" {
		risk = "O risco principal é confundir dúvida administrativa, documento incompleto e tese jurídica madura."
	}
	return "Pelo canal digital, " + lowerFirst(trimSentenceEnd(action)) + ". Risco observado: " + lowerFirst(trimSentenceEnd(risk)) + ". Mensagem de WhatsApp leva origem, intenção, documentos e fase do pedido para consulta remota responsável, sem substituir canal público nem prometer resultado."
}

func polishFinalSnippet(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	replacer := strings.NewReplacer(
		"Antes de qualquer CTA", "Antes do CTA",
		"antes de qualquer CTA", "antes do CTA",
		"Antes de qualquer avaliacao remota", "Antes da avaliação remota",
		"antes de qualquer avaliacao remota", "antes da avaliação remota",
		"Antes de qualquer avaliação remota", "Antes da avaliação remota",
		"antes de qualquer avaliação remota", "antes da avaliação remota",
		"Antes de qualquer roteiro juridico", "Antes do roteiro jurídico",
		"antes de qualquer roteiro juridico", "antes do roteiro jurídico",
		"Antes de qualquer roteiro jurídico", "Antes do roteiro jurídico",
		"antes de qualquer roteiro jurídico", "antes do roteiro jurídico",
		"antes de qualquer conclusão", "antes da conclusão",
		"antes de qualquer publicacao", "antes da publicação",
		"juridico", "jurídico",
		"generico", "genérico",
		"avaliacao", "avaliação",
		"responsavel", "responsável",
		"identificacao", "identificação",
		"comunicacao", "comunicação",
		"matricula", "matrícula",
		"dividas", "dívidas",
		"patrimonio", "patrimônio",
		"Decisao", "Decisão",
	)
	return replacer.Replace(value)
}

func cleanFinalTerm(term string) string {
	term = strings.TrimSpace(term)
	marker := " com foco em "
	if index := strings.LastIndex(term, marker); index > 0 {
		return strings.TrimSpace(term[:index])
	}
	return term
}

func trimSentenceEnd(value string) string {
	return strings.TrimRight(strings.TrimSpace(value), ".!?")
}

func lowerFirst(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return value
	}
	runes := []rune(value)
	runes[0] = unicode.ToLower(runes[0])
	return string(runes)
}

func finalCTA(candidate selectedCandidate, manifest batchpublicmanifest.Record) string {
	documents := documentChecklist(candidate)
	return "Origem: " + manifest.CandidatePath + " | Intent: " + compactIntentID(candidate.UniqueIntentID) + " | Documentos: " + documents + " | Contratação: triagem jurídica particular online com orçamento de honorários e contexto do caso antes da análise."
}

func areaLabel(candidate selectedCandidate) string {
	switch candidate.Draft.LegalArea {
	case "consumidor-financeiro":
		return "consumo financeiro"
	case "previdenciario":
		return "direito previdenciário"
	case "saude-suplementar":
		return "saúde suplementar"
	case "trabalhista":
		return "direito do trabalho"
	case "familia":
		return "família"
	case "sucessorio":
		return "sucessões"
	default:
		return "tema jurídico"
	}
}

func focusLabel(candidate selectedCandidate) string {
	id := candidate.UniqueIntentID
	switch {
	case strings.Contains(id, "documento-minimo"):
		return "documento mínimo verificável"
	case strings.Contains(id, "fonte-primaria"):
		return "fonte oficial aplicável ao subtema"
	case strings.Contains(id, "negociacao-previa"):
		return "tentativa de solução e resposta da parte contrária"
	case strings.Contains(id, "risco-economico"):
		return "valor envolvido e consequência financeira"
	case strings.Contains(id, "vulnerabilidade"):
		return "prioridade prática sem usar atributo sensível"
	case strings.Contains(id, "competencia-digital"):
		return "execução remota do atendimento"
	case strings.Contains(id, "linha-do-tempo"):
		return "cronologia dos fatos e prazos"
	case strings.Contains(id, "prazo-e-urgencia"):
		return "urgência documentada e prazo"
	case strings.Contains(id, "prova-digital"):
		return "prints, protocolos e arquivos eletrônicos"
	default:
		return "problema central documentado"
	}
}

func documentChecklist(candidate selectedCandidate) string {
	switch candidate.Draft.LegalArea {
	case "consumidor-financeiro":
		return "extrato, comprovante da operação, contrato ou fatura, protocolo do banco, prints do aplicativo, boletim quando houver e resposta administrativa"
	case "previdenciario":
		return "decisão do INSS, laudos, exames, CNIS, protocolo do Meu INSS, descrição da atividade profissional, atestados e histórico de requerimentos"
	case "saude-suplementar":
		return "prescrição, relatório clínico, exames, negativa formal, protocolo da operadora, contrato do plano, carteirinha, orçamento e comprovante de pagamento"
	case "trabalhista":
		return "contrato, holerites, cartões de ponto, CAT ou atestado, mensagens, advertências, TRCT, extrato do FGTS e descrição da rotina"
	case "familia":
		return "certidões, documentos dos filhos, comprovantes de renda, proposta de acordo, relação de bens, despesas e mensagens entre as partes"
	case "sucessorio":
		return "certidão de óbito, documentos dos herdeiros, matrícula de imóvel, extratos, testamento quando existir, dívidas do espólio e impostos"
	default:
		return "contrato, protocolos, comprovantes, mensagens, datas relevantes e resposta administrativa"
	}
}

func riskLabel(candidate selectedCandidate) string {
	switch candidate.Draft.LegalArea {
	case "consumidor-financeiro":
		return "perda de prova bancária, cobrança indevida, negativação ou demora em resposta sobre valor identificado"
	case "previdenciario":
		return "prazo administrativo, perda de renda, laudo incompleto, qualidade de segurado ou divergência entre doença e função"
	case "saude-suplementar":
		return "agravamento clínico, custo particular, negativa sem justificativa, prazo assistencial e falta de documento médico robusto"
	case "trabalhista":
		return "prescrição, prova frágil, cálculo incorreto, ausência de registro de jornada ou impacto econômico da rescisão"
	case "familia":
		return "acordo incompleto, conflito sobre bens, despesas dos filhos, guarda, pensão ou documento que impeça ato remoto"
	case "sucessorio":
		return "prazo fiscal, bloqueio de valores, disputa entre herdeiros, bem sem matrícula regular ou obrigação do espólio"
	default:
		return "prazo, valor envolvido, prova fraca ou resposta administrativa insuficiente"
	}
}

func firstSentence(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	for index, r := range value {
		if r == '.' || r == '!' || r == '?' {
			return strings.TrimSpace(value[:index+1])
		}
	}
	return trimRunesAtWord(value, 190)
}

func compactIntentID(value string) string {
	return strings.ReplaceAll(strings.TrimSpace(value), "-", "")
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

func candidateTitle(term string, previous string) string {
	previous = strings.TrimSpace(previous)
	if validTitle(previous) {
		return previous
	}
	title := titleFromTerm(term)
	if title == "" {
		title = "Tema jurídico para triagem online"
	}
	if len([]rune(title)) < seo.TitleMinCharacters {
		title = strings.TrimSpace(title + " jurídico online")
	}
	if len([]rune(title)) > seo.TitleMaxCharacters {
		title = trimTitleAtWord(title, seo.TitleMaxCharacters)
	}
	return title
}

func candidateMeta(term string, previous string) string {
	previous = strings.TrimSpace(previous)
	if validMeta(previous) {
		return previous
	}
	subject := metaSubject(term)
	meta := "Organize documentos, datas, protocolos e fonte oficial antes da triagem jurídica online sobre " + subject + "."
	if len([]rune(meta)) > seo.MetaDescriptionMaxCharacters {
		meta = "Organize documentos, datas e fonte oficial antes da triagem jurídica online: " + trimRunesAtWord(subject, 72) + "."
	}
	if len([]rune(meta)) > seo.MetaDescriptionMaxCharacters {
		meta = "Organize documentos, datas e fonte oficial antes da triagem jurídica online do caso."
	}
	if len([]rune(meta)) < seo.MetaDescriptionMinCharacters {
		meta = strings.TrimRight(meta, ".") + ", com contexto suficiente para análise remota."
	}
	return meta
}

func validTitle(value string) bool {
	length := len([]rune(value))
	return length >= seo.TitleMinCharacters && length <= seo.TitleMaxCharacters && !looksTruncatedFocus(value)
}

func validMeta(value string) bool {
	length := len([]rune(value))
	return length >= seo.MetaDescriptionMinCharacters && length <= seo.MetaDescriptionMaxCharacters && !looksTruncatedFocus(value)
}

func uppercaseFirst(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	runes := []rune(value)
	runes[0] = unicode.ToUpper(runes[0])
	return string(runes)
}

func titleFromTerm(term string) string {
	term = strings.TrimSpace(term)
	if term == "" {
		return ""
	}
	separator := " com foco em "
	if index := strings.LastIndex(strings.ToLower(term), separator); index > 0 {
		base := strings.TrimSpace(term[:index])
		focus := strings.TrimSpace(term[index+len(separator):])
		if base != "" && focus != "" {
			title := uppercaseFirst(base) + ": " + focus
			if len([]rune(title)) <= seo.TitleMaxCharacters {
				return title
			}
			availableBase := seo.TitleMaxCharacters - len([]rune(": "+focus))
			if availableBase >= seo.TitleMinCharacters {
				return trimRunesAtWord(uppercaseFirst(base), availableBase) + ": " + focus
			}
		}
	}
	return uppercaseFirst(term)
}

func metaSubject(term string) string {
	subject := titleFromTerm(term)
	if subject == "" {
		return "o caso"
	}
	return lowercaseFirst(subject)
}

func lowercaseFirst(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	runes := []rune(value)
	runes[0] = unicode.ToLower(runes[0])
	return string(runes)
}

func trimTitleAtWord(value string, limit int) string {
	trimmed := trimRunesAtWord(value, limit)
	if !looksTruncatedFocus(trimmed) {
		return trimmed
	}
	cleaned := strings.TrimSpace(strings.TrimSuffix(trimmed, "com foco em"))
	cleaned = strings.TrimSpace(strings.TrimSuffix(cleaned, "com foco"))
	cleaned = strings.TrimRight(cleaned, " :-")
	if len([]rune(cleaned)) >= seo.TitleMinCharacters {
		return cleaned
	}
	return trimmed
}

func looksTruncatedFocus(value string) bool {
	normalized := strings.ToLower(strings.TrimSpace(value))
	normalized = strings.TrimRight(normalized, ".:;- ")
	return strings.HasSuffix(normalized, "com foco em") || strings.HasSuffix(normalized, "com foco")
}

func trimRunesAtWord(value string, limit int) string {
	runes := []rune(strings.TrimSpace(value))
	if len(runes) <= limit {
		return string(runes)
	}
	cut := limit
	for cut > seo.TitleMinCharacters && !unicode.IsSpace(runes[cut-1]) {
		cut--
	}
	if cut <= seo.TitleMinCharacters {
		cut = limit
	}
	return strings.TrimSpace(string(runes[:cut]))
}

func writeRecords[T any](projectRoot string, relativePath string, records []T) error {
	path := filepath.Join(projectRoot, relativePath)
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	var builder strings.Builder
	for _, record := range records {
		encoded, err := json.Marshal(record)
		if err != nil {
			return err
		}
		builder.Write(encoded)
		builder.WriteByte('\n')
	}
	return os.WriteFile(path, []byte(builder.String()), 0644)
}

func convertReviewIssues(report batchcandidatereviews.Report) []Issue {
	issues := make([]Issue, 0, len(report.Issues))
	for _, issue := range report.Issues {
		issues = append(issues, Issue{Code: "batch_candidate_pipeline_candidate_" + issue.Code, Message: issue.Message})
	}
	return issues
}

func convertMatrixIssues(report batchsourcematrix.Report) []Issue {
	issues := make([]Issue, 0, len(report.Issues))
	for _, issue := range report.Issues {
		issues = append(issues, Issue{Code: "batch_candidate_pipeline_matrix_" + issue.Code, Message: issue.Message})
	}
	return issues
}

func convertSourceSpecificityIssues(report batchsourcespecificity.Report) []Issue {
	issues := make([]Issue, 0, len(report.Issues))
	for _, issue := range report.Issues {
		issues = append(issues, Issue{Code: "batch_candidate_pipeline_source_specificity_" + issue.Code, Message: issue.Message})
	}
	return issues
}

func convertFinalIssues(report batchfinaldrafts.Report) []Issue {
	issues := make([]Issue, 0, len(report.Issues))
	for _, issue := range report.Issues {
		issues = append(issues, Issue{Code: "batch_candidate_pipeline_final_" + issue.Code, Message: issue.Message})
	}
	return issues
}

func convertPrepublicationIssues(report batchprepublication.Report) []Issue {
	issues := make([]Issue, 0, len(report.Issues))
	for _, issue := range report.Issues {
		issues = append(issues, Issue{Code: "batch_candidate_pipeline_prepublication_" + issue.Code, Message: issue.Message})
	}
	return issues
}

func convertPaidIssues(report paidintent.Report) []Issue {
	issues := make([]Issue, 0, len(report.Issues))
	for _, issue := range report.Issues {
		issues = append(issues, Issue{Code: "batch_candidate_pipeline_paid_" + issue.Code, Message: issue.Message})
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
			return "", fmt.Errorf("go.mod not found from %s", start)
		}
		current = parent
	}
}

package contract_test

import (
	"strings"
	"testing"

	"portaljuridico/internal/batchfinaldrafts"
)

func TestBatchFinalAuthorialDraftsCoverEligibleManifestWithoutPublishing(t *testing.T) {
	report := batchfinaldrafts.Validate(".")
	if !report.Passed() {
		t.Fatalf("batch final authorial drafts failed contract: %v", report.Messages())
	}

	records, loadReport := batchfinaldrafts.LoadRecords(".")
	if !loadReport.Passed() {
		t.Fatalf("could not load batch final authorial drafts: %v", loadReport.Messages())
	}
	index, indexReport := batchfinaldrafts.BuildManifestIndex(".")
	if !indexReport.Passed() {
		t.Fatalf("could not build final draft manifest index: %v", indexReport.Messages())
	}
	if len(records) != len(index.EligibleByIntent) {
		t.Fatalf("final authorial drafts=%d eligible_manifest=%d, want one blocked draft for each source-locked manifest gate", len(records), len(index.EligibleByIntent))
	}
	if len(records) < 5 {
		t.Fatalf("final authorial drafts=%d, want at least 5 paid-passed preserved drafts after filtering free/admin-risk intents", len(records))
	}
	if len(records) < 168 {
		t.Fatalf("final authorial drafts=%d, want at least 168 blocked drafts generated from source-locked mass manifests", len(records))
	}

	for _, entry := range records {
		record := entry.Record
		if record.Language != "pt-BR" {
			t.Fatalf("line=%d language=%q", entry.Line, record.Language)
		}
		if record.HumanScore < 85 || record.AILikeScore > 20 {
			t.Fatalf("line=%d invalid score human=%d ai=%d", entry.Line, record.HumanScore, record.AILikeScore)
		}
		if record.RenderAllowed || record.SitemapAllowed || record.PublicationAllowed || record.PublicPath != "" {
			t.Fatalf("line=%d draft escaped blocked contract: render=%t sitemap=%t publication=%t path=%q", entry.Line, record.RenderAllowed, record.SitemapAllowed, record.PublicationAllowed, record.PublicPath)
		}
		if record.CTAContextMessage == "" || !record.HasContextualCTA() {
			t.Fatalf("line=%d CTA must carry origin, intent and documents", entry.Line)
		}
		if !record.HasInformationalNotice() {
			t.Fatalf("line=%d missing informational legal notice", entry.Line)
		}
	}
}

func TestBatchFinalAuthorialDraftsKeepPrevidenciarioSemanticVariation(t *testing.T) {
	records, loadReport := batchfinaldrafts.LoadRecords(".")
	if !loadReport.Passed() {
		t.Fatalf("could not load batch final authorial drafts: %v", loadReport.Messages())
	}

	previdenciarioCount := 0
	openings := make(map[string]int)
	documentGuidance := make(map[string]int)
	digitalTriage := make(map[string]int)
	for _, entry := range records {
		record := entry.Record
		if record.BatchID != "batch-previdenciario-digital" {
			continue
		}
		previdenciarioCount++
		openings[firstSentenceKey(record.Opening)]++
		documentGuidance[firstSentenceKey(record.DocumentGuidance)]++
		digitalTriage[firstSentenceKey(record.DigitalTriage)]++
	}

	if previdenciarioCount < 60 {
		t.Fatalf("previdenciario final drafts=%d, want at least 60 blocked drafts before 590 expansion", previdenciarioCount)
	}
	if len(openings) < 10 {
		t.Fatalf("previdenciario opening variants=%d, want at least 10 non-mechanical variants", len(openings))
	}
	if len(documentGuidance) < 10 {
		t.Fatalf("previdenciario document guidance variants=%d, want at least 10 non-mechanical variants", len(documentGuidance))
	}
	if len(digitalTriage) < 10 {
		t.Fatalf("previdenciario digital triage variants=%d, want at least 10 non-mechanical variants", len(digitalTriage))
	}
	maxAllowed := previdenciarioCount / 3
	if mostRepeated(openings) > maxAllowed {
		t.Fatalf("previdenciario opening repeated=%d, max allowed=%d", mostRepeated(openings), maxAllowed)
	}
	if mostRepeated(documentGuidance) > maxAllowed {
		t.Fatalf("previdenciario document guidance repeated=%d, max allowed=%d", mostRepeated(documentGuidance), maxAllowed)
	}
	if mostRepeated(digitalTriage) > maxAllowed {
		t.Fatalf("previdenciario digital triage repeated=%d, max allowed=%d", mostRepeated(digitalTriage), maxAllowed)
	}
}

func TestBatchFinalAuthorialDraftsAvoidMechanicalBeforeAnythingPhrase(t *testing.T) {
	records, loadReport := batchfinaldrafts.LoadRecords(".")
	if !loadReport.Passed() {
		t.Fatalf("could not load batch final authorial drafts: %v", loadReport.Messages())
	}

	for _, entry := range records {
		record := entry.Record
		if record.BatchID != "batch-previdenciario-digital" {
			continue
		}
		count := strings.Count(strings.ToLower(record.FullText()), "antes de qualquer")
		if count >= 3 {
			t.Fatalf("line=%d %s repeats 'antes de qualquer' %d times; final draft must polish archive snippets instead of compounding boilerplate", entry.Line, record.UniqueIntentID, count)
		}
	}
}

func TestBatchFinalAuthorialDraftRejectsSourceBlockedOrPublicDraft(t *testing.T) {
	index := batchfinaldrafts.ManifestIndex{
		BaseURL: "https://wikijuridica.com.br",
		EligibleByIntent: map[string]batchfinaldrafts.EligibleManifest{
			"consumidor-financeiro-pix-fraude-resposta-banco": {
				ManifestGateID:           "public-manifest-consumidor-financeiro-pix-fraude-resposta-banco",
				UniqueIntentID:           "consumidor-financeiro-pix-fraude-resposta-banco",
				BatchID:                  "batch-consumidor-financeiro-digital",
				SourceMatrixID:           "consumidor-financeiro-pix-fraude-resposta-banco",
				Term:                     "fraude via Pix e resposta insuficiente do banco",
				CandidatePath:            "/temas/consumidor-financeiro-pix-fraude-resposta-banco/",
				CandidateCanonicalURL:    "https://wikijuridica.com.br/temas/consumidor-financeiro-pix-fraude-resposta-banco/",
				CandidateTitle:           "Fraude via Pix e resposta insuficiente do banco",
				CandidateMetaDescription: "Entenda como cronologia, comprovante Pix, boletim e protocolos ajudam a avaliar resposta bancária insuficiente.",
				SelectedSourceURLs:       []string{"https://www.bcb.gov.br/estabilidadefinanceira/pix"},
			},
		},
		BlockedByIntent: map[string]bool{
			"familia-divorcio-consensual-filhos-bens": true,
		},
	}

	record := batchfinaldrafts.Record{
		DraftID:                  "bad-final-draft",
		ManifestGateID:           "public-manifest-familia-divorcio-consensual-filhos-bens",
		UniqueIntentID:           "familia-divorcio-consensual-filhos-bens",
		BatchID:                  "batch-familia-digital",
		SourceMatrixID:           "familia-divorcio-consensual-filhos-bens",
		Term:                     "divórcio consensual online com filhos e bens",
		CandidatePath:            "/temas/familia-divorcio-consensual-filhos-bens/?utm=1",
		CandidateCanonicalURL:    "https://portal-juridico.example/temas/familia-divorcio-consensual-filhos-bens/",
		CandidateTitle:           "Divórcio",
		CandidateMetaDescription: "Meta curta.",
		SelectedSourceURLs:       []string{"https://www.cnj.jus.br/"},
		DraftStatus:              "published",
		Language:                 "pt-BR",
		Opening:                  "Texto curto e genérico.",
		SourceUse:                "Sem fonte específica.",
		DocumentGuidance:         "Sem documentos.",
		DigitalTriage:            "Sem triagem.",
		CTAContextMessage:        "Mensagem sem origem.",
		InformationalNotice:      "Aviso fraco.",
		HumanScore:               100,
		AILikeScore:              0,
		IndexPolicy:              "index",
		RenderAllowed:            true,
		SitemapAllowed:           true,
		PublicationAllowed:       true,
		PublicPath:               "/temas/familia-divorcio-consensual-filhos-bens/",
		CheckedAt:                "2026-06-09",
	}

	report := batchfinaldrafts.ValidateRecordAgainstManifestIndex(record, index)
	if report.Passed() {
		t.Fatal("ValidateRecordAgainstManifestIndex passed, want blocked draft failures")
	}
	for _, code := range []string{
		"batch_final_draft_source_blocked",
		"batch_final_draft_candidate_path_not_clean",
		"batch_final_draft_canonical_mismatch",
		"batch_final_draft_title_too_short",
		"batch_final_draft_meta_too_short",
		"batch_final_draft_invalid_status",
		"batch_final_draft_invalid_index_policy",
		"batch_final_draft_text_failed_human_score",
		"batch_final_draft_text_failed_quality",
		"batch_final_draft_cta_not_contextual",
		"batch_final_draft_missing_notice",
		"batch_final_draft_render_allowed",
		"batch_final_draft_sitemap_allowed",
		"batch_final_draft_publication_allowed",
		"batch_final_draft_has_public_path",
	} {
		if !report.HasIssue(code) {
			t.Fatalf("missing issue %q in %v", code, report.Codes())
		}
	}
}

func firstSentenceKey(value string) string {
	value = strings.TrimSpace(strings.ToLower(value))
	for index, r := range value {
		if r == '.' || r == '!' || r == '?' {
			return strings.TrimSpace(value[:index+1])
		}
	}
	return value
}

func mostRepeated(counts map[string]int) int {
	max := 0
	for _, count := range counts {
		if count > max {
			max = count
		}
	}
	return max
}

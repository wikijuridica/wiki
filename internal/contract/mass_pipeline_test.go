package contract_test

import (
	"testing"

	"portaljuridico/internal/humanscore"
	"portaljuridico/internal/scalablebatches"
	"portaljuridico/internal/storage"
)

func TestContractsRequireAggressiveMassValidationAndPreCommitSelfReview(t *testing.T) {
	for _, path := range []string{"AGENTS.md", "GOAL.md"} {
		text := readRootFile(t, path)
		requireContains(t, text, "engenharia agressiva inteligente")
		requireContains(t, text, "Regra de planejamento sem chute")
		requireContains(t, text, "Regra de autocrítica antes do commit")
		requireContains(t, text, "Commit é checkpoint de continuidade, não conclusão do `/goal`")
		requireContains(t, text, "Regra de validação massiva antes de publicação")
		requireContains(t, text, "Regra de CPU no laboratório")
		requireContains(t, text, "laboratório, testes, auditorias, build e validações em massa podem usar CPU de forma agressiva")
	}
}

func TestStorageContractIncludesMassPipelineLayers(t *testing.T) {
	contract, err := storage.LoadContract(".")
	if err != nil {
		t.Fatal(err)
	}

	for _, layerName := range []string{"human_content_score", "scalable_content_batches"} {
		layer, ok := contract.LayerByName(layerName)
		if !ok {
			t.Fatalf("missing storage layer %s", layerName)
		}
		if layer.PublicIndexable {
			t.Fatalf("%s must not be directly indexable", layerName)
		}
		if layer.AllowsRawOfficialText || layer.AllowsEditorialContent {
			t.Fatalf("%s must store metadata/score only, not raw official text or page content", layerName)
		}
		if !layer.RequiresSourceProvenance || !layer.RequiresQualityState {
			t.Fatalf("%s must require source provenance and quality state", layerName)
		}
		if layer.RecordMaxBytes <= 0 || layer.RecordMaxBytes > 32768 {
			t.Fatalf("%s record budget=%d, want lightweight JSONL budget", layerName, layer.RecordMaxBytes)
		}
	}
}

func TestHumanContentScoreSeparatesUsefulLegalTextFromAILikeTemplate(t *testing.T) {
	natural := `Uma negativa de cobertura do plano de saúde costuma exigir leitura do contrato, do pedido médico e da resposta formal da operadora. A pessoa que recebeu a recusa precisa guardar protocolo, relatório clínico, indicação do procedimento, prazo informado e qualquer mensagem enviada pelo aplicativo ou pela central. A análise jurídica não começa pela promessa de liminar; começa pela comparação entre o caso concreto, a segmentação do plano, o Rol da ANS e a urgência documentada. Em atendimento digital, esses documentos podem ser enviados por WhatsApp para triagem inicial, permitindo avaliar se há falha de cobertura, risco de demora, alternativa administrativa e necessidade de medida judicial. O texto público deve orientar sem inventar resultado e deve explicar quais informações mudam a estratégia.`
	score := humanscore.ScoreText(natural)
	if score.HumanScore < 85 {
		t.Fatalf("HumanScore=%d, want >=85; issues=%v", score.HumanScore, score.Codes())
	}
	if score.AILikeScore > 20 {
		t.Fatalf("AILikeScore=%d, want <=20; issues=%v", score.AILikeScore, score.Codes())
	}
	if len(score.BlockingIssues) != 0 {
		t.Fatalf("natural text has blocking issues: %v", score.Codes())
	}

	mechanical := `Este conteúdo explica negativa cobertura plano saúde. Negativa cobertura plano saúde é importante. Negativa cobertura plano saúde precisa advogado. De forma geral, este conteúdo explica tudo sobre negativa cobertura plano saúde. Guia completo de negativa cobertura plano saúde para contratar advogado. Negativa cobertura plano saúde documentos necessários. Negativa cobertura plano saúde quando procurar advogado. Negativa cobertura plano saúde WhatsApp. Em resumo, negativa cobertura plano saúde é um tema importante e este conteúdo explica de forma geral a negativa cobertura plano saúde.`
	badScore := humanscore.ScoreText(mechanical)
	if badScore.HumanScore >= 60 {
		t.Fatalf("mechanical HumanScore=%d, want <60; issues=%v", badScore.HumanScore, badScore.Codes())
	}
	if badScore.AILikeScore < 50 {
		t.Fatalf("mechanical AILikeScore=%d, want >=50; issues=%v", badScore.AILikeScore, badScore.Codes())
	}
	for _, code := range []string{"ai_like_generic_markers", "low_specificity", "repeated_ngram", "keyword_density"} {
		if !badScore.HasIssue(code) {
			t.Fatalf("mechanical score missing issue %q in %v", code, badScore.Codes())
		}
	}
}

func TestScalableContentBatchesPrepareMillionPagePipelineWithoutPublication(t *testing.T) {
	report := scalablebatches.Validate(".")
	if !report.Passed() {
		t.Fatalf("scalable content batches failed contract: %v", report.Messages())
	}

	records, loadReport := scalablebatches.LoadRecords(".")
	if !loadReport.Passed() {
		t.Fatalf("could not load scalable batches: %v", loadReport.Messages())
	}
	if len(records) < 6 {
		t.Fatalf("batch count=%d, want at least 6 legal families", len(records))
	}
	if scalablebatches.TotalPlannedPages(records) < 1000000 {
		t.Fatalf("planned pages=%d, want at least 1000000", scalablebatches.TotalPlannedPages(records))
	}

	areas := make(map[string]bool)
	for _, entry := range records {
		record := entry.Record
		areas[record.LegalArea] = true
		if record.BatchStatus != "mass_generation_blocked" {
			t.Fatalf("%s status=%q, want mass_generation_blocked", record.BatchID, record.BatchStatus)
		}
		if record.GenerationStrategy != "unique_intent_mass_batch" {
			t.Fatalf("%s strategy=%q, want unique_intent_mass_batch", record.BatchID, record.GenerationStrategy)
		}
		if record.ValidationMode != "mass_batch_before_publication" {
			t.Fatalf("%s validation=%q, want mass_batch_before_publication", record.BatchID, record.ValidationMode)
		}
		if record.ReviewStrategy == "manual_review_page_by_page" {
			t.Fatalf("%s uses manual page-by-page bottleneck", record.BatchID)
		}
		if !record.DigitalOnly || !record.WhatsAppContextRequired || !record.AutoRewriteOnFail {
			t.Fatalf("%s missing digital/whatsapp/rewrite contract", record.BatchID)
		}
		if record.MinimumHumanScore < 85 || record.MaximumAILikeScore > 20 || record.MaximumSimilarity > 0.64 {
			t.Fatalf("%s weak quality thresholds: human=%d ai=%d similarity=%.2f", record.BatchID, record.MinimumHumanScore, record.MaximumAILikeScore, record.MaximumSimilarity)
		}
		if len(record.IntentDimensions) < 6 || len(record.SourceFamilies) < 2 || len(record.SampleValidationRules) < 6 {
			t.Fatalf("%s lacks dimensions/source/rules for mass validation", record.BatchID)
		}
		if record.RenderAllowed || record.SitemapAllowed || record.PublicationAllowed || record.PublicPath != "" {
			t.Fatalf("%s escaped blocked publication contract", record.BatchID)
		}
	}
	if len(areas) < 6 {
		t.Fatalf("legal area diversity=%d, want at least 6", len(areas))
	}
}

func TestScalableContentBatchRejectsSpamOrManualBottleneck(t *testing.T) {
	record := scalablebatches.Record{
		BatchID:                 "spam-keyword-cities",
		LegalArea:               "consumidor",
		BatchStatus:             "published",
		GenerationStrategy:      "keyword_city_permutation",
		ValidationMode:          "single_script",
		ReviewStrategy:          "manual_review_page_by_page",
		PlannedPageCount:        1000000,
		DigitalOnly:             false,
		WhatsAppContextRequired: false,
		AutoRewriteOnFail:       false,
		MinimumHumanScore:       50,
		MaximumAILikeScore:      70,
		MaximumSimilarity:       0.90,
		IntentDimensions:        []string{"cidade", "keyword"},
		SourceFamilies:          []string{"blog privado"},
		SampleValidationRules:   []string{"contar palavras"},
		RenderAllowed:           true,
		SitemapAllowed:          true,
		PublicationAllowed:      true,
		PublicPath:              "/temas/spam/",
		CheckedAt:               "2026-06-09",
	}

	report := scalablebatches.ValidateRecord(record)
	if report.Passed() {
		t.Fatal("ValidateRecord passed spam batch, want failures")
	}
	for _, code := range []string{
		"batch_invalid_status",
		"batch_template_strategy",
		"batch_single_script_validation",
		"batch_manual_review_bottleneck",
		"batch_not_digital_only",
		"batch_missing_whatsapp_context",
		"batch_rewrite_disabled",
		"batch_human_score_too_low",
		"batch_ai_score_too_high",
		"batch_similarity_too_high",
		"batch_too_few_dimensions",
		"batch_too_few_sources",
		"batch_too_few_validation_rules",
		"batch_render_allowed",
		"batch_sitemap_allowed",
		"batch_publication_allowed",
		"batch_has_public_path",
	} {
		if !report.HasIssue(code) {
			t.Fatalf("missing issue %q in %v", code, report.Codes())
		}
	}
}

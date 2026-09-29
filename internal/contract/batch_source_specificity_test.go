package contract_test

import (
	"testing"

	"portaljuridico/internal/batchsourcespecificity"
)

func TestBatchSourceSpecificityCoversPrepublicationWithoutPublishing(t *testing.T) {
	report := batchsourcespecificity.Validate(".")
	if !report.Passed() {
		t.Fatalf("batch source specificity failed contract: %v", report.Messages())
	}

	records, loadReport := batchsourcespecificity.LoadRecords(".")
	if !loadReport.Passed() {
		t.Fatalf("could not load batch source specificity resolutions: %v", loadReport.Messages())
	}
	index, indexReport := batchsourcespecificity.BuildSourceIndex(".")
	if !indexReport.Passed() {
		t.Fatalf("could not build source specificity index: %v", indexReport.Messages())
	}
	if len(records) != len(index.PrepublicationByIntent) {
		t.Fatalf("source specificity resolutions=%d prepublication=%d, want one record for each prepublication gate", len(records), len(index.PrepublicationByIntent))
	}
	if len(records) < 168 {
		t.Fatalf("source specificity resolutions=%d, want at least 168 expanded paid-passed candidates", len(records))
	}

	for _, entry := range records {
		record := entry.Record
		if record.CandidateRobots != "noindex,follow" {
			t.Fatalf("line=%d robots=%q, want noindex,follow", entry.Line, record.CandidateRobots)
		}
		if record.UsePolicy != "reference_only_no_scraping_no_ingestion" {
			t.Fatalf("line=%d use_policy=%q", entry.Line, record.UsePolicy)
		}
		if record.ScrapingAllowed || record.IngestionAllowed || record.RenderAllowed || record.SitemapAllowed || record.PublicationAllowed || record.PublicPath != "" {
			t.Fatalf("line=%d record escaped blocked contract: scraping=%t ingestion=%t render=%t sitemap=%t publication=%t path=%q", entry.Line, record.ScrapingAllowed, record.IngestionAllowed, record.RenderAllowed, record.SitemapAllowed, record.PublicationAllowed, record.PublicPath)
		}
		if record.SourceSpecificityStatus == "final_source_blocked_needs_specific_url" && (record.BlockingReason == "" || record.NeededSourceDetail == "") {
			t.Fatalf("line=%d blocked resolution must explain reason and needed detail", entry.Line)
		}
	}
}

func TestBatchSourceSpecificityPromotesAuditedSpecificMatricesAtScale(t *testing.T) {
	records, loadReport := batchsourcespecificity.LoadRecords(".")
	if !loadReport.Passed() {
		t.Fatalf("could not load batch source specificity resolutions: %v", loadReport.Messages())
	}

	lockedByIntent := make(map[string]batchsourcespecificity.Record)
	for _, entry := range records {
		record := entry.Record
		if record.SourceSpecificityStatus == batchsourcespecificity.LockedStatus {
			lockedByIntent[record.UniqueIntentID] = record
		}
	}

	if len(lockedByIntent) < 168 {
		t.Fatalf("source-locked candidates=%d, want at least 168 candidates promoted by audited specific official URLs", len(lockedByIntent))
	}

	for _, intentID := range []string{
		"consumidor-financeiro-pix-fraude-resposta-banco-documento-minimo",
		"familia-divorcio-consensual-filhos-bens-fonte-primaria",
		"previdenciario-auxilio-incapacidade-pericia-fonte-primaria",
		"saude-suplementar-prazo-consulta-especialista-competencia-digital",
		"sucessorio-inventario-extrajudicial-consenso-documento-minimo",
		"trabalhista-verbas-rescisorias-nao-pagas",
		"trabalhista-acidente-trabalho-estabilidade-documento-minimo",
		"familia-pensao-revisao-desemprego-documento-minimo",
		"consumidor-financeiro-negativacao-divida-desconhecida-fonte-primaria",
	} {
		record, ok := lockedByIntent[intentID]
		if !ok {
			t.Fatalf("intent %q stayed blocked even though its matrix has audited specific official source URLs", intentID)
		}
		if len(record.SelectedSourceURLs) == 0 {
			t.Fatalf("intent %q locked without selected source URLs", intentID)
		}
	}
}

func TestBatchSourceSpecificityRejectsUnauditedOrPublicResolution(t *testing.T) {
	index := batchsourcespecificity.SourceIndex{
		BaseURL: "https://wikijuridica.com.br",
		PrepublicationByIntent: map[string]batchsourcespecificity.PrepublicationCandidate{
			"familia-divorcio-consensual-filhos-bens": {
				PrepublicationID:      "prepub-familia-divorcio-consensual-filhos-bens",
				ReviewID:              "review-familia-divorcio-consensual-filhos-bens",
				BatchID:               "batch-familia-digital",
				UniqueIntentID:        "familia-divorcio-consensual-filhos-bens",
				SourceMatrixID:        "familia-divorcio-consensual-filhos-bens",
				Term:                  "divórcio consensual online com filhos e bens",
				CandidatePath:         "/temas/familia-divorcio-consensual-filhos-bens/",
				CandidateCanonicalURL: "https://wikijuridica.com.br/temas/familia-divorcio-consensual-filhos-bens/",
				CandidateRobots:       "noindex,follow",
			},
		},
		AuditsByURL: map[string]batchsourcespecificity.AuditedSource{
			"https://www.planalto.gov.br/ccivil_03/leis/2002/l10406compilada.htm": {
				SourceURL:    "https://www.planalto.gov.br/ccivil_03/leis/2002/l10406compilada.htm",
				MatrixIDs:    []string{"familia-divorcio-consensual-filhos-bens"},
				OfficialHost: true,
				AuditStatus:  "source_url_audited_reference_only_blocked",
				UsePolicy:    "reference_only_no_scraping_no_ingestion",
			},
		},
	}

	record := batchsourcespecificity.Record{
		ResolutionID:            "bad-source-specificity",
		PrepublicationID:        "prepub-familia-divorcio-consensual-filhos-bens",
		ReviewID:                "review-familia-divorcio-consensual-filhos-bens",
		BatchID:                 "batch-familia-digital",
		UniqueIntentID:          "familia-divorcio-consensual-filhos-bens",
		SourceMatrixID:          "fonte-errada",
		Term:                    "divórcio consensual online com filhos e bens",
		CandidatePath:           "/temas/familia-divorcio-consensual-filhos-bens/?utm=1",
		CandidateCanonicalURL:   "https://portal-juridico.example/temas/familia-divorcio-consensual-filhos-bens/",
		CandidateRobots:         "index,follow",
		SourceSpecificityStatus: "final_source_locked_reference_only",
		SelectedSourceURLs: []string{
			"https://www.planalto.gov.br/ccivil_03/leis/2002/l10406compilada.htm",
			"https://www.exemplo.invalid/fonte",
		},
		UsePolicy:          "copy_or_scrape",
		ScrapingAllowed:    true,
		IngestionAllowed:   true,
		RenderAllowed:      true,
		SitemapAllowed:     true,
		PublicationAllowed: true,
		PublicPath:         "/temas/familia-divorcio-consensual-filhos-bens/",
		CheckedAt:          "2026-06-09",
	}

	report := batchsourcespecificity.ValidateRecordAgainstSourceIndex(record, index)
	if report.Passed() {
		t.Fatal("ValidateRecordAgainstSourceIndex passed, want blocked source-specificity failures")
	}
	for _, code := range []string{
		"batch_source_specificity_source_matrix_mismatch",
		"batch_source_specificity_candidate_path_not_clean",
		"batch_source_specificity_canonical_mismatch",
		"batch_source_specificity_not_noindex",
		"batch_source_specificity_source_url_not_audited",
		"batch_source_specificity_source_url_not_linked",
		"batch_source_specificity_invalid_use_policy",
		"batch_source_specificity_scraping_allowed",
		"batch_source_specificity_ingestion_allowed",
		"batch_source_specificity_render_allowed",
		"batch_source_specificity_sitemap_allowed",
		"batch_source_specificity_publication_allowed",
		"batch_source_specificity_has_public_path",
	} {
		if !report.HasIssue(code) {
			t.Fatalf("missing issue %q in %v", code, report.Codes())
		}
	}
}

func TestBatchSourceSpecificityTreatsGeneralLegalCodesAsBroadSources(t *testing.T) {
	index := batchsourcespecificity.SourceIndex{
		BaseURL: "https://wikijuridica.com.br",
		PrepublicationByIntent: map[string]batchsourcespecificity.PrepublicationCandidate{
			"trabalhista-rescisao-indireta-assedio-salario": {
				PrepublicationID:      "prepub-trabalhista-rescisao-indireta-assedio-salario",
				ReviewID:              "review-trabalhista-rescisao-indireta-assedio-salario",
				BatchID:               "batch-trabalhista-digital",
				UniqueIntentID:        "trabalhista-rescisao-indireta-assedio-salario",
				SourceMatrixID:        "trabalhista-rescisao-indireta-assedio-salario",
				Term:                  "rescisão indireta por assédio e atraso salarial",
				CandidatePath:         "/temas/trabalhista-rescisao-indireta-assedio-salario/",
				CandidateCanonicalURL: "https://wikijuridica.com.br/temas/trabalhista-rescisao-indireta-assedio-salario/",
				CandidateRobots:       "noindex,follow",
			},
		},
		AuditsByURL: map[string]batchsourcespecificity.AuditedSource{
			"https://www.planalto.gov.br/ccivil_03/decreto-lei/del5452compilado.htm": {
				SourceURL:        "https://www.planalto.gov.br/ccivil_03/decreto-lei/del5452compilado.htm",
				MatrixIDs:        []string{"trabalhista-rescisao-indireta-assedio-salario"},
				OfficialHost:     true,
				AuditStatus:      "source_url_audited_reference_only_blocked",
				UsePolicy:        "reference_only_no_scraping_no_ingestion",
				SourceType:       "clt_compilada",
				ScrapingAllowed:  false,
				IngestionAllowed: false,
			},
		},
	}

	record := batchsourcespecificity.Record{
		ResolutionID:            "source-specificity-trabalhista-rescisao-indireta-assedio-salario",
		PrepublicationID:        "prepub-trabalhista-rescisao-indireta-assedio-salario",
		ReviewID:                "review-trabalhista-rescisao-indireta-assedio-salario",
		BatchID:                 "batch-trabalhista-digital",
		UniqueIntentID:          "trabalhista-rescisao-indireta-assedio-salario",
		SourceMatrixID:          "trabalhista-rescisao-indireta-assedio-salario",
		Term:                    "rescisão indireta por assédio e atraso salarial",
		CandidatePath:           "/temas/trabalhista-rescisao-indireta-assedio-salario/",
		CandidateCanonicalURL:   "https://wikijuridica.com.br/temas/trabalhista-rescisao-indireta-assedio-salario/",
		CandidateRobots:         "noindex,follow",
		SourceSpecificityStatus: "final_source_locked_reference_only",
		SelectedSourceURLs: []string{
			"https://www.planalto.gov.br/ccivil_03/decreto-lei/del5452compilado.htm",
		},
		UsePolicy: "reference_only_no_scraping_no_ingestion",
		CheckedAt: "2026-06-09",
	}

	report := batchsourcespecificity.ValidateRecordAgainstSourceIndex(record, index)
	if !report.HasIssue("batch_source_specificity_locked_without_specific_url") {
		t.Fatalf("general CLT source unlocked specific recorte; issues=%v", report.Codes())
	}
}

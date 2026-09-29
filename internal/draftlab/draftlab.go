package draftlab

import (
	"fmt"
	"strings"

	"portaljuridico/internal/quality"
	"portaljuridico/internal/terms"
)

type Draft struct {
	TermID      string
	Term        string
	Status      string
	IndexPolicy string
	PublicPath  string
	SourceID    string
	SourceURL   string
	Text        string
}

func Build(seed terms.Seed) (Draft, error) {
	if report := terms.ValidateSeed(seed); !report.Passed() {
		return Draft{}, fmt.Errorf("invalid_seed=%s", strings.Join(report.Messages(), " | "))
	}
	sourceName := sourceLabel(seed.SourceID)
	displayTerm := displayTerm(seed.Term)
	text := fmt.Sprintf(`%s é uma pauta jurídica que precisa ser tratada com contexto, fonte e revisão antes de qualquer publicação. Este rascunho de laboratório usa a fonte %s (%s), verificada em %s, apenas para orientar a pesquisa inicial e organizar perguntas úteis para o leitor.

Em uma página pública, o tema deve explicar quando a dúvida costuma aparecer, quais documentos ou fatos mudam a análise e por que a resposta depende do caso concreto. No caso de %s, o texto precisa separar a referência oficial, a explicação editorial e os cuidados práticos, sem inventar decisão, prazo, número de processo ou trecho legal.

O objetivo comercial pode existir, inclusive com atendimento por WhatsApp quando a página estiver aprovada, mas isso não pode atropelar fonte, revisão e qualidade. Antes de expor este assunto ao Googlebot, o conteúdo deve passar pelos gates de duplicidade, intenção única, linguagem natural em PT-BR, aviso informativo e revisão jurídica. Este rascunho não substitui consulta jurídica individual e não deve ser publicado como página indexável.`, capitalize(displayTerm), sourceName, seed.SourceURL, seed.CheckedAt, displayTerm)

	analysis := quality.AnalyzeText(text)
	if !analysis.Passed() {
		return Draft{}, fmt.Errorf("draft_quality_failed=%s", strings.Join(analysis.Messages(), " | "))
	}
	return Draft{
		TermID:      seed.TermID,
		Term:        displayTerm,
		Status:      "draft",
		IndexPolicy: "noindex",
		SourceID:    seed.SourceID,
		SourceURL:   seed.SourceURL,
		Text:        text,
	}, nil
}

func sourceLabel(sourceID string) string {
	switch sourceID {
	case "lexml":
		return "LexML"
	case "ans":
		return "ANS"
	case "camara-dados-abertos":
		return "Câmara dos Deputados"
	case "cnj":
		return "CNJ"
	case "inss":
		return "INSS"
	case "planalto":
		return "Planalto"
	case "previdencia":
		return "Previdência Social"
	case "stj":
		return "STJ"
	default:
		return sourceID
	}
}

func displayTerm(value string) string {
	normalized := strings.TrimSpace(strings.ToLower(value))
	replacements := map[string]string{
		"auxilio doenca negado":           "auxílio-doença negado",
		"desconto indevido inss":          "desconto indevido no INSS",
		"divorcio online":                 "divórcio online",
		"inventario extrajudicial online": "inventário extrajudicial online",
		"negativa cobertura plano saude":  "negativa de cobertura do plano de saúde",
		"pensao alimenticia":              "pensão alimentícia",
		"rescisao indireta":               "rescisão indireta",
		"advogado trabalhista online":     "advogado trabalhista online",
		"bpc loas negado":                 "BPC/LOAS negado",
		"dano moral consumidor":           "dano moral do consumidor",
		"responsabilidade civil":          "responsabilidade civil",
	}
	if replacement, ok := replacements[normalized]; ok {
		return replacement
	}
	return strings.TrimSpace(value)
}

func capitalize(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return value
	}
	runes := []rune(value)
	runes[0] = []rune(strings.ToUpper(string(runes[0])))[0]
	return string(runes)
}

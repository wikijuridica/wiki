package quality

import "testing"

func TestTextQualityAcceptsNaturalTemporaryLegalContent(t *testing.T) {
	text := "A pensão alimentícia pode ser discutida quando há mudança concreta na necessidade de quem recebe ou na capacidade econômica de quem paga. " +
		"Em uma análise responsável, o leitor precisa entender que documentos de renda, despesas essenciais, idade dos filhos e decisões anteriores influenciam o caminho jurídico. " +
		"O conteúdo informativo deve explicar o tema com calma, separar orientação geral de avaliação individual e indicar que um advogado pode examinar provas, prazos e riscos do caso. " +
		"Também é importante mostrar que acordos familiares precisam ser formalizados de maneira segura, porque soluções improvisadas podem gerar cobrança futura, conflito processual e insegurança para todos os envolvidos."

	analysis := AnalyzeText(text)

	if !analysis.Passed() {
		t.Fatalf("natural content failed quality analysis: %v", analysis.Messages())
	}
}

func TestTextQualityRejectsMechanicalKeywordPermutationBeforeGooglebot(t *testing.T) {
	text := "Advogado trabalhista em São Paulo. Advogado trabalhista em Campinas. Advogado trabalhista em Santos. " +
		"Advogado trabalhista em Sorocaba. Advogado trabalhista em Osasco. Advogado trabalhista em Guarulhos. " +
		"Contrate advogado trabalhista pelo WhatsApp. Advogado trabalhista rápido. Advogado trabalhista urgente. " +
		"Advogado trabalhista para rescisão. Advogado trabalhista para horas extras. Advogado trabalhista para justa causa. " +
		"Advogado trabalhista em São Paulo para rescisão. Advogado trabalhista em Campinas para rescisão. Advogado trabalhista em Santos para rescisão."

	analysis := AnalyzeText(text)

	for _, want := range []string{"mechanical_keyword_permutation", "keyword_stuffing"} {
		if !analysis.HasIssue(want) {
			t.Fatalf("missing issue %q in %v", want, analysis.Messages())
		}
	}
}

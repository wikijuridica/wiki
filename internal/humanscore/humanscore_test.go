package humanscore

import "testing"

func TestScoreTextDoesNotTreatRepeatedArticlesAsMonotoneStarts(t *testing.T) {
	text := "A pauta usa contrato, protocolo, extratos e resposta oficial para explicar o problema. " +
		"A triagem separa documentos, datas, valores e risco juridico antes de qualquer consulta. " +
		"A cautela evita promessa de resultado e mantém o conteúdo informativo para o leitor. " +
		"A fonte oficial sustenta o recorte sem copiar texto público ou criar pagina mecanica. " +
		"O atendimento digital organiza documentos e contexto para conversa objetiva com advogado."

	score := ScoreText(text)
	if score.HasIssue("monotone_sentence_starts") {
		t.Fatalf("article-only sentence starts were treated as monotone: %v", score.Codes())
	}
}

func TestScoreTextStillRejectsRelevantRepeatedNGrams(t *testing.T) {
	text := "Contrato banco fraude protocolo resposta contrato banco fraude protocolo resposta contrato banco fraude protocolo resposta. " +
		"Extratos mensagens contestacao e documento oficial aparecem, mas o trecho repete a mesma semantica central sem acrescentar contexto real. " +
		"A triagem juridica precisa separar fonte, risco, valor, documentos e consulta online antes de qualquer publicacao."

	score := ScoreText(text)
	if !score.HasIssue("repeated_ngram") {
		t.Fatalf("relevant repeated n-gram passed, codes=%v", score.Codes())
	}
}

func TestScoreTextRecognizesConcreteDigitalEvidenceWithoutGenericDocumentList(t *testing.T) {
	text := "A pessoa reuniu comprovante de transferência, boletim de ocorrência, protocolos eletrônicos, identidade, autenticação da instituição financeira e contestação registrada no aplicativo. " +
		"A análise separa horário, canal usado, resposta recebida, valor movimentado, titularidade da conta e preservação dos arquivos enviados pela central. " +
		"O texto informa limites do rascunho, origem da prova, risco de resposta incompleta e pergunta jurídica para atendimento online sem prometer resultado. " +
		"Também aponta que a avaliação depende de autoridade pública, histórico de comunicação, conferência individual dos arquivos e cuidado com orientação genérica. " +
		"A narrativa mantém fase, canal, pessoa responsável pelo contato, registro de contestação e dúvida principal separados para leitura técnica."

	score := ScoreText(text)
	if score.HasIssue("low_specificity") {
		t.Fatalf("concrete digital evidence was treated as low specificity: %v", score.Messages())
	}
}

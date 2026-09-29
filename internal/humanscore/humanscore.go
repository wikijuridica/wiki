package humanscore

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode"
)

type Issue struct {
	Code    string
	Message string
}

type Score struct {
	HumanScore     int
	AILikeScore    int
	BlockingIssues []Issue
}

type Record struct {
	ScoreID            string   `json:"score_id"`
	TermID             string   `json:"term_id"`
	BatchID            string   `json:"batch_id"`
	TextKind           string   `json:"text_kind"`
	ScoreStatus        string   `json:"score_status"`
	Language           string   `json:"language"`
	HumanScore         int      `json:"human_score"`
	AILikeScore        int      `json:"ai_like_score"`
	IssueCodes         []string `json:"issue_codes"`
	RewriteRequired    bool     `json:"rewrite_required"`
	RewriteAttempted   bool     `json:"rewrite_attempted"`
	PublicationAllowed bool     `json:"publication_allowed"`
	PublicPath         string   `json:"public_path"`
	CheckedAt          string   `json:"checked_at"`
}

type Entry struct {
	Line   int
	Record Record
}

type Report struct {
	Issues []Issue
}

var scoreStopWords = map[string]bool{
	"a": true, "o": true, "os": true, "as": true, "um": true, "uma": true, "de": true, "do": true, "da": true, "dos": true, "das": true,
	"e": true, "ou": true, "em": true, "no": true, "na": true, "nos": true, "nas": true, "para": true, "por": true, "com": true,
	"que": true, "se": true, "ao": true, "aos": true, "pela": true, "pelo": true, "pelas": true, "pelos": true, "isso": true,
	"este": true, "esta": true, "esse": true, "essa": true, "sobre": true, "tambem": true, "também": true,
}

var specificitySignals = map[string]bool{
	"contrato": true, "pedido": true, "médico": true, "medico": true, "resposta": true, "operadora": true, "protocolo": true,
	"relatório": true, "relatorio": true, "clínico": true, "clinico": true, "procedimento": true, "prazo": true, "aplicativo": true,
	"central": true, "ans": true, "rol": true, "urgência": true, "urgencia": true, "cobertura": true, "documentos": true,
	"prescrição": true, "prescricao": true, "especialista": true, "orçamento": true, "orcamento": true,
	"terapêutica": true, "terapeutica": true, "exclusão": true, "exclusao": true,
	"triagem": true, "administrativa": true, "judicial": true, "lei": true, "artigo": true, "processo": true, "inss": true,
	"benefício": true, "beneficio": true, "cartório": true, "cartorio": true, "notariado": true, "laudo": true,
	"clt": true, "holerites": true, "mensagens": true, "advertências": true, "advertencias": true, "testemunhas": true,
	"jornada": true, "rescisão": true, "rescisao": true, "verbas": true, "escala": true, "ponto": true,
	"casamento": true, "filhos": true, "guarda": true, "convivência": true, "convivencia": true, "pensão": true,
	"pensao": true, "renda": true, "acordo": true, "bens": true, "sentença": true, "sentenca": true, "despesas": true,
	"criança": true, "crianca": true, "residência": true, "residencia": true, "calendário": true, "calendario": true,
	"cadúnico": true, "cadunico": true, "cnis": true, "perícia": true, "pericia": true, "laudos": true,
	"exames": true, "atestados": true, "comunicado": true, "decisão": true, "decisao": true, "recurso": true,
	"cadastro": true, "dívida": true, "divida": true, "extratos": true, "fraude": true, "banco": true, "pix": true,
	"comprovante": true, "comprovantes": true, "boletim": true, "protocolos": true, "eletrônico": true,
	"eletronico": true, "eletrônicos": true, "eletronicos": true, "identidade": true, "autenticação": true,
	"autenticacao": true, "instituição": true, "instituicao": true, "titularidade": true, "transferência": true,
	"transferencia": true, "contestação": true, "contestacao": true,
	"certidão": true, "certidao": true, "óbito": true, "obito": true, "herdeiros": true, "matrícula": true,
	"matricula": true, "certidões": true, "certidoes": true, "testamento": true, "espólio": true, "espolio": true, "alvará": true, "alvara": true,
	"extrato": true, "imóvel": true, "imovel": true, "dívidas": true, "dividas": true, "imposto": true,
	"vínculo": true, "vinculo": true, "vínculos": true, "vinculos": true, "carteira": true, "guias": true,
	"ppp": true, "requerimento": true, "exigência": true, "exigencia": true, "fgts": true, "trct": true,
	"aviso": true, "pagamento": true, "recibos": true, "homologação": true, "homologacao": true,
}

func ScoreText(value string) Score {
	words := normalizedWords(value)
	signalWords := removeStopWords(words)
	issues := make([]Issue, 0)
	penalty := 0
	ai := 0

	if len(words) < 90 {
		issues = append(issues, Issue{Code: "thin_text", Message: fmt.Sprintf("texto com %d palavras; score humano exige amostra substantiva", len(words))})
		penalty += 12
		ai += 10
	}
	if genericMarkerCount(value) >= 2 {
		issues = append(issues, Issue{Code: "ai_like_generic_markers", Message: "texto usa marcadores genericos comuns em conteudo mecanico"})
		penalty += 22
		ai += 28
	}
	if specificityCount(words) < 5 {
		issues = append(issues, Issue{Code: "low_specificity", Message: "faltam documentos, fonte, prazo, orgao ou detalhe juridico concreto"})
		penalty += 25
		ai += 22
	}
	if code, message, ok := repeatedNGram(signalWords, 3, 3); ok {
		issues = append(issues, Issue{Code: code, Message: message})
		penalty += 24
		ai += 22
	}
	if code, message, ok := keywordDensity(signalWords); ok {
		issues = append(issues, Issue{Code: code, Message: message})
		penalty += 20
		ai += 20
	}
	if diversity(signalWords) < 0.42 && len(signalWords) >= 50 {
		issues = append(issues, Issue{Code: "low_lexical_diversity", Message: "diversidade lexical baixa para texto juridico informativo"})
		penalty += 14
		ai += 14
	}
	if monotoneSentenceStarts(value) {
		issues = append(issues, Issue{Code: "monotone_sentence_starts", Message: "sentencas iniciam de forma repetida"})
		penalty += 10
		ai += 10
	}

	human := clamp(100-penalty, 0, 100)
	return Score{
		HumanScore:     human,
		AILikeScore:    clamp(ai, 0, 100),
		BlockingIssues: issues,
	}
}

func Validate(root string) Report {
	entries, report := LoadRecords(root)
	if !report.Passed() {
		return report
	}
	issues := make([]Issue, 0)
	if len(entries) == 0 {
		issues = append(issues, Issue{Code: "human_scores_empty", Message: "data/editorial/human_content_scores.jsonl"})
	}
	for _, entry := range entries {
		recordReport := ValidateRecord(entry.Record)
		for _, issue := range recordReport.Issues {
			issue.Message = fmt.Sprintf("line=%d %s", entry.Line, issue.Message)
			issues = append(issues, issue)
		}
	}
	return Report{Issues: issues}
}

func LoadRecords(root string) ([]Entry, Report) {
	projectRoot, err := findProjectRoot(root)
	if err != nil {
		return nil, Report{Issues: []Issue{{Code: "project_root_not_found", Message: err.Error()}}}
	}
	path := filepath.Join(projectRoot, "data", "editorial", "human_content_scores.jsonl")
	file, err := os.Open(path)
	if err != nil {
		return nil, Report{Issues: []Issue{{Code: "human_scores_missing", Message: err.Error()}}}
	}
	defer file.Close()

	issues := make([]Issue, 0)
	entries := make([]Entry, 0)
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
			issues = append(issues, Issue{Code: "human_score_invalid_json", Message: fmt.Sprintf("line=%d %s", lineNumber, err.Error())})
			continue
		}
		entries = append(entries, Entry{Line: lineNumber, Record: record})
	}
	if err := scanner.Err(); err != nil {
		issues = append(issues, Issue{Code: "human_score_scan_failed", Message: err.Error()})
	}
	return entries, Report{Issues: issues}
}

func ValidateRecord(record Record) Report {
	issues := make([]Issue, 0)
	if record.ScoreID == "" || record.TermID == "" || record.BatchID == "" {
		issues = append(issues, Issue{Code: "human_score_missing_identity", Message: record.ScoreID})
	}
	if record.ScoreStatus != "human_content_score_passed_blocked_publication" && record.ScoreStatus != "human_content_score_rewrite_required" {
		issues = append(issues, Issue{Code: "human_score_invalid_status", Message: record.ScoreStatus})
	}
	if record.Language != "pt-BR" {
		issues = append(issues, Issue{Code: "human_score_not_ptbr", Message: record.ScoreID})
	}
	if record.HumanScore < 85 && !record.RewriteRequired {
		issues = append(issues, Issue{Code: "human_score_low_without_rewrite", Message: record.ScoreID})
	}
	if record.AILikeScore > 20 && !record.RewriteRequired {
		issues = append(issues, Issue{Code: "human_score_ai_like_without_rewrite", Message: record.ScoreID})
	}
	if record.RewriteRequired && !record.RewriteAttempted {
		issues = append(issues, Issue{Code: "human_score_rewrite_not_attempted", Message: record.ScoreID})
	}
	if record.PublicationAllowed {
		issues = append(issues, Issue{Code: "human_score_publication_allowed", Message: record.ScoreID})
	}
	if record.PublicPath != "" {
		issues = append(issues, Issue{Code: "human_score_has_public_path", Message: record.PublicPath})
	}
	if record.CheckedAt == "" {
		issues = append(issues, Issue{Code: "human_score_without_checked_at", Message: record.ScoreID})
	}
	return Report{Issues: issues}
}

func (s Score) HasIssue(code string) bool {
	for _, issue := range s.BlockingIssues {
		if issue.Code == code {
			return true
		}
	}
	return false
}

func (s Score) Codes() []string {
	codes := make([]string, 0, len(s.BlockingIssues))
	for _, issue := range s.BlockingIssues {
		codes = append(codes, issue.Code)
	}
	sort.Strings(codes)
	return codes
}

func (s Score) Messages() []string {
	messages := make([]string, 0, len(s.BlockingIssues))
	for _, issue := range s.BlockingIssues {
		messages = append(messages, issue.Code+": "+issue.Message)
	}
	return messages
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

func normalizedWords(value string) []string {
	normalized := strings.Map(func(r rune) rune {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			return unicode.ToLower(r)
		}
		return ' '
	}, value)
	return strings.Fields(normalized)
}

func removeStopWords(words []string) []string {
	filtered := make([]string, 0, len(words))
	for _, word := range words {
		if !scoreStopWords[word] && len(word) > 2 {
			filtered = append(filtered, word)
		}
	}
	return filtered
}

func genericMarkerCount(value string) int {
	normalized := strings.ToLower(value)
	markers := []string{
		"este conteúdo explica",
		"este conteudo explica",
		"de forma geral",
		"guia completo",
		"em resumo",
		"é importante",
		"e importante",
		"documentos necessários",
		"quando procurar advogado",
	}
	count := 0
	for _, marker := range markers {
		if strings.Contains(normalized, marker) {
			count++
		}
	}
	return count
}

func specificityCount(words []string) int {
	count := 0
	seen := make(map[string]bool)
	for _, word := range words {
		if specificitySignals[word] && !seen[word] {
			seen[word] = true
			count++
		}
	}
	return count
}

func repeatedNGram(words []string, size int, limit int) (string, string, bool) {
	if len(words) < size {
		return "", "", false
	}
	counts := make(map[string]int)
	for i := 0; i <= len(words)-size; i++ {
		key := strings.Join(words[i:i+size], " ")
		counts[key]++
		if counts[key] >= limit {
			return "repeated_ngram", fmt.Sprintf("n-grama %q repetido %d vezes", key, counts[key]), true
		}
	}
	return "", "", false
}

func keywordDensity(words []string) (string, string, bool) {
	if len(words) < 12 {
		return "", "", false
	}
	counts := make(map[string]int)
	topWord := ""
	topCount := 0
	for _, word := range words {
		counts[word]++
		if counts[word] > topCount {
			topWord = word
			topCount = counts[word]
		}
	}
	if float64(topCount)/float64(len(words)) > 0.14 && topCount >= 5 {
		return "keyword_density", fmt.Sprintf("termo %q aparece %d vezes em %d termos relevantes", topWord, topCount, len(words)), true
	}
	return "", "", false
}

func diversity(words []string) float64 {
	if len(words) == 0 {
		return 1
	}
	seen := make(map[string]bool)
	for _, word := range words {
		seen[word] = true
	}
	return float64(len(seen)) / float64(len(words))
}

func monotoneSentenceStarts(value string) bool {
	parts := strings.FieldsFunc(value, func(r rune) bool {
		return r == '.' || r == '!' || r == '?'
	})
	starts := make(map[string]int)
	for _, part := range parts {
		words := normalizedWords(part)
		if len(words) == 0 {
			continue
		}
		start := ""
		for _, word := range words {
			if !scoreStopWords[word] && len(word) > 2 {
				start = word
				break
			}
		}
		if start == "" {
			continue
		}
		starts[start]++
		if starts[start] >= 4 {
			return true
		}
	}
	return false
}

func clamp(value int, min int, max int) int {
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
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

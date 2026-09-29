package quality

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"unicode"

	"portaljuridico/internal/content"
	"portaljuridico/internal/editorial"
	"portaljuridico/internal/router"
	"portaljuridico/internal/seo"
	"portaljuridico/internal/sources"
)

const minIndexableWords = 90
const minInternalLinks = 2
const nearDuplicateThreshold = 0.82
const minLexicalDiversity = 0.38

type Issue struct {
	PagePath string
	Code     string
	Message  string
}

type Report struct {
	Issues []Issue
}

type TextIssue struct {
	Code    string
	Message string
}

type TextAnalysis struct {
	WordCount        int
	SignalWords      int
	LexicalDiversity float64
	Issues           []TextIssue
}

func (r Report) Passed() bool {
	return len(r.Issues) == 0
}

func (r Report) Codes() []string {
	codes := make([]string, 0, len(r.Issues))
	for _, issue := range r.Issues {
		codes = append(codes, issue.Code)
	}
	return codes
}

func (r Report) HasIssue(code string) bool {
	for _, issue := range r.Issues {
		if issue.Code == code {
			return true
		}
	}
	return false
}

func (r Report) Messages() []string {
	messages := make([]string, 0, len(r.Issues))
	for _, issue := range r.Issues {
		messages = append(messages, fmt.Sprintf("%s: %s: %s", issue.PagePath, issue.Code, issue.Message))
	}
	return messages
}

func ValidatePages(pages []content.Page) Report {
	issues := make([]Issue, 0)
	indexable := make([]content.Page, 0)

	for _, page := range pages {
		if !editorial.ValidStatuses[page.Status] {
			addIssue(&issues, page, "invalid_status", "status editorial desconhecido")
		}
		if !editorial.ValidIndexPolicies[page.IndexPolicy] {
			addIssue(&issues, page, "invalid_index_policy", "politica index/noindex invalida")
		}
		if !router.IsCleanPublicPath(page.Path) {
			addIssue(&issues, page, "unclean_url", "URL deve ser limpa, minuscula e sem parametros")
		}
		if !seo.IsAbsoluteHTTPSURL(page.CanonicalURL) {
			addIssue(&issues, page, "invalid_canonical", "canonical deve ser HTTPS absoluto")
		}
		if router.CanonicalPath(page.CanonicalURL) != page.Path {
			addIssue(&issues, page, "canonical_path_mismatch", "canonical deve apontar para a propria rota")
		}
		if editorial.IsIndexable(page) {
			indexable = append(indexable, page)
		}
	}

	for _, page := range indexable {
		if page.UniqueIntentID == "" {
			addIssue(&issues, page, "missing_unique_intent", "pagina indexavel sem unique_intent_id")
		}
		if page.Title == "" {
			addIssue(&issues, page, "missing_title", "pagina indexavel sem titulo")
		}
		if page.MetaDescription == "" {
			addIssue(&issues, page, "missing_meta_description", "pagina indexavel sem meta description")
		}
		if len(page.InternalLinks) < minInternalLinks {
			addIssue(&issues, page, "missing_useful_internal_links", "pagina indexavel exige links internos uteis")
		}
		if page.PublicationPurpose == "" {
			addIssue(&issues, page, "missing_publication_purpose", "pagina indexavel exige motivo de publicacao")
		}
		if page.IsLegalContent() && len(page.SourceProvenance) == 0 {
			addIssue(&issues, page, "legal_content_without_source", "conteudo juridico indexavel exige fonte")
		}
		if page.IsLegalContent() && (page.ReviewedAt == "" || page.Reviewer == "") {
			addIssue(&issues, page, "legal_content_without_review", "conteudo juridico indexavel exige revisao")
		}
		if page.IsLegalContent() && page.LegalNotice == "" {
			addIssue(&issues, page, "legal_content_without_notice", "conteudo juridico exige aviso informativo")
		}
		for _, textIssue := range AnalyzePage(page).Issues {
			addIssue(&issues, page, textIssue.Code, textIssue.Message)
		}
		for _, appearanceIssue := range seo.ValidateSearchAppearance(page) {
			addIssue(&issues, page, appearanceIssue.Code, appearanceIssue.Message)
		}
	}

	checkDuplicates(indexable, &issues)
	return Report{Issues: issues}
}

func ValidatePagesWithSources(pages []content.Page, registry sources.Registry) Report {
	report := ValidatePages(pages)
	issues := append([]Issue{}, report.Issues...)
	for _, page := range pages {
		if !editorial.IsIndexable(page) || !page.IsLegalContent() {
			continue
		}
		for _, provenance := range page.SourceProvenance {
			source, ok := registry.ByID(provenance.SourceID)
			if !ok {
				addIssue(&issues, page, "source_not_registered", "fonte juridica nao registrada")
				continue
			}
			if source.AuditStatus != "approved" || source.IngestionEnabled {
				addIssue(&issues, page, "source_not_approved_for_indexable_legal_content", "fonte ainda nao aprovada para conteudo juridico indexavel")
			}
		}
	}
	return Report{Issues: issues}
}

func addIssue(issues *[]Issue, page content.Page, code string, message string) {
	*issues = append(*issues, Issue{PagePath: page.Path, Code: code, Message: message})
}

func (a TextAnalysis) Passed() bool {
	return len(a.Issues) == 0
}

func (a TextAnalysis) HasIssue(code string) bool {
	for _, issue := range a.Issues {
		if issue.Code == code {
			return true
		}
	}
	return false
}

func (a TextAnalysis) Messages() []string {
	messages := make([]string, 0, len(a.Issues))
	for _, issue := range a.Issues {
		messages = append(messages, issue.Code+": "+issue.Message)
	}
	return messages
}

func AnalyzeText(value string) TextAnalysis {
	words := strings.Fields(normalizeText(value))
	signalWords := removeStopWords(words)
	issues := make([]TextIssue, 0)
	if len(words) < minIndexableWords {
		issues = append(issues, TextIssue{
			Code:    "thin_content",
			Message: fmt.Sprintf("texto com %d palavras; minimo indexavel e %d", len(words), minIndexableWords),
		})
	}

	lexicalDiversity := diversity(signalWords)
	if len(words) >= minIndexableWords && lexicalDiversity < minLexicalDiversity {
		issues = append(issues, TextIssue{
			Code:    "low_lexical_diversity",
			Message: fmt.Sprintf("diversidade lexical %.2f abaixo do minimo %.2f", lexicalDiversity, minLexicalDiversity),
		})
	}

	mechanicalSignals := 0
	if code, message, ok := keywordStuffing(signalWords); ok {
		issues = append(issues, TextIssue{Code: code, Message: message})
		mechanicalSignals++
	}
	if code, message, ok := repeatedPhrase(words); ok {
		issues = append(issues, TextIssue{Code: code, Message: message})
		mechanicalSignals++
	}
	if code, message, ok := repeatedSentence(value); ok {
		issues = append(issues, TextIssue{Code: code, Message: message})
		mechanicalSignals++
	}
	if lexicalDiversity < minLexicalDiversity && len(words) >= minIndexableWords {
		mechanicalSignals++
	}
	if mechanicalSignals >= 2 {
		issues = append(issues, TextIssue{
			Code:    "mechanical_keyword_permutation",
			Message: "texto combina repeticao, baixa diversidade ou permutacao de termos; bloquear antes do Googlebot",
		})
	}

	return TextAnalysis{
		WordCount:        len(words),
		SignalWords:      len(signalWords),
		LexicalDiversity: lexicalDiversity,
		Issues:           issues,
	}
}

func AnalyzePage(page content.Page) TextAnalysis {
	parts := []string{
		page.Summary,
		page.LegalNotice,
		page.PublicationPurpose,
	}
	for _, section := range page.BodySections {
		parts = append(parts, section.Title, section.Body)
	}
	return AnalyzeText(strings.Join(parts, " "))
}

func checkDuplicates(indexable []content.Page, issues *[]Issue) {
	checkField(indexable, issues, "duplicate_intent", "intencao unica duplicada", func(p content.Page) string {
		return p.UniqueIntentID
	})
	checkField(indexable, issues, "duplicate_title", "titulo duplicado", func(p content.Page) string {
		return p.Title
	})
	checkField(indexable, issues, "duplicate_meta_description", "meta description duplicada", func(p content.Page) string {
		return p.MetaDescription
	})
	checkField(indexable, issues, "duplicate_canonical", "canonical duplicado", func(p content.Page) string {
		return p.CanonicalURL
	})

	seenHashes := make(map[string]content.Page)
	for _, page := range indexable {
		digest := normalizedHash(page.PlainText())
		if previous, exists := seenHashes[digest]; exists {
			addIssue(issues, page, "duplicate_content_hash", "hash normalizado duplicado")
			addIssue(issues, previous, "duplicate_content_hash", "hash normalizado duplicado")
			continue
		}
		seenHashes[digest] = page
	}

	for i := 0; i < len(indexable); i++ {
		for j := i + 1; j < len(indexable); j++ {
			if shingleSimilarity(indexable[i].PlainText(), indexable[j].PlainText()) >= nearDuplicateThreshold {
				addIssue(issues, indexable[i], "near_duplicate_content", "conteudo parecido acima do limite")
				addIssue(issues, indexable[j], "near_duplicate_content", "conteudo parecido acima do limite")
			}
		}
	}
}

func checkField(indexable []content.Page, issues *[]Issue, code string, message string, value func(content.Page) string) {
	seen := make(map[string]content.Page)
	for _, page := range indexable {
		normalized := normalizeText(value(page))
		if previous, exists := seen[normalized]; exists {
			addIssue(issues, page, code, message)
			addIssue(issues, previous, code, message)
			continue
		}
		seen[normalized] = page
	}
}

func normalizeText(value string) string {
	lower := strings.ToLower(value)
	var clean strings.Builder
	for _, r := range lower {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			clean.WriteRune(r)
			continue
		}
		clean.WriteByte(' ')
	}
	return strings.Join(strings.Fields(clean.String()), " ")
}

func removeStopWords(words []string) []string {
	result := make([]string, 0, len(words))
	for _, word := range words {
		if len([]rune(word)) <= 2 || portugueseStopWords()[word] {
			continue
		}
		result = append(result, word)
	}
	return result
}

func portugueseStopWords() map[string]bool {
	return map[string]bool{
		"a": true, "ao": true, "aos": true, "as": true, "até": true,
		"com": true, "como": true, "da": true, "das": true, "de": true,
		"do": true, "dos": true, "e": true, "em": true, "entre": true,
		"essa": true, "esse": true, "esta": true, "este": true, "isso": true,
		"na": true, "nas": true, "no": true, "nos": true, "não": true,
		"o": true, "os": true, "ou": true, "para": true, "pela": true,
		"pelo": true, "por": true, "porque": true, "que": true, "se": true,
		"sem": true, "sua": true, "suas": true, "seu": true, "seus": true,
		"um": true, "uma": true,
	}
}

func diversity(words []string) float64 {
	if len(words) == 0 {
		return 0
	}
	unique := make(map[string]bool)
	for _, word := range words {
		unique[word] = true
	}
	return float64(len(unique)) / float64(len(words))
}

func keywordStuffing(words []string) (string, string, bool) {
	if len(words) == 0 {
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
	ratio := float64(topCount) / float64(len(words))
	if topCount >= 8 && ratio >= 0.12 {
		return "keyword_stuffing", fmt.Sprintf("termo %q aparece %d vezes em %d termos relevantes", topWord, topCount, len(words)), true
	}
	return "", "", false
}

func repeatedPhrase(words []string) (string, string, bool) {
	counts := shingleCounts(words, 3)
	topPhrase := ""
	topCount := 0
	for phrase, count := range counts {
		if count > topCount {
			topPhrase = phrase
			topCount = count
		}
	}
	if topCount >= 3 {
		return "repeated_phrase", fmt.Sprintf("frase %q repetida %d vezes", topPhrase, topCount), true
	}
	return "", "", false
}

func repeatedSentence(value string) (string, string, bool) {
	counts := make(map[string]int)
	for _, sentence := range splitSentences(value) {
		normalized := normalizeText(sentence)
		if wordCount(normalized) < 6 {
			continue
		}
		counts[normalized]++
		if counts[normalized] >= 3 {
			return "repeated_sentence", "mesma frase repetida tres ou mais vezes", true
		}
	}
	return "", "", false
}

func splitSentences(value string) []string {
	return strings.FieldsFunc(value, func(r rune) bool {
		return r == '.' || r == '!' || r == '?' || r == '\n'
	})
}

func shingleCounts(words []string, size int) map[string]int {
	result := make(map[string]int)
	if len(words) < size {
		return result
	}
	for i := 0; i <= len(words)-size; i++ {
		result[strings.Join(words[i:i+size], " ")]++
	}
	return result
}

func normalizedHash(value string) string {
	sum := sha256.Sum256([]byte(normalizeText(value)))
	return hex.EncodeToString(sum[:])
}

func wordCount(value string) int {
	normalized := normalizeText(value)
	if normalized == "" {
		return 0
	}
	return len(strings.Fields(normalized))
}

func shingles(value string, size int) map[string]bool {
	words := strings.Fields(normalizeText(value))
	result := make(map[string]bool)
	if len(words) < size {
		return result
	}
	for i := 0; i <= len(words)-size; i++ {
		result[strings.Join(words[i:i+size], " ")] = true
	}
	return result
}

func shingleSimilarity(left string, right string) float64 {
	leftSet := shingles(left, 5)
	rightSet := shingles(right, 5)
	if len(leftSet) == 0 || len(rightSet) == 0 {
		return 0
	}
	intersection := 0
	union := make(map[string]bool)
	for value := range leftSet {
		union[value] = true
		if rightSet[value] {
			intersection++
		}
	}
	for value := range rightSet {
		union[value] = true
	}
	return float64(intersection) / float64(len(union))
}

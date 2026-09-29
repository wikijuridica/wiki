package authorialdrafts

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"unicode"
)

type Section struct {
	Heading string `json:"heading"`
	Text    string `json:"text"`
}

type Record struct {
	TermID             string    `json:"term_id"`
	Term               string    `json:"term"`
	DraftStatus        string    `json:"draft_status"`
	Language           string    `json:"language"`
	SourceBriefID      string    `json:"source_brief_id"`
	UniqueAngle        string    `json:"unique_angle"`
	ReaderProblem      string    `json:"reader_problem"`
	Opening            string    `json:"opening"`
	BodySections       []Section `json:"body_sections"`
	DigitalCTAContext  string    `json:"digital_cta_context"`
	OfficialSourceURLs []string  `json:"official_source_urls"`
	PublicationAllowed bool      `json:"publication_allowed"`
	PublicPath         string    `json:"public_path"`
	CheckedAt          string    `json:"checked_at"`
}

type Entry struct {
	Line   int
	Record Record
}

type Issue struct {
	Code    string
	Message string
}

type Report struct {
	Issues []Issue
}

var termIDPattern = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)

func Validate(root string) Report {
	entries, report := LoadRecords(root)
	if !report.Passed() {
		return report
	}
	issues := make([]Issue, 0)
	if len(entries) == 0 {
		issues = append(issues, Issue{Code: "authorial_drafts_empty", Message: "data/editorial/authorial_drafts.jsonl"})
	}

	angles := make(map[string]int)
	sourceBriefs := make(map[string]int)
	openingPrefixes := make(map[string]int)
	for _, entry := range entries {
		recordReport := ValidateRecord(entry.Record)
		for _, issue := range recordReport.Issues {
			issue.Message = fmt.Sprintf("line=%d %s", entry.Line, issue.Message)
			issues = append(issues, issue)
		}

		angleKey := normalizeText(entry.Record.UniqueAngle)
		if angleKey != "" {
			if previousLine := angles[angleKey]; previousLine > 0 {
				issues = append(issues, Issue{Code: "authorial_draft_duplicate_angle", Message: fmt.Sprintf("line=%d previous_line=%d", entry.Line, previousLine)})
			}
			angles[angleKey] = entry.Line
		}

		briefKey := strings.TrimSpace(entry.Record.SourceBriefID)
		if briefKey != "" {
			if previousLine := sourceBriefs[briefKey]; previousLine > 0 {
				issues = append(issues, Issue{Code: "authorial_draft_duplicate_source_brief", Message: fmt.Sprintf("line=%d previous_line=%d source_brief_id=%s", entry.Line, previousLine, briefKey)})
			}
			sourceBriefs[briefKey] = entry.Line
		}

		prefix := firstNWords(entry.Record.Opening, 10)
		if prefix != "" {
			if previousLine := openingPrefixes[prefix]; previousLine > 0 {
				issues = append(issues, Issue{Code: "authorial_draft_reused_opening", Message: fmt.Sprintf("line=%d previous_line=%d", entry.Line, previousLine)})
			}
			openingPrefixes[prefix] = entry.Line
		}
	}

	issues = append(issues, detectRepeatedSectionShapes(entries)...)
	return Report{Issues: issues}
}

func LoadRecords(root string) ([]Entry, Report) {
	projectRoot, err := findProjectRoot(root)
	if err != nil {
		return nil, Report{Issues: []Issue{{Code: "project_root_not_found", Message: err.Error()}}}
	}
	path := filepath.Join(projectRoot, "data", "editorial", "authorial_drafts.jsonl")
	file, err := os.Open(path)
	if err != nil {
		return nil, Report{Issues: []Issue{{Code: "authorial_drafts_missing", Message: err.Error()}}}
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
			issues = append(issues, Issue{Code: "authorial_draft_invalid_json", Message: fmt.Sprintf("line=%d %s", lineNumber, err.Error())})
			continue
		}
		entries = append(entries, Entry{Line: lineNumber, Record: record})
	}
	if err := scanner.Err(); err != nil {
		issues = append(issues, Issue{Code: "authorial_draft_scan_failed", Message: err.Error()})
	}
	return entries, Report{Issues: issues}
}

func ValidateRecord(record Record) Report {
	issues := make([]Issue, 0)
	if record.TermID == "" || !termIDPattern.MatchString(record.TermID) || record.Term == "" {
		issues = append(issues, Issue{Code: "authorial_draft_invalid_term", Message: record.TermID})
	}
	if record.DraftStatus != "authorial_draft" {
		issues = append(issues, Issue{Code: "authorial_draft_invalid_status", Message: record.TermID + ":" + record.DraftStatus})
	}
	if record.Language != "pt-BR" {
		issues = append(issues, Issue{Code: "authorial_draft_not_ptbr", Message: record.TermID})
	}
	if record.SourceBriefID == "" || !termIDPattern.MatchString(record.SourceBriefID) {
		issues = append(issues, Issue{Code: "authorial_draft_missing_source_brief", Message: record.TermID})
	}
	if isGenericAngle(record.UniqueAngle) {
		issues = append(issues, Issue{Code: "authorial_draft_generic_angle", Message: record.TermID})
	}
	if len(normalizedWords(record.ReaderProblem)) < 8 {
		issues = append(issues, Issue{Code: "authorial_draft_thin_reader_problem", Message: record.TermID})
	}
	if len(normalizedWords(record.Opening)) < 35 || looksMechanicalOpening(record.Opening) {
		issues = append(issues, Issue{Code: "authorial_draft_thin_opening", Message: record.TermID})
	}
	if len(record.BodySections) < 4 {
		issues = append(issues, Issue{Code: "authorial_draft_too_few_sections", Message: record.TermID})
	}
	for _, section := range record.BodySections {
		if isGenericHeading(section.Heading) {
			issues = append(issues, Issue{Code: "authorial_draft_generic_heading", Message: record.TermID + ":" + section.Heading})
		}
		if len(normalizedWords(section.Text)) < 30 {
			issues = append(issues, Issue{Code: "authorial_draft_thin_section_text", Message: record.TermID + ":" + section.Heading})
		}
	}
	if len(normalizedWords(record.DigitalCTAContext)) < 18 || looksMechanicalCTA(record.DigitalCTAContext) {
		issues = append(issues, Issue{Code: "authorial_draft_thin_cta_context", Message: record.TermID})
	}
	if badToken := mechanicalPTBRToken(record); badToken != "" {
		issues = append(issues, Issue{Code: "authorial_draft_ptbr_spelling_failed", Message: record.TermID + ":" + badToken})
	}
	if len(record.OfficialSourceURLs) == 0 {
		issues = append(issues, Issue{Code: "authorial_draft_without_official_source", Message: record.TermID})
	}
	for _, url := range record.OfficialSourceURLs {
		if !isOfficialURL(url) {
			issues = append(issues, Issue{Code: "authorial_draft_untrusted_source_url", Message: record.TermID + ":" + url})
		}
	}
	if record.PublicationAllowed {
		issues = append(issues, Issue{Code: "authorial_draft_publication_allowed", Message: record.TermID})
	}
	if record.PublicPath != "" {
		issues = append(issues, Issue{Code: "authorial_draft_has_public_path", Message: record.PublicPath})
	}
	if record.CheckedAt == "" {
		issues = append(issues, Issue{Code: "authorial_draft_without_checked_at", Message: record.TermID})
	}
	return Report{Issues: issues}
}

func detectRepeatedSectionShapes(entries []Entry) []Issue {
	issues := make([]Issue, 0)
	for leftIndex := 0; leftIndex < len(entries); leftIndex++ {
		left := entries[leftIndex]
		leftHeadings := headingSet(left.Record.BodySections)
		for rightIndex := leftIndex + 1; rightIndex < len(entries); rightIndex++ {
			right := entries[rightIndex]
			shared := 0
			for heading := range headingSet(right.Record.BodySections) {
				if leftHeadings[heading] {
					shared++
				}
			}
			if shared >= 3 {
				issues = append(issues, Issue{Code: "authorial_draft_reused_section_shape", Message: fmt.Sprintf("line=%d other_line=%d shared_headings=%d", left.Line, right.Line, shared)})
			}
		}
	}
	return issues
}

func headingSet(sections []Section) map[string]bool {
	set := make(map[string]bool)
	for _, section := range sections {
		key := normalizeText(section.Heading)
		if key != "" {
			set[key] = true
		}
	}
	return set
}

func isGenericAngle(value string) bool {
	normalized := normalizeText(value)
	return len(strings.Fields(normalized)) < 7 ||
		strings.Contains(normalized, "conteudo informativo") ||
		strings.Contains(normalized, "explicar o tema") ||
		strings.Contains(normalized, "guia completo")
}

func isGenericHeading(value string) bool {
	normalized := normalizeText(value)
	generic := []string{
		"o que e",
		"quando procurar advogado",
		"documentos necessarios",
		"como funciona",
		"quanto custa",
		"introducao",
		"conclusao",
		"perguntas frequentes",
	}
	for _, item := range generic {
		if normalized == item || strings.Contains(normalized, item) {
			return true
		}
	}
	return len(strings.Fields(normalized)) < 4
}

func looksMechanicalOpening(value string) bool {
	normalized := normalizeText(value)
	mechanical := []string{
		"este conteudo explica",
		"este texto explica",
		"neste conteudo voce vai entender",
		"de forma geral",
		"guia completo",
	}
	for _, item := range mechanical {
		if strings.Contains(normalized, item) {
			return true
		}
	}
	return false
}

func looksMechanicalCTA(value string) bool {
	normalized := normalizeText(value)
	if strings.TrimSpace(normalized) == "chamar no whatsapp" || strings.TrimSpace(normalized) == "pode chamar no whatsapp" {
		return true
	}
	return len(strings.Fields(normalized)) < 18
}

func mechanicalPTBRToken(record Record) string {
	text := strings.Join(editorialFields(record), " ")
	tokens := make(map[string]bool)
	for _, word := range rawPTBRWords(text) {
		tokens[word] = true
	}
	mechanical := []string{
		"acao",
		"analise",
		"assedio",
		"avaliacao",
		"beneficio",
		"carencia",
		"codigo",
		"conteudo",
		"cronologica",
		"decisao",
		"disponivel",
		"divorcio",
		"existencia",
		"frequencia",
		"historico",
		"hipoteses",
		"indenizacao",
		"interpretacao",
		"juridica",
		"juridico",
		"medica",
		"medico",
		"nao",
		"orientacao",
		"possivel",
		"pratica",
		"previdenciaria",
		"propria",
		"publica",
		"publico",
		"reclamacao",
		"referencia",
		"regulatoria",
		"responsavel",
		"sequencia",
		"situacao",
		"situacoes",
		"unica",
		"urgencia",
		"vinculo",
	}
	for _, token := range mechanical {
		if tokens[token] {
			return token
		}
	}
	return ""
}

func editorialFields(record Record) []string {
	fields := []string{
		record.Term,
		record.UniqueAngle,
		record.ReaderProblem,
		record.Opening,
		record.DigitalCTAContext,
	}
	for _, section := range record.BodySections {
		fields = append(fields, section.Heading, section.Text)
	}
	return fields
}

func rawPTBRWords(value string) []string {
	words := make([]string, 0)
	current := strings.Builder{}
	for _, r := range strings.ToLower(value) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			current.WriteRune(r)
			continue
		}
		if current.Len() > 0 {
			words = append(words, current.String())
			current.Reset()
		}
	}
	if current.Len() > 0 {
		words = append(words, current.String())
	}
	return words
}

func isOfficialURL(value string) bool {
	prefixes := []string{
		"https://www.gov.br/",
		"https://www.cnj.jus.br/",
		"https://www.planalto.gov.br/",
	}
	for _, prefix := range prefixes {
		if strings.HasPrefix(value, prefix) {
			return true
		}
	}
	return false
}

func firstNWords(value string, limit int) string {
	words := normalizedWords(value)
	if len(words) < limit {
		return ""
	}
	return strings.Join(words[:limit], " ")
}

func normalizeText(value string) string {
	words := normalizedWords(value)
	return strings.Join(words, " ")
}

func normalizedWords(value string) []string {
	words := make([]string, 0)
	current := strings.Builder{}
	for _, r := range strings.ToLower(value) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			current.WriteRune(stripAccent(r))
			continue
		}
		if current.Len() > 0 {
			words = append(words, current.String())
			current.Reset()
		}
	}
	if current.Len() > 0 {
		words = append(words, current.String())
	}
	return words
}

func stripAccent(r rune) rune {
	switch r {
	case 'á', 'à', 'â', 'ã', 'ä':
		return 'a'
	case 'é', 'è', 'ê', 'ë':
		return 'e'
	case 'í', 'ì', 'î', 'ï':
		return 'i'
	case 'ó', 'ò', 'ô', 'õ', 'ö':
		return 'o'
	case 'ú', 'ù', 'û', 'ü':
		return 'u'
	case 'ç':
		return 'c'
	default:
		return r
	}
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

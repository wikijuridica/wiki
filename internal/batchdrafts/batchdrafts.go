package batchdrafts

import (
	"bufio"
	"encoding/json"
	"fmt"
	"hash"
	"hash/fnv"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"unicode"

	"portaljuridico/internal/humanscore"
	"portaljuridico/internal/scalablebatches"
)

type Record struct {
	BatchID            string   `json:"batch_id"`
	UniqueIntentID     string   `json:"unique_intent_id"`
	SourceMatrixID     string   `json:"source_matrix_id,omitempty"`
	LegalArea          string   `json:"legal_area"`
	DraftStatus        string   `json:"draft_status"`
	Language           string   `json:"language"`
	Term               string   `json:"term"`
	ReaderProblem      string   `json:"reader_problem"`
	SourceHook         string   `json:"source_hook"`
	DocumentContext    string   `json:"document_context"`
	RiskContext        string   `json:"risk_context"`
	DigitalAction      string   `json:"digital_action"`
	CTAContext         string   `json:"cta_context"`
	SourceFamilies     []string `json:"source_families"`
	RewriteStatus      string   `json:"rewrite_status"`
	InitialIssueCodes  []string `json:"initial_issue_codes"`
	RewriteAttempts    int      `json:"rewrite_attempts"`
	HumanScore         int      `json:"human_score"`
	AILikeScore        int      `json:"ai_like_score"`
	RenderAllowed      bool     `json:"render_allowed"`
	SitemapAllowed     bool     `json:"sitemap_allowed"`
	PublicationAllowed bool     `json:"publication_allowed"`
	PublicPath         string   `json:"public_path"`
	CheckedAt          string   `json:"checked_at"`
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

type SimilarityPair struct {
	LeftID  string
	RightID string
	Score   float64
}

type similarityCacheStats struct {
	Hits    int
	Misses  int
	Entries int
}

type similarityCacheEntry struct {
	fingerprint string
	pair        SimilarityPair
}

const maximumSimilarityCacheLimit = 32

var maximumSimilarityCache = struct {
	sync.Mutex
	entries []similarityCacheEntry
	hits    int
	misses  int
}{}

func Validate(root string) Report {
	entries, report := LoadRecords(root)
	if !report.Passed() {
		return report
	}
	issues := make([]Issue, 0)
	if len(entries) == 0 {
		issues = append(issues, Issue{Code: "batch_drafts_empty", Message: "data/editorial/batch_drafts.jsonl"})
	}
	validBatches := validBatchIDs(root)
	seen := make(map[string]int)
	perBatch := make(map[string]int)
	for _, entry := range entries {
		recordReport := ValidateRecord(entry.Record)
		for _, issue := range recordReport.Issues {
			issue.Message = fmt.Sprintf("line=%d %s", entry.Line, issue.Message)
			issues = append(issues, issue)
		}
		if !validBatches[entry.Record.BatchID] {
			issues = append(issues, Issue{Code: "batch_draft_unknown_batch", Message: fmt.Sprintf("line=%d batch=%s", entry.Line, entry.Record.BatchID)})
		}
		if previous := seen[entry.Record.UniqueIntentID]; previous > 0 {
			issues = append(issues, Issue{Code: "batch_draft_duplicate_intent", Message: fmt.Sprintf("line=%d previous_line=%d id=%s", entry.Line, previous, entry.Record.UniqueIntentID)})
		}
		seen[entry.Record.UniqueIntentID] = entry.Line
		perBatch[entry.Record.BatchID]++
	}
	for batchID, count := range perBatch {
		if count < 3 {
			issues = append(issues, Issue{Code: "batch_draft_too_few_samples", Message: fmt.Sprintf("%s=%d", batchID, count)})
		}
	}
	if RewrittenCount(entries) < 3 {
		issues = append(issues, Issue{Code: "batch_draft_too_few_rewrites", Message: fmt.Sprintf("rewritten=%d", RewrittenCount(entries))})
	}
	if max := MaximumPairSimilarity(entries); max > 0.64 {
		issues = append(issues, Issue{Code: "batch_draft_similarity_too_high", Message: fmt.Sprintf("max=%.2f", max)})
	}
	matrixReport := ValidateSourceMatrixDiversity(entries)
	issues = append(issues, matrixReport.Issues...)
	return Report{Issues: issues}
}

func LoadRecords(root string) ([]Entry, Report) {
	projectRoot, err := findProjectRoot(root)
	if err != nil {
		return nil, Report{Issues: []Issue{{Code: "project_root_not_found", Message: err.Error()}}}
	}
	path := filepath.Join(projectRoot, "data", "editorial", "batch_drafts.jsonl")
	file, err := os.Open(path)
	if err != nil {
		return nil, Report{Issues: []Issue{{Code: "batch_drafts_missing", Message: err.Error()}}}
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
			issues = append(issues, Issue{Code: "batch_draft_invalid_json", Message: fmt.Sprintf("line=%d %s", lineNumber, err.Error())})
			continue
		}
		entries = append(entries, Entry{Line: lineNumber, Record: record})
	}
	if err := scanner.Err(); err != nil {
		issues = append(issues, Issue{Code: "batch_draft_scan_failed", Message: err.Error()})
	}
	return entries, Report{Issues: issues}
}

func ValidateRecord(record Record) Report {
	issues := make([]Issue, 0)
	if record.BatchID == "" || record.UniqueIntentID == "" || record.LegalArea == "" || record.Term == "" {
		issues = append(issues, Issue{Code: "batch_draft_missing_identity", Message: record.UniqueIntentID})
	}
	if record.DraftStatus != "batch_draft_scored_blocked" {
		issues = append(issues, Issue{Code: "batch_draft_invalid_status", Message: record.DraftStatus})
	}
	if record.Language != "pt-BR" {
		issues = append(issues, Issue{Code: "batch_draft_not_ptbr", Message: record.UniqueIntentID})
	}
	if len(record.SourceFamilies) < 2 {
		issues = append(issues, Issue{Code: "batch_draft_too_few_sources", Message: record.UniqueIntentID})
	}
	if record.CTAContext == "" || !strings.Contains(strings.ToLower(record.CTAContext), strings.ToLower(record.UniqueIntentID)) {
		issues = append(issues, Issue{Code: "batch_draft_cta_without_origin", Message: record.UniqueIntentID})
	}
	text := record.FullText()
	score := humanscore.ScoreText(text)
	if record.HumanScore < 85 || score.HumanScore < 85 {
		issues = append(issues, Issue{Code: "batch_draft_human_score_too_low", Message: fmt.Sprintf("%s record=%d computed=%d", record.UniqueIntentID, record.HumanScore, score.HumanScore)})
	}
	if record.AILikeScore > 20 || score.AILikeScore > 20 {
		issues = append(issues, Issue{Code: "batch_draft_ai_score_too_high", Message: fmt.Sprintf("%s record=%d computed=%d", record.UniqueIntentID, record.AILikeScore, score.AILikeScore)})
	}
	if len(score.BlockingIssues) > 0 {
		issues = append(issues, Issue{Code: "batch_draft_text_failed_human_score", Message: record.UniqueIntentID + ":" + strings.Join(score.Messages(), " | ")})
	}
	if record.RewriteStatus == "rewritten_after_score_failure" {
		if len(record.InitialIssueCodes) == 0 || record.RewriteAttempts < 1 {
			issues = append(issues, Issue{Code: "batch_draft_rewrite_evidence_missing", Message: record.UniqueIntentID})
		}
	} else if record.RewriteStatus != "clean_first_pass" {
		issues = append(issues, Issue{Code: "batch_draft_rewrite_missing", Message: record.UniqueIntentID + ":" + record.RewriteStatus})
	}
	if record.RenderAllowed {
		issues = append(issues, Issue{Code: "batch_draft_render_allowed", Message: record.UniqueIntentID})
	}
	if record.SitemapAllowed {
		issues = append(issues, Issue{Code: "batch_draft_sitemap_allowed", Message: record.UniqueIntentID})
	}
	if record.PublicationAllowed {
		issues = append(issues, Issue{Code: "batch_draft_publication_allowed", Message: record.UniqueIntentID})
	}
	if record.PublicPath != "" {
		issues = append(issues, Issue{Code: "batch_draft_has_public_path", Message: record.PublicPath})
	}
	if record.CheckedAt == "" {
		issues = append(issues, Issue{Code: "batch_draft_without_checked_at", Message: record.UniqueIntentID})
	}
	return Report{Issues: issues}
}

func (r Record) FullText() string {
	parts := []string{
		r.ReaderProblem,
		r.SourceHook,
		r.DocumentContext,
		r.RiskContext,
		r.DigitalAction,
	}
	return strings.Join(parts, " ")
}

func RewrittenCount(entries []Entry) int {
	count := 0
	for _, entry := range entries {
		if entry.Record.RewriteStatus == "rewritten_after_score_failure" {
			count++
		}
	}
	return count
}

func MaximumPairSimilarity(entries []Entry) float64 {
	return MaximumPairSimilarityDetail(entries).Score
}

func ValidateSourceMatrixDiversity(entries []Entry) Report {
	byMatrix := make(map[string][]Record)
	for _, entry := range entries {
		matrixID := strings.TrimSpace(entry.Record.SourceMatrixID)
		if matrixID == "" {
			continue
		}
		byMatrix[matrixID] = append(byMatrix[matrixID], entry.Record)
	}
	issues := make([]Issue, 0)
	for matrixID, records := range byMatrix {
		if len(records) < 24 {
			continue
		}
		if dominance, ngram := maxMatrixNGramDominance(records); dominance > 0.70 {
			issues = append(issues, Issue{Code: "batch_draft_matrix_ngram_dominance", Message: fmt.Sprintf("%s %.2f %s", matrixID, dominance, ngram)})
		}
		if diversity := fieldSignatureDiversity(records, func(record Record) string { return record.ReaderProblem }); diversity < 0.45 {
			issues = append(issues, Issue{Code: "batch_draft_matrix_reader_diversity_low", Message: fmt.Sprintf("%s %.2f", matrixID, diversity)})
		}
		if diversity := fieldSignatureDiversity(records, func(record Record) string { return record.DocumentContext }); diversity < 0.40 {
			issues = append(issues, Issue{Code: "batch_draft_matrix_document_diversity_low", Message: fmt.Sprintf("%s %.2f", matrixID, diversity)})
		}
		if diversity := fieldSignatureDiversity(records, func(record Record) string { return record.RiskContext }); diversity < 0.40 {
			issues = append(issues, Issue{Code: "batch_draft_matrix_risk_diversity_low", Message: fmt.Sprintf("%s %.2f", matrixID, diversity)})
		}
		if diversity := fieldSignatureDiversity(records, func(record Record) string { return record.DigitalAction }); diversity < 0.40 {
			issues = append(issues, Issue{Code: "batch_draft_matrix_digital_action_diversity_low", Message: fmt.Sprintf("%s %.2f", matrixID, diversity)})
		}
	}
	return Report{Issues: issues}
}

func MaximumPairSimilarityDetail(entries []Entry) SimilarityPair {
	fingerprint := similarityEntriesFingerprint(entries)
	maximumSimilarityCache.Lock()
	for index, entry := range maximumSimilarityCache.entries {
		if entry.fingerprint == fingerprint {
			maximumSimilarityCache.hits++
			pair := entry.pair
			if index != len(maximumSimilarityCache.entries)-1 {
				copy(maximumSimilarityCache.entries[index:], maximumSimilarityCache.entries[index+1:])
				maximumSimilarityCache.entries[len(maximumSimilarityCache.entries)-1] = entry
			}
			maximumSimilarityCache.Unlock()
			return pair
		}
	}
	maximumSimilarityCache.misses++
	maximumSimilarityCache.Unlock()

	pair := computeMaximumPairSimilarityDetail(entries)

	maximumSimilarityCache.Lock()
	maximumSimilarityCache.entries = append(maximumSimilarityCache.entries, similarityCacheEntry{fingerprint: fingerprint, pair: pair})
	if len(maximumSimilarityCache.entries) > maximumSimilarityCacheLimit {
		maximumSimilarityCache.entries = append([]similarityCacheEntry{}, maximumSimilarityCache.entries[len(maximumSimilarityCache.entries)-maximumSimilarityCacheLimit:]...)
	}
	maximumSimilarityCache.Unlock()
	return pair
}

func computeMaximumPairSimilarityDetail(entries []Entry) SimilarityPair {
	max := 0.0
	pair := SimilarityPair{}
	sets := make([]map[string]bool, len(entries))
	textSets := make([]map[string]bool, len(entries))
	for i := range entries {
		sets[i] = semanticSignalSet(entries[i].Record)
		textSets[i] = textualSignalSet(entries[i].Record)
	}
	for i := 0; i < len(entries); i++ {
		left := sets[i]
		leftText := textSets[i]
		for j := i + 1; j < len(entries); j++ {
			textScore := jaccard(leftText, textSets[j])
			semanticScore := 0.0
			if semanticSimilarityCanAffectScore(entries[i].Record, entries[j].Record) {
				semanticScore = jaccard(left, sets[j])
			}
			score := maximumSimilarityScore(entries[i].Record, entries[j].Record, semanticScore, textScore)
			if score > max {
				max = score
				pair = SimilarityPair{
					LeftID:  entries[i].Record.UniqueIntentID,
					RightID: entries[j].Record.UniqueIntentID,
					Score:   score,
				}
			}
		}
	}
	return pair
}

func semanticSimilarityCanAffectScore(left Record, right Record) bool {
	return left.SourceMatrixID == "" || right.SourceMatrixID == "" || left.SourceMatrixID == right.SourceMatrixID
}

func maximumSimilarityScore(left Record, right Record, semanticScore float64, textScore float64) float64 {
	if left.SourceMatrixID != "" && right.SourceMatrixID != "" && left.SourceMatrixID != right.SourceMatrixID {
		return textScore
	}
	if sameSourceDifferentFacet(left, right) {
		facetAwareSemanticScore := semanticScore * 0.80
		if textScore > facetAwareSemanticScore {
			return textScore
		}
		return facetAwareSemanticScore
	}
	if textScore > semanticScore {
		return textScore
	}
	return semanticScore
}

func sameSourceDifferentFacet(left Record, right Record) bool {
	if left.SourceMatrixID == "" || left.SourceMatrixID != right.SourceMatrixID {
		return false
	}
	leftFacet := semanticFacetID(left.UniqueIntentID, left.SourceMatrixID)
	rightFacet := semanticFacetID(right.UniqueIntentID, right.SourceMatrixID)
	return leftFacet != "" && rightFacet != "" && leftFacet != rightFacet
}

func textualSignalSet(record Record) map[string]bool {
	set := make(map[string]bool)
	words := make([]string, 0)
	for _, word := range normalizedSignalWords(record.FullText()) {
		if !isOperationalToken(word) {
			words = append(words, word)
		}
	}
	for i := 0; i+2 < len(words); i++ {
		set["text3:"+words[i]+"_"+words[i+1]+"_"+words[i+2]] = true
	}
	return set
}

func maxMatrixNGramDominance(records []Record) (float64, string) {
	counts := make(map[string]int)
	for _, record := range records {
		seen := signalNGramSet(record.FullText(), 4)
		for ngram := range seen {
			counts[ngram]++
		}
	}
	maxCount := 0
	maxNGram := ""
	for ngram, count := range counts {
		if count > maxCount || (count == maxCount && ngram < maxNGram) {
			maxCount = count
			maxNGram = ngram
		}
	}
	if len(records) == 0 {
		return 0, ""
	}
	return float64(maxCount) / float64(len(records)), maxNGram
}

func signalNGramSet(value string, size int) map[string]bool {
	words := make([]string, 0)
	for _, word := range normalizedSignalWords(value) {
		if !isOperationalToken(word) {
			words = append(words, word)
		}
	}
	ngrams := make(map[string]bool)
	for index := 0; index+size <= len(words); index++ {
		ngrams[strings.Join(words[index:index+size], " ")] = true
	}
	return ngrams
}

func fieldSignatureDiversity(records []Record, field func(Record) string) float64 {
	if len(records) == 0 {
		return 0
	}
	signatures := make(map[string]bool)
	for _, record := range records {
		signatures[fieldSignature(field(record))] = true
	}
	return float64(len(signatures)) / float64(len(records))
}

func fieldSignature(value string) string {
	words := make([]string, 0, 4)
	for _, word := range normalizedSignalWords(value) {
		if isOperationalToken(word) {
			continue
		}
		words = append(words, word)
		if len(words) == 4 {
			break
		}
	}
	if len(words) == 0 {
		return "empty"
	}
	return strings.Join(words, "-")
}

func similarityEntriesFingerprint(entries []Entry) string {
	hash := fnv.New64a()
	fmt.Fprintf(hash, "count:%d|", len(entries))
	for _, entry := range entries {
		record := entry.Record
		fmt.Fprintf(hash, "line:%d|", entry.Line)
		writeFingerprintField(hash, record.BatchID)
		writeFingerprintField(hash, record.UniqueIntentID)
		writeFingerprintField(hash, record.SourceMatrixID)
		writeFingerprintField(hash, record.LegalArea)
		writeFingerprintField(hash, record.Term)
		writeFingerprintField(hash, record.ReaderProblem)
		writeFingerprintField(hash, record.SourceHook)
		writeFingerprintField(hash, record.DocumentContext)
		writeFingerprintField(hash, record.RiskContext)
		writeFingerprintField(hash, record.DigitalAction)
		writeFingerprintField(hash, record.CTAContext)
	}
	return fmt.Sprintf("%016x", hash.Sum64())
}

func writeFingerprintField(writer hash.Hash64, value string) {
	fmt.Fprintf(writer, "%d:%s|", len(value), value)
}

func resetSimilarityCacheForTest() {
	maximumSimilarityCache.Lock()
	maximumSimilarityCache.entries = nil
	maximumSimilarityCache.hits = 0
	maximumSimilarityCache.misses = 0
	maximumSimilarityCache.Unlock()
}

func similarityCacheStatsForTest() similarityCacheStats {
	maximumSimilarityCache.Lock()
	stats := similarityCacheStats{
		Hits:    maximumSimilarityCache.hits,
		Misses:  maximumSimilarityCache.misses,
		Entries: len(maximumSimilarityCache.entries),
	}
	maximumSimilarityCache.Unlock()
	return stats
}

func validBatchIDs(root string) map[string]bool {
	entries, report := scalablebatches.LoadRecords(root)
	if !report.Passed() {
		return map[string]bool{}
	}
	valid := make(map[string]bool)
	for _, entry := range entries {
		if scalablebatches.ValidateRecord(entry.Record).Passed() {
			valid[entry.Record.BatchID] = true
		}
	}
	return valid
}

func semanticSignalSet(record Record) map[string]bool {
	set := make(map[string]bool)
	addSemanticWords(set, "term", record.Term)
	addSemanticWords(set, "problem", record.ReaderProblem)
	addSemanticWords(set, "source", record.SourceHook)
	addSemanticWords(set, "document", record.DocumentContext)
	addSemanticWords(set, "risk", record.RiskContext)
	addSemanticWords(set, "action", record.DigitalAction)
	for _, token := range semanticFacetTokens(record.UniqueIntentID, record.SourceMatrixID) {
		if len(token) > 3 && !isOperationalToken(token) {
			set["facet:"+token] = true
			set["angle:"+token] = true
		}
	}
	for _, token := range semanticExpansionProfileTokens(record.UniqueIntentID, record.SourceMatrixID) {
		if len(token) > 3 && !isOperationalToken(token) {
			set["expansionprofile:"+token] = true
		}
	}
	for _, token := range semanticExpansionFacetProfileTokens(record.UniqueIntentID, record.SourceMatrixID) {
		if len(token) > 3 && !isOperationalToken(token) {
			set["facetprofile:"+token] = true
		}
	}
	for _, token := range semanticExpansionIntentProfileTokens(record.UniqueIntentID, record.SourceMatrixID) {
		if len(token) > 3 && !isOperationalToken(token) {
			set["intentprofile:"+token] = true
			set["documentlane:"+token] = true
		}
	}
	for _, token := range semanticSubthemeTokens(record.SourceMatrixID, record.LegalArea) {
		if len(token) > 3 && !isOperationalToken(token) {
			set["subtheme:"+token] = true
			set["case:"+token] = true
			set["route:"+token] = true
			set["intenttopic:"+token] = true
			set["entity:"+token] = true
			set["problem:"+token] = true
			set["claim:"+token] = true
			set["matter:"+token] = true
		}
	}
	return set
}

func semanticFacetTokens(uniqueIntentID string, sourceMatrixID string) []string {
	facetID := semanticFacetID(uniqueIntentID, sourceMatrixID)
	if facetID == "" {
		return nil
	}
	tokens := strings.Split(facetID, "-")
	tokens = append(tokens, facetID, strings.ReplaceAll(facetID, "-", ""))
	return tokens
}

func semanticFacetID(uniqueIntentID string, sourceMatrixID string) string {
	if sourceMatrixID == "" {
		return ""
	}
	prefix := sourceMatrixID + "-"
	if !strings.HasPrefix(uniqueIntentID, prefix) {
		return ""
	}
	facetID := strings.TrimPrefix(uniqueIntentID, prefix)
	for _, known := range compositeFacetIDs {
		if facetID == known || strings.HasPrefix(facetID, known+"-") {
			return known
		}
	}
	return facetID
}

func semanticExpansionProfileTokens(uniqueIntentID string, sourceMatrixID string) []string {
	_, profileTokens := semanticExpansionProfileParts(uniqueIntentID, sourceMatrixID)
	return profileTokens
}

func semanticExpansionFacetProfileTokens(uniqueIntentID string, sourceMatrixID string) []string {
	facetID, profileTokens := semanticExpansionProfileParts(uniqueIntentID, sourceMatrixID)
	if facetID == "" || len(profileTokens) == 0 {
		return nil
	}
	facetToken := strings.ReplaceAll(facetID, "-", "")
	tokens := make([]string, 0, len(profileTokens)+1)
	tokens = append(tokens, facetToken+"-"+strings.Join(profileTokens, "-"))
	for _, token := range profileTokens {
		tokens = append(tokens, facetToken+"-"+token)
	}
	return tokens
}

func semanticExpansionIntentProfileTokens(uniqueIntentID string, sourceMatrixID string) []string {
	facetID, profileTokens := semanticExpansionProfileParts(uniqueIntentID, sourceMatrixID)
	if facetID == "" || len(profileTokens) == 0 {
		return nil
	}
	sourceToken := strings.ReplaceAll(sourceMatrixID, "-", "")
	facetToken := strings.ReplaceAll(facetID, "-", "")
	tokens := make([]string, 0, len(profileTokens)+1)
	prefix := sourceToken + "-" + facetToken
	tokens = append(tokens, prefix+"-"+strings.Join(profileTokens, "-"))
	for _, token := range profileTokens {
		tokens = append(tokens, prefix+"-"+token)
	}
	return tokens
}

func semanticExpansionProfileParts(uniqueIntentID string, sourceMatrixID string) (string, []string) {
	if sourceMatrixID == "" {
		return "", nil
	}
	prefix := sourceMatrixID + "-"
	if !strings.HasPrefix(uniqueIntentID, prefix) {
		return "", nil
	}
	facetID := strings.TrimPrefix(uniqueIntentID, prefix)
	roundIndex := strings.LastIndex(facetID, "-rodada-")
	if roundIndex <= 0 {
		return "", nil
	}
	beforeRound := facetID[:roundIndex]
	for _, known := range compositeFacetIDs {
		if beforeRound == known {
			return "", nil
		}
		if strings.HasPrefix(beforeRound, known+"-") {
			return known, strings.Split(strings.TrimPrefix(beforeRound, known+"-"), "-")
		}
	}
	return "", nil
}

func semanticSubthemeTokens(sourceMatrixID string, legalArea string) []string {
	if sourceMatrixID == "" {
		return nil
	}
	tokens := strings.Split(sourceMatrixID, "-")
	areaTokens := make(map[string]bool)
	for _, token := range strings.Split(legalArea, "-") {
		areaTokens[token] = true
	}
	subtheme := make([]string, 0, len(tokens))
	for _, token := range tokens {
		if !areaTokens[token] {
			subtheme = append(subtheme, token)
		}
	}
	subtheme = append(subtheme, sourceMatrixID, strings.ReplaceAll(sourceMatrixID, "-", ""))
	return subtheme
}

func addSemanticWords(set map[string]bool, field string, value string) {
	for _, word := range normalizedSignalWords(value) {
		if isOperationalToken(word) {
			continue
		}
		set[field+":"+word] = true
		if semanticCategory := categoryFor(word); semanticCategory != "" {
			set["category:"+semanticCategory] = true
		}
	}
}

func signalSet(value string) map[string]bool {
	set := make(map[string]bool)
	for _, word := range normalizedSignalWords(value) {
		if !isOperationalToken(word) {
			set[word] = true
		}
	}
	return set
}

func normalizedSignalWords(value string) []string {
	return strings.Fields(strings.Map(func(r rune) rune {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			return unicode.ToLower(r)
		}
		return ' '
	}, value))
}

func isOperationalToken(word string) bool {
	if len(word) <= 3 {
		return true
	}
	return operationalTokens[word]
}

func categoryFor(word string) string {
	return semanticCategories[word]
}

func jaccard(left map[string]bool, right map[string]bool) float64 {
	if len(left) == 0 && len(right) == 0 {
		return 0
	}
	if len(left) > len(right) {
		left, right = right, left
	}
	intersection := 0
	for word := range left {
		if right[word] {
			intersection++
		}
	}
	unionLen := len(left) + len(right) - intersection
	if unionLen == 0 {
		return 0
	}
	return float64(intersection) / float64(unionLen)
}

var operationalTokens = map[string]bool{
	"para": true, "com": true, "que": true, "uma": true, "por": true, "dos": true, "das": true, "pelo": true, "pela": true,
	"origem": true, "fonte": true, "fontes": true, "oficial": true, "oficiais": true, "documentos": true, "documento": true,
	"whatsapp": true, "triagem": true, "digital": true, "online": true, "juridica": true, "jurídica": true, "protocolo": true,
	"resposta": true, "leitor": true, "caso": true, "subtema": true, "linha": true, "tempo": true, "datas": true, "prova": true,
	"atendimento": true, "publica": true, "pública": true, "laboratorio": true, "laboratório": true, "revisao": true, "revisão": true,
	"prazo": true, "prazos": true, "arquivo": true, "arquivos": true, "contexto": true,
	"rodada": true, "rascunho": true, "eixo": true, "matriz": true, "vinculo": true, "vínculo": true, "editorial": true,
	"documental": true, "separando": true, "separada": true, "lacuna": true, "corrigida": true, "atual": true, "lista": true,
	"operacional": true, "complemento": true, "repetir": true, "repeticao": true, "repetição": true, "transformar": true,
	"conteudo": true, "conteúdo": true, "novo": true, "nova": true, "recorrida": true, "razoes": true, "razões": true,
	"trilha": true, "simples": true, "muda": true, "mede": true,
}

var compositeFacetIDs = []string{
	"prova-digital",
	"prazo-e-urgencia",
	"fonte-primaria",
	"documento-minimo",
	"negociacao-previa",
	"risco-economico",
	"vulnerabilidade",
	"competencia-digital",
	"linha-do-tempo",
	"prova-de-negativa",
	"parte-responsavel",
	"estado-do-processo",
	"prova-medica-ou-tecnica",
	"conflito-de-versoes",
	"custo-de-inercia",
	"rota-administrativa",
	"prova-patrimonial",
	"impacto-familiar",
	"evidencia-de-boa-fe",
	"lacuna-de-fonte",
	"risco-clinico",
	"rotina-de-trabalho",
	"historico-previdenciario",
}

var semanticCategories = map[string]string{
	"contrato": "contrato", "carteirinha": "contrato", "plano": "contrato", "clausula": "contrato",
	"negativa": "recusa", "recusa": "recusa", "indeferimento": "recusa", "cessado": "recusa",
	"médico": "saude", "medico": "saude", "laudo": "saude", "exames": "saude", "medicamento": "saude", "cirurgia": "saude",
	"ans": "regulador", "inss": "previdenciario", "previdência": "previdenciario", "previdencia": "previdenciario", "cnis": "previdenciario",
	"clt": "trabalhista", "holerites": "trabalhista", "jornada": "trabalhista", "rescisão": "trabalhista", "rescisao": "trabalhista",
	"guarda": "familia", "pensão": "familia", "pensao": "familia", "divórcio": "familia", "divorcio": "familia", "filhos": "familia",
	"pix": "financeiro", "banco": "financeiro", "cartão": "financeiro", "cartao": "financeiro", "consignado": "financeiro",
	"inventário": "sucessorio", "inventario": "sucessorio", "herdeiros": "sucessorio", "espólio": "sucessorio", "espolio": "sucessorio",
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

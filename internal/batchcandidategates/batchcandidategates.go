package batchcandidategates

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"portaljuridico/internal/batchdraftarchive"
	"portaljuridico/internal/batchdrafts"
	"portaljuridico/internal/content"
	"portaljuridico/internal/router"
)

type Record struct {
	GateID                  string   `json:"gate_id"`
	GateGroupID             string   `json:"gate_group_id,omitempty"`
	ShardIndex              int      `json:"shard_index,omitempty"`
	ShardCount              int      `json:"shard_count,omitempty"`
	SelectedTotal           int      `json:"selected_total,omitempty"`
	BatchID                 string   `json:"batch_id"`
	GateStatus              string   `json:"gate_status"`
	SourceArchivePath       string   `json:"source_archive_path"`
	ArchiveMinimumRecords   int      `json:"archive_minimum_records"`
	SelectedUniqueIntentIDs []string `json:"selected_unique_intent_ids"`
	CandidatePathPrefix     string   `json:"candidate_path_prefix"`
	BaseURLMode             string   `json:"base_url_mode"`
	OfficialURLLocked       bool     `json:"official_url_locked"`
	MaxSimilarityAllowed    float64  `json:"max_similarity_allowed"`
	MaxSimilarityObserved   float64  `json:"max_similarity_observed"`
	MinimumHumanScore       int      `json:"minimum_human_score"`
	CTAContextRequired      bool     `json:"cta_context_required"`
	SourceURLAuditRequired  bool     `json:"source_url_audit_required"`
	PublicationBlockReason  string   `json:"publication_block_reason"`
	RenderAllowed           bool     `json:"render_allowed"`
	SitemapAllowed          bool     `json:"sitemap_allowed"`
	PublicationAllowed      bool     `json:"publication_allowed"`
	PublicPath              string   `json:"public_path"`
	CheckedAt               string   `json:"checked_at"`
}

type Entry struct {
	Line   int
	Record Record
}

type ArchiveIndex struct {
	ByIntent          map[string]batchdrafts.Record
	CountByBatch      map[string]int
	MaxSimilarity     float64
	BaseURLMode       string
	OfficialURLLocked bool
}

type Issue struct {
	Code    string
	Message string
}

type Report struct {
	Issues []Issue
}

var idPattern = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)

const MaxSelectedIntentIDsPerRecord = 120

func Validate(root string) Report {
	entries, report := LoadRecords(root)
	if !report.Passed() {
		return report
	}
	issues := make([]Issue, 0)
	index, indexReport := BuildArchiveIndex(root)
	issues = append(issues, indexReport.Issues...)
	issues = append(issues, validateEntriesAgainstArchive(entries, index)...)
	return Report{Issues: issues}
}

func validateEntriesAgainstArchive(entries []Entry, index ArchiveIndex) []Issue {
	issues := make([]Issue, 0)
	seenGateID := make(map[string]int)
	batches := make(map[string]*batchShardState)
	for _, entry := range entries {
		recordReport := ValidateRecordAgainstArchive(entry.Record, index)
		for _, issue := range recordReport.Issues {
			issue.Message = fmt.Sprintf("line=%d %s", entry.Line, issue.Message)
			issues = append(issues, issue)
		}
		if previous := seenGateID[entry.Record.GateID]; previous > 0 {
			issues = append(issues, Issue{Code: "batch_candidate_duplicate_gate_id", Message: fmt.Sprintf("line=%d previous_line=%d gate=%s", entry.Line, previous, entry.Record.GateID)})
		}
		seenGateID[entry.Record.GateID] = entry.Line
		state := batches[entry.Record.BatchID]
		if state == nil {
			state = &batchShardState{indices: make(map[int]int), intentLines: make(map[string]int)}
			batches[entry.Record.BatchID] = state
		}
		baseID := baseGateID(entry.Record)
		if state.groupID == "" {
			state.groupID = baseID
		} else if state.groupID != baseID {
			state.issues = append(state.issues, Issue{Code: "batch_candidate_multiple_gate_groups_for_batch", Message: fmt.Sprintf("line=%d batch=%s group=%s expected=%s", entry.Line, entry.Record.BatchID, baseID, state.groupID)})
		}
		state.selected += len(entry.Record.SelectedUniqueIntentIDs)
		for _, intentID := range entry.Record.SelectedUniqueIntentIDs {
			if previous := state.intentLines[intentID]; previous > 0 {
				state.issues = append(state.issues, Issue{Code: "batch_candidate_duplicate_selected_intent_in_batch", Message: fmt.Sprintf("line=%d previous_line=%d batch=%s intent=%s", entry.Line, previous, entry.Record.BatchID, intentID)})
			}
			state.intentLines[intentID] = entry.Line
		}
		if hasShardMetadata(entry.Record) {
			state.sharded = true
			if state.shardCount == 0 {
				state.shardCount = entry.Record.ShardCount
			} else if state.shardCount != entry.Record.ShardCount {
				state.issues = append(state.issues, Issue{Code: "batch_candidate_shard_count_mismatch", Message: fmt.Sprintf("line=%d batch=%s count=%d expected=%d", entry.Line, entry.Record.BatchID, entry.Record.ShardCount, state.shardCount)})
			}
			if state.selectedTotal == 0 {
				state.selectedTotal = entry.Record.SelectedTotal
			} else if state.selectedTotal != entry.Record.SelectedTotal {
				state.issues = append(state.issues, Issue{Code: "batch_candidate_shard_total_mismatch", Message: fmt.Sprintf("line=%d batch=%s total=%d expected=%d", entry.Line, entry.Record.BatchID, entry.Record.SelectedTotal, state.selectedTotal)})
			}
			if previous := state.indices[entry.Record.ShardIndex]; previous > 0 {
				state.issues = append(state.issues, Issue{Code: "batch_candidate_duplicate_shard_index", Message: fmt.Sprintf("line=%d previous_line=%d batch=%s index=%d", entry.Line, previous, entry.Record.BatchID, entry.Record.ShardIndex)})
			}
			state.indices[entry.Record.ShardIndex] = entry.Line
		} else {
			state.unsharded = true
		}
	}
	if len(batches) != 6 {
		issues = append(issues, Issue{Code: "batch_candidate_gate_batch_count_invalid", Message: fmt.Sprintf("batches=%d records=%d", len(batches), len(entries))})
	}
	for batchID, state := range batches {
		issues = append(issues, validateShardState(batchID, state)...)
	}
	return issues
}

type batchShardState struct {
	selected      int
	sharded       bool
	unsharded     bool
	groupID       string
	shardCount    int
	selectedTotal int
	indices       map[int]int
	intentLines   map[string]int
	issues        []Issue
}

func validateShardState(batchID string, state *batchShardState) []Issue {
	issues := make([]Issue, 0)
	issues = append(issues, state.issues...)
	if state.selected < 18 {
		issues = append(issues, Issue{Code: "batch_candidate_too_few_selected_intents", Message: fmt.Sprintf("%s=%d", batchID, state.selected)})
	}
	if state.sharded && state.unsharded {
		issues = append(issues, Issue{Code: "batch_candidate_mixed_shard_metadata", Message: batchID})
	}
	if !state.sharded {
		return issues
	}
	if state.selectedTotal != state.selected {
		issues = append(issues, Issue{Code: "batch_candidate_shard_total_mismatch", Message: fmt.Sprintf("%s selected=%d total=%d", batchID, state.selected, state.selectedTotal)})
	}
	if len(state.indices) != state.shardCount {
		issues = append(issues, Issue{Code: "batch_candidate_shard_count_mismatch", Message: fmt.Sprintf("%s shards=%d expected=%d", batchID, len(state.indices), state.shardCount)})
	}
	for index := 1; index <= state.shardCount; index++ {
		if state.indices[index] == 0 {
			issues = append(issues, Issue{Code: "batch_candidate_shard_index_missing", Message: fmt.Sprintf("%s=%d", batchID, index)})
		}
	}
	return issues
}

func hasShardMetadata(record Record) bool {
	return record.GateGroupID != "" || record.ShardCount > 0 || record.ShardIndex > 0 || record.SelectedTotal > 0
}

func BuildArchiveIndex(root string) (ArchiveIndex, Report) {
	archiveEntries, archiveReport := batchdraftarchive.LoadRecords(root)
	if !archiveReport.Passed() {
		return ArchiveIndex{}, Report{Issues: convertArchiveIssues(archiveReport)}
	}
	repo, err := content.LoadRepository(root)
	if err != nil {
		return ArchiveIndex{}, Report{Issues: []Issue{{Code: "batch_candidate_site_config_unavailable", Message: err.Error()}}}
	}
	index := ArchiveIndex{
		ByIntent:          make(map[string]batchdrafts.Record),
		CountByBatch:      make(map[string]int),
		MaxSimilarity:     batchdrafts.MaximumPairSimilarity(archiveEntries),
		BaseURLMode:       repo.BaseURLMode,
		OfficialURLLocked: repo.OfficialURLLocked,
	}
	for _, entry := range archiveEntries {
		index.ByIntent[entry.Record.UniqueIntentID] = entry.Record
		index.CountByBatch[entry.Record.BatchID]++
	}
	return index, Report{}
}

func ValidateRecordAgainstArchive(record Record, index ArchiveIndex) Report {
	issues := make([]Issue, 0)
	if record.GateID == "" || !idPattern.MatchString(record.GateID) {
		issues = append(issues, Issue{Code: "batch_candidate_invalid_gate_id", Message: record.GateID})
	}
	if record.GateGroupID != "" && !idPattern.MatchString(record.GateGroupID) {
		issues = append(issues, Issue{Code: "batch_candidate_invalid_gate_group_id", Message: record.GateGroupID})
	}
	if hasShardMetadata(record) {
		if record.GateGroupID == "" {
			issues = append(issues, Issue{Code: "batch_candidate_shard_without_group_id", Message: record.GateID})
		}
		if record.ShardIndex <= 0 || record.ShardCount <= 0 || record.ShardIndex > record.ShardCount {
			issues = append(issues, Issue{Code: "batch_candidate_invalid_shard_index", Message: fmt.Sprintf("%s index=%d count=%d", record.GateID, record.ShardIndex, record.ShardCount)})
		}
		if record.ShardCount == 1 {
			issues = append(issues, Issue{Code: "batch_candidate_single_shard_metadata", Message: record.GateID})
		}
		expectedGateID := fmt.Sprintf("%s-shard-%03d", record.GateGroupID, record.ShardIndex)
		if record.GateGroupID != "" && record.ShardIndex > 0 && record.GateID != expectedGateID {
			issues = append(issues, Issue{Code: "batch_candidate_shard_gate_id_mismatch", Message: fmt.Sprintf("%s expected=%s", record.GateID, expectedGateID)})
		}
		if record.SelectedTotal < len(record.SelectedUniqueIntentIDs) {
			issues = append(issues, Issue{Code: "batch_candidate_invalid_selected_total", Message: fmt.Sprintf("%s total=%d selected=%d", record.GateID, record.SelectedTotal, len(record.SelectedUniqueIntentIDs))})
		}
	}
	if record.BatchID == "" || !idPattern.MatchString(record.BatchID) {
		issues = append(issues, Issue{Code: "batch_candidate_invalid_batch_id", Message: record.BatchID})
	}
	if record.GateStatus != "blocked_batch_candidate" {
		issues = append(issues, Issue{Code: "batch_candidate_status_not_blocked", Message: record.GateStatus})
	}
	if record.SourceArchivePath != "data/editorial/batch_draft_expansion_archive.jsonl" {
		issues = append(issues, Issue{Code: "batch_candidate_wrong_archive_path", Message: record.SourceArchivePath})
	}
	if record.ArchiveMinimumRecords < 100 {
		issues = append(issues, Issue{Code: "batch_candidate_archive_minimum_too_low", Message: fmt.Sprintf("%d", record.ArchiveMinimumRecords)})
	}
	if index.CountByBatch != nil && index.CountByBatch[record.BatchID] < record.ArchiveMinimumRecords {
		issues = append(issues, Issue{Code: "batch_candidate_archive_batch_too_small", Message: fmt.Sprintf("%s=%d", record.BatchID, index.CountByBatch[record.BatchID])})
	}
	if len(record.SelectedUniqueIntentIDs) == 0 {
		issues = append(issues, Issue{Code: "batch_candidate_empty_selected_intents", Message: record.BatchID})
	}
	if len(record.SelectedUniqueIntentIDs) < 3 && record.ShardCount <= 1 {
		issues = append(issues, Issue{Code: "batch_candidate_too_few_selected_intents", Message: record.BatchID})
	}
	if len(record.SelectedUniqueIntentIDs) > MaxSelectedIntentIDsPerRecord {
		issues = append(issues, Issue{Code: "batch_candidate_gate_record_too_many_intents", Message: fmt.Sprintf("%s=%d max=%d", record.GateID, len(record.SelectedUniqueIntentIDs), MaxSelectedIntentIDsPerRecord)})
	}
	seenIntents := make(map[string]bool)
	for _, intentID := range record.SelectedUniqueIntentIDs {
		if !idPattern.MatchString(intentID) {
			issues = append(issues, Issue{Code: "batch_candidate_invalid_selected_intent", Message: intentID})
		}
		if seenIntents[intentID] {
			issues = append(issues, Issue{Code: "batch_candidate_duplicate_selected_intent", Message: intentID})
		}
		seenIntents[intentID] = true
		draft, ok := index.ByIntent[intentID]
		if !ok {
			issues = append(issues, Issue{Code: "batch_candidate_selected_intent_missing", Message: intentID})
			continue
		}
		if draft.BatchID != record.BatchID {
			issues = append(issues, Issue{Code: "batch_candidate_selected_intent_wrong_batch", Message: fmt.Sprintf("%s:%s", intentID, draft.BatchID)})
		}
		if draft.HumanScore < record.MinimumHumanScore {
			issues = append(issues, Issue{Code: "batch_candidate_selected_score_too_low", Message: fmt.Sprintf("%s=%d", intentID, draft.HumanScore)})
		}
		if record.CTAContextRequired && strings.TrimSpace(draft.CTAContext) == "" {
			issues = append(issues, Issue{Code: "batch_candidate_selected_without_cta_context", Message: intentID})
		}
		if draft.SourceMatrixID == "" {
			issues = append(issues, Issue{Code: "batch_candidate_selected_without_source_matrix", Message: intentID})
		}
		if draft.RenderAllowed || draft.SitemapAllowed || draft.PublicationAllowed || draft.PublicPath != "" {
			issues = append(issues, Issue{Code: "batch_candidate_selected_public_flag", Message: intentID})
		}
		candidatePath := record.CandidatePathPrefix + intentID + "/"
		if !router.IsCleanPublicPath(candidatePath) {
			issues = append(issues, Issue{Code: "batch_candidate_generated_path_not_clean", Message: candidatePath})
		}
	}
	if !router.IsCleanPublicPath(record.CandidatePathPrefix) {
		issues = append(issues, Issue{Code: "batch_candidate_prefix_not_clean", Message: record.CandidatePathPrefix})
	}
	if index.BaseURLMode != "" && record.BaseURLMode != index.BaseURLMode {
		issues = append(issues, Issue{Code: "batch_candidate_url_config_mismatch", Message: fmt.Sprintf("mode record=%s site=%s", record.BaseURLMode, index.BaseURLMode)})
	}
	if record.OfficialURLLocked != index.OfficialURLLocked {
		issues = append(issues, Issue{Code: "batch_candidate_url_config_mismatch", Message: fmt.Sprintf("locked record=%t site=%t", record.OfficialURLLocked, index.OfficialURLLocked)})
	}
	if record.MaxSimilarityAllowed > 0.64 || record.MaxSimilarityAllowed <= 0 {
		issues = append(issues, Issue{Code: "batch_candidate_similarity_limit_invalid", Message: fmt.Sprintf("%.4f", record.MaxSimilarityAllowed)})
	}
	if record.MaxSimilarityObserved > record.MaxSimilarityAllowed {
		issues = append(issues, Issue{Code: "batch_candidate_similarity_observed_too_high", Message: fmt.Sprintf("%.4f", record.MaxSimilarityObserved)})
	}
	if index.MaxSimilarity > record.MaxSimilarityAllowed+0.0001 {
		issues = append(issues, Issue{Code: "batch_candidate_archive_similarity_too_high", Message: fmt.Sprintf("%.4f", index.MaxSimilarity)})
	}
	if record.MinimumHumanScore < 88 {
		issues = append(issues, Issue{Code: "batch_candidate_minimum_score_too_low", Message: fmt.Sprintf("%d", record.MinimumHumanScore)})
	}
	if !record.CTAContextRequired {
		issues = append(issues, Issue{Code: "batch_candidate_without_cta_requirement", Message: record.BatchID})
	}
	if !record.SourceURLAuditRequired {
		issues = append(issues, Issue{Code: "batch_candidate_without_source_url_audit", Message: record.BatchID})
	}
	if strings.TrimSpace(record.PublicationBlockReason) == "" {
		issues = append(issues, Issue{Code: "batch_candidate_missing_block_reason", Message: record.BatchID})
	}
	if record.RenderAllowed {
		issues = append(issues, Issue{Code: "batch_candidate_render_allowed", Message: record.BatchID})
	}
	if record.SitemapAllowed {
		issues = append(issues, Issue{Code: "batch_candidate_sitemap_allowed", Message: record.BatchID})
	}
	if record.PublicationAllowed {
		issues = append(issues, Issue{Code: "batch_candidate_publication_allowed", Message: record.BatchID})
	}
	if record.PublicPath != "" {
		issues = append(issues, Issue{Code: "batch_candidate_has_public_path", Message: record.PublicPath})
	}
	if record.CheckedAt == "" {
		issues = append(issues, Issue{Code: "batch_candidate_without_checked_at", Message: record.BatchID})
	}
	return Report{Issues: issues}
}

func LoadRecords(root string) ([]Entry, Report) {
	projectRoot, err := findProjectRoot(root)
	if err != nil {
		return nil, Report{Issues: []Issue{{Code: "project_root_not_found", Message: err.Error()}}}
	}
	path := filepath.Join(projectRoot, "data", "editorial", "batch_candidate_gates.jsonl")
	file, err := os.Open(path)
	if err != nil {
		return nil, Report{Issues: []Issue{{Code: "batch_candidate_gates_missing", Message: err.Error()}}}
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
			issues = append(issues, Issue{Code: "batch_candidate_gate_invalid_json", Message: fmt.Sprintf("line=%d %s", lineNumber, err.Error())})
			continue
		}
		entries = append(entries, Entry{Line: lineNumber, Record: record})
	}
	if err := scanner.Err(); err != nil {
		issues = append(issues, Issue{Code: "batch_candidate_gate_scan_failed", Message: err.Error()})
	}
	return entries, Report{Issues: issues}
}

func WriteRecords(root string, records []Record) error {
	projectRoot, err := findProjectRoot(root)
	if err != nil {
		return err
	}
	path := filepath.Join(projectRoot, "data", "editorial", "batch_candidate_gates.jsonl")
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	index, indexReport := BuildArchiveIndex(root)
	if !indexReport.Passed() {
		return fmt.Errorf("batch_candidate_archive_index_failed=%s", strings.Join(indexReport.Messages(), " | "))
	}
	normalized := NormalizeShardRecords(records)
	if messages := (Report{Issues: validateEntriesAgainstArchive(entriesFromRecords(normalized), index)}).Messages(); len(messages) > 0 {
		return fmt.Errorf("invalid_batch_candidate_gate=%s", strings.Join(messages, " | "))
	}
	var builder strings.Builder
	for _, record := range normalized {
		data, err := json.Marshal(record)
		if err != nil {
			return err
		}
		builder.Write(data)
		builder.WriteByte('\n')
	}
	tempPath := path + ".tmp"
	if err := os.WriteFile(tempPath, []byte(builder.String()), 0644); err != nil {
		return err
	}
	if err := os.Rename(tempPath, path); err != nil {
		return err
	}
	return nil
}

func entriesFromRecords(records []Record) []Entry {
	entries := make([]Entry, 0, len(records))
	for index, record := range records {
		entries = append(entries, Entry{Line: index + 1, Record: record})
	}
	return entries
}

func NormalizeShardRecords(records []Record) []Record {
	groups := make(map[string][]Record)
	groupOrder := make([]string, 0, len(records))
	for _, record := range records {
		key := record.BatchID + "\x00" + baseGateID(record)
		if _, ok := groups[key]; !ok {
			groupOrder = append(groupOrder, key)
		}
		groups[key] = append(groups[key], record)
	}
	normalized := make([]Record, 0, len(records))
	for _, key := range groupOrder {
		groupRecords := orderedGroupRecords(groups[key])
		template := normalizeSingleRecord(groupRecords[0])
		selected := make([]string, 0)
		for _, record := range groupRecords {
			selected = append(selected, record.SelectedUniqueIntentIDs...)
		}
		template.SelectedUniqueIntentIDs = selected
		if len(selected) <= MaxSelectedIntentIDsPerRecord {
			normalized = append(normalized, template)
			continue
		}
		normalized = append(normalized, splitRecord(template)...)
	}
	sort.SliceStable(normalized, func(left int, right int) bool {
		if normalized[left].BatchID != normalized[right].BatchID {
			return normalized[left].BatchID < normalized[right].BatchID
		}
		if normalized[left].ShardIndex != normalized[right].ShardIndex {
			return normalized[left].ShardIndex < normalized[right].ShardIndex
		}
		return normalized[left].GateID < normalized[right].GateID
	})
	return normalized
}

func orderedGroupRecords(records []Record) []Record {
	ordered := append([]Record{}, records...)
	sort.SliceStable(ordered, func(left int, right int) bool {
		leftSharded := hasShardMetadata(ordered[left])
		rightSharded := hasShardMetadata(ordered[right])
		if leftSharded && rightSharded && ordered[left].ShardIndex != ordered[right].ShardIndex {
			return ordered[left].ShardIndex < ordered[right].ShardIndex
		}
		if leftSharded != rightSharded {
			return leftSharded
		}
		if !leftSharded {
			return false
		}
		return ordered[left].GateID < ordered[right].GateID
	})
	return ordered
}

func normalizeSingleRecord(record Record) Record {
	if record.GateGroupID != "" || record.ShardCount > 0 || record.ShardIndex > 0 || record.SelectedTotal > 0 {
		record.GateID = baseGateID(record)
		record.GateGroupID = ""
		record.ShardIndex = 0
		record.ShardCount = 0
		record.SelectedTotal = 0
	}
	return record
}

func splitRecord(record Record) []Record {
	baseGate := baseGateID(record)
	selected := append([]string{}, record.SelectedUniqueIntentIDs...)
	shardCount := (len(selected) + MaxSelectedIntentIDsPerRecord - 1) / MaxSelectedIntentIDsPerRecord
	shards := make([]Record, 0, shardCount)
	for index := 0; index < shardCount; index++ {
		start := index * MaxSelectedIntentIDsPerRecord
		end := start + MaxSelectedIntentIDsPerRecord
		if end > len(selected) {
			end = len(selected)
		}
		shard := record
		shard.GateGroupID = baseGate
		shard.ShardIndex = index + 1
		shard.ShardCount = shardCount
		shard.SelectedTotal = len(selected)
		shard.GateID = fmt.Sprintf("%s-shard-%03d", baseGate, index+1)
		shard.SelectedUniqueIntentIDs = append([]string{}, selected[start:end]...)
		shards = append(shards, shard)
	}
	return shards
}

func baseGateID(record Record) string {
	if record.GateGroupID != "" {
		return record.GateGroupID
	}
	if index := strings.LastIndex(record.GateID, "-shard-"); index > 0 {
		suffix := record.GateID[index+len("-shard-"):]
		if len(suffix) == 3 && isDigits(suffix) {
			return record.GateID[:index]
		}
	}
	return record.GateID
}

func isDigits(value string) bool {
	for _, r := range value {
		if r < '0' || r > '9' {
			return false
		}
	}
	return value != ""
}

func convertArchiveIssues(report batchdraftarchive.Report) []Issue {
	issues := make([]Issue, 0, len(report.Issues))
	for _, issue := range report.Issues {
		issues = append(issues, Issue{Code: "batch_candidate_archive_" + issue.Code, Message: issue.Message})
	}
	return issues
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

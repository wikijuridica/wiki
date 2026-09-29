package termpromotion

import (
	"fmt"
	"sort"
	"strings"

	"portaljuridico/internal/storage"
	"portaljuridico/internal/termintents"
	"portaljuridico/internal/terms"
)

type PromotedSeed struct {
	Candidate termintents.Candidate
	Seed      terms.Seed
	Score     int
}

type Plan struct {
	Seeds []PromotedSeed
}

type PromoteResult struct {
	Added   []terms.Seed
	Skipped []terms.Seed
}

type Issue struct {
	Code    string
	Message string
}

type Report struct {
	Issues []Issue
}

type ScoreCandidate struct {
	TermID            string
	PracticeArea      string
	RiskLevel         string
	WhatsAppCTAIntent string
	OnlineServiceMode string
	OfficialSourceURL string
}

func PlanTopCandidates(root string, limit int) (Plan, error) {
	if limit <= 0 {
		return Plan{}, fmt.Errorf("invalid_limit=%d", limit)
	}
	candidates, report := termintents.LoadCandidates(root)
	if !report.Passed() {
		return Plan{}, fmt.Errorf(strings.Join(report.Messages(), "; "))
	}

	ranked := make([]PromotedSeed, 0)
	for _, entry := range candidates {
		validation := termintents.ValidateCandidate(entry.Candidate)
		if !validation.Passed() {
			continue
		}
		ranked = append(ranked, PromotedSeed{
			Candidate: entry.Candidate,
			Seed:      BuildSeed(entry.Candidate),
			Score:     score(entry.Candidate),
		})
	}
	sort.SliceStable(ranked, func(i, j int) bool {
		if ranked[i].Score == ranked[j].Score {
			return ranked[i].Candidate.TermID < ranked[j].Candidate.TermID
		}
		return ranked[i].Score > ranked[j].Score
	})

	selected := selectWithAreaDiversity(ranked, limit)
	return Plan{Seeds: selected}, nil
}

func BuildSeed(candidate termintents.Candidate) terms.Seed {
	return terms.Seed{
		TermID:             candidate.TermID,
		Term:               candidate.Term,
		Language:           "pt-BR",
		QualityState:       "draft_only",
		SourceID:           candidate.OfficialSourceID,
		SourceURL:          candidate.OfficialSourceURL,
		CheckedAt:          candidate.CheckedAt,
		IntentHint:         "priorizar conteudo informativo com demanda humana e contratacao juridica digital",
		CandidateID:        candidate.TermID,
		DemandEvidenceType: candidate.DemandEvidenceType,
		DemandEvidenceURL:  candidate.DemandEvidenceURL,
		DemandQueryGroup:   candidate.DemandQueryGroup,
		OnlineServiceMode:  candidate.OnlineServiceMode,
		WhatsAppCTAIntent:  candidate.WhatsAppCTAIntent,
		Notes:              "Seed promovida de candidato de alta intencao; segue draft_only, sem URL publica e sem CTA ate revisao completa.",
	}
}

func PromoteTopCandidates(root string, limit int) (PromoteResult, error) {
	plan, err := PlanTopCandidates(root, limit)
	if err != nil {
		return PromoteResult{}, err
	}
	existing, report := terms.LoadSeeds(root)
	if !report.Passed() {
		return PromoteResult{}, fmt.Errorf(strings.Join(report.Messages(), "; "))
	}
	seen := make(map[string]bool)
	for _, entry := range existing {
		seen[entry.Seed.TermID] = true
	}

	result := PromoteResult{}
	for _, promoted := range plan.Seeds {
		if seen[promoted.Seed.TermID] {
			result.Skipped = append(result.Skipped, promoted.Seed)
			continue
		}
		if err := storage.AppendJSONL(root, "term_seeds", promoted.Seed); err != nil {
			return result, err
		}
		result.Added = append(result.Added, promoted.Seed)
		seen[promoted.Seed.TermID] = true
	}
	return result, nil
}

func ValidatePromotedSeeds(root string, minimum int) Report {
	seeds, report := terms.LoadSeeds(root)
	if !report.Passed() {
		return Report{Issues: fromTermIssues(report)}
	}
	issues := make([]Issue, 0)
	count := 0
	for _, entry := range seeds {
		if entry.Seed.CandidateID == "" {
			continue
		}
		count++
		seedReport := terms.ValidateSeed(entry.Seed)
		for _, issue := range seedReport.Issues {
			issues = append(issues, Issue{Code: issue.Code, Message: fmt.Sprintf("line=%d %s", entry.Line, issue.Message)})
		}
	}
	if count < minimum {
		issues = append(issues, Issue{Code: "insufficient_promoted_seeds", Message: fmt.Sprintf("got=%d want=%d", count, minimum)})
	}
	return Report{Issues: issues}
}

func selectWithAreaDiversity(ranked []PromotedSeed, limit int) []PromotedSeed {
	selected := make([]PromotedSeed, 0, limit)
	used := make(map[string]bool)
	for _, item := range ranked {
		if len(selected) >= limit {
			return selected
		}
		if used[item.Candidate.PracticeArea] {
			continue
		}
		selected = append(selected, item)
		used[item.Candidate.PracticeArea] = true
	}
	for _, item := range ranked {
		if len(selected) >= limit {
			return selected
		}
		if containsSeed(selected, item.Seed.TermID) {
			continue
		}
		selected = append(selected, item)
	}
	return selected
}

func containsSeed(items []PromotedSeed, termID string) bool {
	for _, item := range items {
		if item.Seed.TermID == termID {
			return true
		}
	}
	return false
}

func score(candidate termintents.Candidate) int {
	return scoreCandidate(ScoreCandidate{
		TermID:            candidate.TermID,
		PracticeArea:      candidate.PracticeArea,
		RiskLevel:         candidate.RiskLevel,
		WhatsAppCTAIntent: candidate.WhatsAppCTAIntent,
		OnlineServiceMode: candidate.OnlineServiceMode,
		OfficialSourceURL: candidate.OfficialSourceURL,
	})
}

func ScoreCandidateForTest(candidate ScoreCandidate) int {
	return scoreCandidate(candidate)
}

func scoreCandidate(candidate ScoreCandidate) int {
	score := areaPriority(candidate.PracticeArea)
	if strings.Contains(candidate.TermID, "online") {
		score += 10
	}
	if candidate.RiskLevel == "high" {
		score += 5
	}
	if candidate.WhatsAppCTAIntent == "high" {
		score += 5
	}
	if candidate.OnlineServiceMode == "digital_only" {
		score += 5
	} else {
		score -= 80
	}
	if isOfficialSourceURL(candidate.OfficialSourceURL) {
		score += 8
	} else {
		score -= 60
	}
	return score
}

func isOfficialSourceURL(value string) bool {
	prefixes := []string{
		"https://www.gov.br/",
		"https://www.cnj.jus.br/",
		"https://datajud-wiki.cnj.jus.br/",
		"https://dadosabertos.camara.leg.br/",
		"https://dadosabertos.web.stj.jus.br/",
		"https://legis.senado.leg.br/",
		"https://www.planalto.gov.br/",
	}
	for _, prefix := range prefixes {
		if strings.HasPrefix(value, prefix) {
			return true
		}
	}
	return false
}

func areaPriority(area string) int {
	switch area {
	case "saude_consumidor":
		return 100
	case "previdenciario":
		return 95
	case "previdenciario_consumidor":
		return 92
	case "trabalhista":
		return 90
	case "familia":
		return 86
	case "familia_sucessoes":
		return 84
	case "consumidor":
		return 75
	default:
		return 50
	}
}

func fromTermIssues(report terms.Report) []Issue {
	issues := make([]Issue, 0, len(report.Issues))
	for _, issue := range report.Issues {
		issues = append(issues, Issue{Code: issue.Code, Message: issue.Message})
	}
	return issues
}

func (r Report) Passed() bool {
	return len(r.Issues) == 0
}

func (r Report) Messages() []string {
	messages := make([]string, 0, len(r.Issues))
	for _, issue := range r.Issues {
		messages = append(messages, issue.Code+": "+issue.Message)
	}
	return messages
}

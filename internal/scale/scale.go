package scale

import (
	"encoding/json"
	"os"
	"path/filepath"
)

type Blueprint struct {
	PageType        string   `json:"page_type"`
	Families        []string `json:"families"`
	TopicsPerFamily int      `json:"topics_per_family"`
}

type Plan struct {
	TargetMinimumPages                int         `json:"target_minimum_pages"`
	RequiresArchitectureBeforeContent bool        `json:"requires_architecture_before_content"`
	RequiresSourceProvenance          bool        `json:"requires_source_provenance"`
	RequiresEditorialReview           bool        `json:"requires_editorial_review"`
	PublishedPagesDuringP0Value       int         `json:"published_pages_during_p0"`
	Blueprints                        []Blueprint `json:"blueprints"`
}

func LoadPlan(root string) (Plan, error) {
	projectRoot, err := findProjectRoot(root)
	if err != nil {
		return Plan{}, err
	}
	data, err := os.ReadFile(filepath.Join(projectRoot, "content", "scale_plan.json"))
	if err != nil {
		return Plan{}, err
	}
	var plan Plan
	if err := json.Unmarshal(data, &plan); err != nil {
		return Plan{}, err
	}
	return plan, nil
}

func (p Plan) TotalPlannedPages() int {
	total := 0
	for _, blueprint := range p.Blueprints {
		total += len(blueprint.Families) * blueprint.TopicsPerFamily
	}
	if total < p.TargetMinimumPages {
		return total
	}
	return total
}

func (p Plan) PublishedPagesDuringP0() int {
	return p.PublishedPagesDuringP0Value
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

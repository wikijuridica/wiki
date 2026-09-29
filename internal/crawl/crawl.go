package crawl

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

type BotRule struct {
	UserAgent string   `json:"user_agent"`
	Allow     []string `json:"allow"`
	Disallow  []string `json:"disallow"`
}

type BotPolicy struct {
	Rules       []BotRule `json:"rules"`
	SitemapPath string    `json:"sitemap_path"`
}

func LoadDefaultPolicy(root string) (BotPolicy, error) {
	projectRoot, err := findProjectRoot(root)
	if err != nil {
		return BotPolicy{}, err
	}
	data, err := os.ReadFile(filepath.Join(projectRoot, "content", "crawl_policy.json"))
	if err != nil {
		return BotPolicy{}, err
	}
	var policy BotPolicy
	if err := json.Unmarshal(data, &policy); err != nil {
		return BotPolicy{}, err
	}
	return policy, nil
}

func RenderRobotsTXT(policy BotPolicy, baseURL string) string {
	var out strings.Builder
	for i, rule := range policy.Rules {
		if i > 0 {
			out.WriteString("\n")
		}
		out.WriteString("User-agent: " + rule.UserAgent + "\n")
		for _, path := range rule.Allow {
			out.WriteString("Allow: " + path + "\n")
		}
		for _, path := range rule.Disallow {
			out.WriteString("Disallow: " + path + "\n")
		}
		out.WriteString("\n")
	}
	out.WriteString("Sitemap: " + strings.TrimRight(baseURL, "/") + policy.SitemapPath + "\n")
	return out.String()
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

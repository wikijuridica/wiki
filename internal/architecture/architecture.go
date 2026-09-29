package architecture

import (
	"os"
	"path/filepath"
	"strings"
	"unicode"
)

var requiredModules = map[string]string{
	"render":              "internal/render",
	"approvals":           "internal/approvals",
	"router":              "internal/router",
	"ondemand":            "internal/ondemand",
	"prepublication":      "internal/prepublication",
	"scale":               "internal/scale",
	"cta":                 "internal/cta",
	"content":             "internal/content",
	"batchdraftgen":       "internal/batchdraftgen",
	"batchdrafts":         "internal/batchdrafts",
	"batchsourcematrix":   "internal/batchsourcematrix",
	"authorialdrafts":     "internal/authorialdrafts",
	"contentbriefs":       "internal/contentbriefs",
	"legal":               "internal/legal",
	"legalreviews":        "internal/legalreviews",
	"humanscore":          "internal/humanscore",
	"manualresearch":      "internal/manualresearch",
	"sources":             "internal/sources",
	"sourceresolutions":   "internal/sourceresolutions",
	"storage":             "internal/storage",
	"provenance":          "internal/provenance",
	"termintents":         "internal/termintents",
	"termpromotion":       "internal/termpromotion",
	"publicationblockers": "internal/publicationblockers",
	"sourceblockers":      "internal/sourceblockers",
	"scalablebatches":     "internal/scalablebatches",
	"quality":             "internal/quality",
	"seo":                 "internal/seo",
	"crawl":               "internal/crawl",
	"editorial":           "internal/editorial",
	"editorialdrafts":     "internal/editorialdrafts",
	"reviewqueue":         "internal/reviewqueue",
	"tools":               "tools",
	"docs":                "docs",
	"build":               "internal/build",
	"sitemap":             "internal/sitemap",
	"architecture":        "internal/architecture",
}

var forbiddenFrontendTokens = []string{
	"next.js",
	"nextjs",
	"react",
	"vue",
	"svelte",
	"astro",
	"nuxt",
}

type Report struct {
	Passed        bool
	Modules       []string
	ForbiddenHits []string
	Messages      []string
}

func Validate(root string) Report {
	projectRoot, err := findProjectRoot(root)
	if err != nil {
		return Report{Passed: false, Messages: []string{err.Error()}}
	}

	modules := make([]string, 0)
	messages := make([]string, 0)
	for name, path := range requiredModules {
		if exists(filepath.Join(projectRoot, path)) {
			modules = append(modules, name)
		} else {
			messages = append(messages, "missing_module="+name)
		}
	}

	forbiddenHits := scanForbidden(projectRoot)
	for _, hit := range forbiddenHits {
		messages = append(messages, "forbidden_frontend_stack="+hit)
	}

	return Report{
		Passed:        len(messages) == 0,
		Modules:       modules,
		ForbiddenHits: forbiddenHits,
		Messages:      messages,
	}
}

func scanForbidden(root string) []string {
	scanRoots := []string{"internal", "cmd", "content", "tools"}
	hits := make([]string, 0)
	for _, scanRoot := range scanRoots {
		base := filepath.Join(root, scanRoot)
		if !exists(base) {
			continue
		}
		filepath.Walk(base, func(path string, info os.FileInfo, err error) error {
			if err != nil || info == nil || info.IsDir() {
				return nil
			}
			rel, _ := filepath.Rel(root, path)
			if rel == filepath.Join("internal", "architecture", "architecture.go") {
				return nil
			}
			data, err := os.ReadFile(path)
			if err != nil {
				return nil
			}
			text := strings.ToLower(string(data))
			for _, token := range forbiddenFrontendTokens {
				if containsForbiddenToken(text, token) {
					hits = append(hits, rel+":"+token)
				}
			}
			return nil
		})
	}
	return hits
}

func containsForbiddenToken(text string, token string) bool {
	start := 0
	for {
		index := strings.Index(text[start:], token)
		if index < 0 {
			return false
		}
		absolute := start + index
		beforeOK := absolute == 0 || !isWordRune(rune(text[absolute-1]))
		afterIndex := absolute + len(token)
		afterOK := afterIndex >= len(text) || !isWordRune(rune(text[afterIndex]))
		if beforeOK && afterOK {
			return true
		}
		start = afterIndex
	}
}

func isWordRune(value rune) bool {
	return unicode.IsLetter(value) || unicode.IsDigit(value) || value == '_' || value == '-'
}

func exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
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

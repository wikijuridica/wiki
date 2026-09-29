package checks

import (
	"strings"
	"testing"
)

func TestHTMLPerformanceIssuesRejectsHeavyBotHostileMarkup(t *testing.T) {
	heavyCSS := strings.Repeat(".x{display:block}", 600)
	html := []byte(`<!doctype html>
<html><head>
<script src="/assets/app.js"></script>
<link rel="modulepreload" href="/assets/app.js">
<style>` + heavyCSS + `</style>
</head><body>
<div id="__NEXT_DATA__" data-reactroot data-hydrate="root"></div>
</body></html>`)

	issues := htmlPerformanceIssues("heavy.html", html, int64(len(html)))

	for _, want := range []string{
		"script_not_allowed",
		"inline_style_over_budget",
		"frontend_runtime_marker_not_allowed=__next_data__",
		"frontend_runtime_marker_not_allowed=data-reactroot",
		"frontend_runtime_marker_not_allowed=data-hydrate",
		"frontend_runtime_marker_not_allowed=modulepreload",
		"runtime_asset_reference_not_allowed=.js",
	} {
		if !containsIssue(issues, want) {
			t.Fatalf("missing issue %q in %v", want, issues)
		}
	}
}

func TestCurrentGeneratedPagesStayLightForValuableBots(t *testing.T) {
	issues := Run("performance-budget", ".")
	if len(issues) != 0 {
		t.Fatalf("performance-budget failed: %v", issues)
	}
}

func containsIssue(issues []string, want string) bool {
	for _, issue := range issues {
		if strings.Contains(issue, want) {
			return true
		}
	}
	return false
}

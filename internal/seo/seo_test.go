package seo

import (
	"strings"
	"testing"

	"portaljuridico/internal/content"
)

func TestSearchAppearanceBudgetRejectsTitlesAndDescriptionsOutsideProjectLimits(t *testing.T) {
	page := indexablePageForSEO()
	page.Title = strings.Repeat("A", TitleMaxCharacters+1)
	page.MetaDescription = strings.Repeat("B", MetaDescriptionMaxCharacters+1)

	issues := ValidateSearchAppearance(page)

	for _, want := range []string{"title_too_long", "meta_description_too_long"} {
		if !hasAppearanceIssue(issues, want) {
			t.Fatalf("missing issue %q in %v", want, issues)
		}
	}
}

func TestSearchAppearanceBudgetCountsPTBRRunes(t *testing.T) {
	page := indexablePageForSEO()
	page.Title = strings.Repeat("é", TitleMaxCharacters)
	page.MetaDescription = strings.Repeat("ação ", 32)

	issues := ValidateSearchAppearance(page)

	if hasAppearanceIssue(issues, "title_too_long") {
		t.Fatalf("accented title at rune budget was treated as too long: %v", issues)
	}
	if hasAppearanceIssue(issues, "meta_description_too_long") {
		t.Fatalf("accented meta description at rune budget was treated as too long: %v", issues)
	}
}

func TestRenderHeadAppliesProjectSnippetBudgetForIndexablePages(t *testing.T) {
	indexable := indexablePageForSEO()
	noindex := indexable
	noindex.Status = "noindex"
	noindex.IndexPolicy = "noindex"

	indexableHead := RenderHead(indexable)
	noindexHead := RenderHead(noindex)

	if !strings.Contains(indexableHead, `name="robots" content="index,follow,max-snippet:160"`) {
		t.Fatalf("indexable head missing max-snippet budget: %s", indexableHead)
	}
	if strings.Contains(noindexHead, "max-snippet") {
		t.Fatalf("noindex head should not carry search snippet budget: %s", noindexHead)
	}
}

func hasAppearanceIssue(issues []AppearanceIssue, code string) bool {
	for _, issue := range issues {
		if issue.Code == code {
			return true
		}
	}
	return false
}

func indexablePageForSEO() content.Page {
	return content.Page{
		Path:            "/wiki/teste-seo/",
		PageType:        "wiki",
		Status:          "published",
		IndexPolicy:     "index",
		Title:           "Guia jurídico sobre prova documental",
		MetaDescription: "Entenda quando a prova documental pode ser usada, quais cuidados tomar e por que a revisão jurídica individual continua necessária.",
		CanonicalURL:    "https://portal-juridico.example/wiki/teste-seo/",
	}
}

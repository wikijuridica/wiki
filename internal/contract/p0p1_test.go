package contract_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"portaljuridico/internal/architecture"
	"portaljuridico/internal/build"
	"portaljuridico/internal/content"
	"portaljuridico/internal/crawl"
	"portaljuridico/internal/ondemand"
	"portaljuridico/internal/quality"
)

func TestStaticBuildOutputsCompleteHTMLAndIndexingAssets(t *testing.T) {
	repo, err := content.LoadRepository(".")
	if err != nil {
		t.Fatal(err)
	}

	out := t.TempDir()
	result, err := build.Site(repo, out)
	if err != nil {
		t.Fatal(err)
	}

	if result.IndexablePages < 1 {
		t.Fatalf("IndexablePages = %d, want at least 1", result.IndexablePages)
	}

	indexHTML := readFile(t, filepath.Join(out, "index.html"))
	requireContains(t, indexHTML, "<main")
	requireContains(t, indexHTML, `rel="canonical"`)
	requireContains(t, indexHTML, `name="robots" content="index,follow,max-snippet:160"`)
	requireContains(t, indexHTML, "<h1>Portal Jurídico Brasileiro</h1>")
	requireContains(t, indexHTML, `<a href="/fontes/planalto/">`)
	requireNotContains(t, strings.ToLower(indexHTML), "<script")

	requireFile(t, filepath.Join(out, "robots.txt"))
	requireFile(t, filepath.Join(out, "sitemap.xml"))
	requireFile(t, filepath.Join(out, "sitemaps", "pages-0001.xml"))
}

func TestIndexablePagesRequireUniqueIntentCanonicalAndQuality(t *testing.T) {
	source := content.SourceProvenance{
		SourceID:    "planalto",
		SourceName:  "Planalto",
		SourceURL:   "https://www.planalto.gov.br/",
		CheckedAt:   "2026-06-09",
		LicenseNote: "Fonte oficial governamental; termos pendentes de auditoria antes de ingestao.",
	}
	pageA := content.Page{
		Path:            "/wiki/direito-civil/responsabilidade-civil/",
		PageType:        "wiki",
		Status:          "published",
		IndexPolicy:     "index",
		UniqueIntentID:  "wiki:responsabilidade-civil",
		CanonicalURL:    "https://portal-juridico.example/wiki/direito-civil/responsabilidade-civil/",
		Title:           "Responsabilidade civil",
		MetaDescription: "Guia informativo sobre responsabilidade civil com fonte oficial documentada.",
		Heading:         "Responsabilidade civil",
		Summary:         "Conteudo juridico informativo com fonte, revisao e links internos uteis.",
		BodySections: []content.Section{
			{
				Title: "Finalidade informativa",
				Body:  "Este conteudo explica o tema em linguagem geral, separa informacao oficial de comentario editorial e nao substitui consulta juridica individual. A pagina usa fonte oficial documentada e permanece sujeita a revisao editorial.",
			},
			{
				Title: "Fonte e revisao",
				Body:  "A fonte oficial fica registrada em proveniencia propria. O texto editorial tem autor, revisor, data de publicacao e data de revisao.",
			},
		},
		InternalLinks:      []string{"/", "/fontes/planalto/"},
		PublicationDate:    "2026-06-09",
		ReviewedAt:         "2026-06-09",
		Author:             "Redacao tecnica",
		Reviewer:           "Revisao juridica",
		SourceProvenance:   []content.SourceProvenance{source},
		LegalNotice:        "Conteudo informativo; nao substitui consulta juridica individual.",
		PublicationPurpose: "Contrato de teste para gate P0/P1.",
	}
	pageB := pageA
	pageB.Path = "/wiki/direito-civil/responsabilidade-civil-duplicada/"
	pageB.CanonicalURL = "https://portal-juridico.example/wiki/direito-civil/responsabilidade-civil-duplicada/"

	report := quality.ValidatePages([]content.Page{pageA, pageB})

	if report.Passed() {
		t.Fatal("ValidatePages passed, want duplicate failures")
	}
	for _, code := range []string{"duplicate_intent", "duplicate_title", "duplicate_meta_description", "near_duplicate_content"} {
		if !report.HasIssue(code) {
			t.Fatalf("missing issue %q in %v", code, report.Codes())
		}
	}
}

func TestLegalContentWithoutSourceReviewOrNoticeCannotBeIndexable(t *testing.T) {
	unsafePage := content.Page{
		Path:            "/wiki/direito-civil/sem-fonte/",
		PageType:        "wiki",
		Status:          "published",
		IndexPolicy:     "index",
		UniqueIntentID:  "wiki:sem-fonte",
		CanonicalURL:    "https://portal-juridico.example/wiki/direito-civil/sem-fonte/",
		Title:           "Tema juridico sem fonte",
		MetaDescription: "Pagina juridica sem fonte oficial suficiente.",
		Heading:         "Tema juridico sem fonte",
		Summary:         "Resumo sem proveniencia oficial.",
		BodySections: []content.Section{
			{
				Title: "Resumo",
				Body:  "Este texto tenta publicar conteudo juridico sem fonte oficial, sem revisor e sem aviso informativo suficiente.",
			},
		},
		InternalLinks:      []string{"/"},
		PublicationDate:    "2026-06-09",
		Author:             "Redacao tecnica",
		PublicationPurpose: "Caso negativo do gate.",
	}

	report := quality.ValidatePages([]content.Page{unsafePage})

	if report.Passed() {
		t.Fatal("ValidatePages passed, want legal control failures")
	}
	for _, code := range []string{"legal_content_without_source", "legal_content_without_review", "legal_content_without_notice"} {
		if !report.HasIssue(code) {
			t.Fatalf("missing issue %q in %v", code, report.Codes())
		}
	}
}

func TestSearchAndParameterRoutesAreNoindexAndExcludedFromSitemap(t *testing.T) {
	repo, err := content.LoadRepository(".")
	if err != nil {
		t.Fatal(err)
	}

	out := t.TempDir()
	if _, err := build.Site(repo, out); err != nil {
		t.Fatal(err)
	}

	searchHTML := readFile(t, filepath.Join(out, "buscar", "index.html"))
	sitemap := readFile(t, filepath.Join(out, "sitemaps", "pages-0001.xml"))

	requireContains(t, searchHTML, `name="robots" content="noindex,follow"`)
	requireNotContains(t, sitemap, "/buscar/")
	for _, line := range strings.Split(sitemap, "\n") {
		if strings.Contains(line, "<loc>") {
			requireNotContains(t, line, "?")
		}
	}
}

func TestCrawlPolicySeparatesMajorBots(t *testing.T) {
	policy, err := crawl.LoadDefaultPolicy(".")
	if err != nil {
		t.Fatal(err)
	}

	robots := crawl.RenderRobotsTXT(policy, "https://portal-juridico.example")

	requireContains(t, robots, "User-agent: Googlebot")
	requireContains(t, robots, "User-agent: OAI-SearchBot")
	requireContains(t, robots, "User-agent: GPTBot")
	requireContains(t, robots, "Disallow: /buscar/")
	requireContains(t, robots, "Sitemap: https://portal-juridico.example/sitemap.xml")
}

func TestOwnOnDemandGeneratorRendersAndCachesWithoutNext(t *testing.T) {
	repo, err := content.LoadRepository(".")
	if err != nil {
		t.Fatal(err)
	}

	cacheDir := t.TempDir()
	generator := ondemand.New(repo, cacheDir)
	cacheFile := filepath.Join(cacheDir, "index.html")
	if _, err := os.Stat(cacheFile); err == nil {
		t.Fatal("cache file existed before first on-demand render")
	}

	first, err := generator.RenderPath("/")
	if err != nil {
		t.Fatal(err)
	}
	if first.FromCache {
		t.Fatal("first render came from cache, want generated on demand")
	}
	requireContains(t, first.HTML, `<h1>Portal Jurídico Brasileiro</h1>`)
	requireContains(t, first.HTML, `rel="canonical"`)
	requireFile(t, cacheFile)

	second, err := generator.RenderPath("/")
	if err != nil {
		t.Fatal(err)
	}
	if !second.FromCache {
		t.Fatal("second render was not served from the project-owned cache")
	}

	search, err := generator.RenderPath("/buscar/")
	if err != nil {
		t.Fatal(err)
	}
	requireContains(t, search.HTML, `name="robots" content="noindex,follow"`)
}

func TestArchitectureUsesRequiredModulesAndForbidsFrontendFrameworks(t *testing.T) {
	report := architecture.Validate(".")

	if !report.Passed {
		t.Fatalf("architecture failed: %v", report.Messages)
	}
	if len(report.ForbiddenHits) != 0 {
		t.Fatalf("forbidden frontend stack hits: %v", report.ForbiddenHits)
	}
	if !contains(report.Modules, "render") || !contains(report.Modules, "router") ||
		!contains(report.Modules, "quality") || !contains(report.Modules, "seo") ||
		!contains(report.Modules, "ondemand") {
		t.Fatalf("required modules missing in %v", report.Modules)
	}
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func requireFile(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Stat(path); err != nil {
		t.Fatal(err)
	}
}

func requireContains(t *testing.T, haystack, needle string) {
	t.Helper()
	if !strings.Contains(haystack, needle) {
		t.Fatalf("missing %q in:\n%s", needle, haystack)
	}
}

func requireNotContains(t *testing.T, haystack, needle string) {
	t.Helper()
	if strings.Contains(haystack, needle) {
		t.Fatalf("unexpected %q in:\n%s", needle, haystack)
	}
}

func contains(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

package checks

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"portaljuridico/internal/agentcontext"
	"portaljuridico/internal/approvals"
	"portaljuridico/internal/architecture"
	"portaljuridico/internal/authorialdrafts"
	"portaljuridico/internal/batchcandidateexpansion"
	"portaljuridico/internal/batchcandidategates"
	"portaljuridico/internal/batchcandidatereviews"
	"portaljuridico/internal/batchdraftarchive"
	"portaljuridico/internal/batchdraftgen"
	"portaljuridico/internal/batchdrafts"
	"portaljuridico/internal/batchexpansionstrategy"
	"portaljuridico/internal/batchfinaldrafts"
	"portaljuridico/internal/batchprepublication"
	"portaljuridico/internal/batchpublicmanifest"
	"portaljuridico/internal/batchsourceaudit"
	"portaljuridico/internal/batchsourcematrix"
	"portaljuridico/internal/batchsourcespecificity"
	"portaljuridico/internal/build"
	"portaljuridico/internal/content"
	"portaljuridico/internal/contentbriefs"
	"portaljuridico/internal/crawl"
	"portaljuridico/internal/editorial"
	"portaljuridico/internal/editorialdrafts"
	"portaljuridico/internal/humanscore"
	"portaljuridico/internal/legalreviews"
	"portaljuridico/internal/manualresearch"
	"portaljuridico/internal/paidintent"
	"portaljuridico/internal/paidintentrefinement"
	"portaljuridico/internal/prepublication"
	"portaljuridico/internal/publicationblockers"
	"portaljuridico/internal/quality"
	"portaljuridico/internal/reviewqueue"
	"portaljuridico/internal/scalablebatches"
	"portaljuridico/internal/seo"
	"portaljuridico/internal/sourceblockers"
	"portaljuridico/internal/sourceresolutions"
	"portaljuridico/internal/sources"
	"portaljuridico/internal/storage"
	"portaljuridico/internal/termintents"
	"portaljuridico/internal/termpromotion"
	"portaljuridico/internal/terms"
)

const (
	maxHTMLBytes        int64 = 50000
	maxInlineStyleBytes int64 = 8000
)

var Names = []string{
	"agent-context-ledger",
	"architecture",
	"content-quality",
	"seo",
	"google-search-appearance",
	"crawlability",
	"sources",
	"storage-contract",
	"manual-keyword-research",
	"term-seeds",
	"term-intent-candidates",
	"term-seed-promotions",
	"content-briefs",
	"authorial-content-drafts",
	"editorial-drafts",
	"review-queue",
	"approvals",
	"publication-blockers",
	"source-specificity-blockers",
	"source-specificity-resolutions",
	"prepublication-gates",
	"legal-editorial-reviews",
	"human-content-score",
	"scalable-content-batches",
	"batch-drafts",
	"batch-draft-expansion-archive",
	"batch-candidate-expansion-readiness",
	"batch-expansion-strategy",
	"batch-candidate-gates",
	"batch-candidate-reviews",
	"batch-prepublication-gates",
	"batch-source-specificity",
	"batch-public-manifest-gates",
	"batch-final-authorial-drafts",
	"paid-intent",
	"paid-intent-refinements",
	"batch-draft-generation",
	"batch-source-url-audits",
	"batch-source-matrix",
	"sitemaps",
	"canonicals",
	"no-duplicate-content",
	"mechanical-content",
	"performance-budget",
	"cpu-budget",
}

func Run(name string, root string) []string {
	switch name {
	case "agent-context-ledger":
		return checkAgentContextLedger(root)
	case "architecture":
		return checkArchitecture(root)
	case "content-quality":
		return checkContentQuality(root)
	case "seo":
		return checkSEO(root)
	case "google-search-appearance":
		return checkGoogleSearchAppearance(root)
	case "crawlability":
		return checkCrawlability(root)
	case "sources":
		return checkSources(root)
	case "storage-contract":
		return checkStorageContract(root)
	case "manual-keyword-research":
		return checkManualKeywordResearch(root)
	case "term-seeds":
		return checkTermSeeds(root)
	case "term-intent-candidates":
		return checkTermIntentCandidates(root)
	case "term-seed-promotions":
		return checkTermSeedPromotions(root)
	case "content-briefs":
		return checkContentBriefs(root)
	case "authorial-content-drafts":
		return checkAuthorialContentDrafts(root)
	case "editorial-drafts":
		return checkEditorialDrafts(root)
	case "review-queue":
		return checkReviewQueue(root)
	case "approvals":
		return checkApprovals(root)
	case "publication-blockers":
		return checkPublicationBlockers(root)
	case "source-specificity-blockers":
		return checkSourceSpecificityBlockers(root)
	case "source-specificity-resolutions":
		return checkSourceSpecificityResolutions(root)
	case "prepublication-gates":
		return checkPrepublicationGates(root)
	case "legal-editorial-reviews":
		return checkLegalEditorialReviews(root)
	case "human-content-score":
		return checkHumanContentScore(root)
	case "scalable-content-batches":
		return checkScalableContentBatches(root)
	case "batch-drafts":
		return checkBatchDrafts(root)
	case "batch-draft-expansion-archive":
		return checkBatchDraftExpansionArchive(root)
	case "batch-candidate-expansion-readiness":
		return checkBatchCandidateExpansionReadiness(root)
	case "batch-expansion-strategy":
		return checkBatchExpansionStrategy(root)
	case "batch-candidate-gates":
		return checkBatchCandidateGates(root)
	case "batch-candidate-reviews":
		return checkBatchCandidateReviews(root)
	case "batch-prepublication-gates":
		return checkBatchPrepublicationGates(root)
	case "batch-source-specificity":
		return checkBatchSourceSpecificity(root)
	case "batch-public-manifest-gates":
		return checkBatchPublicManifestGates(root)
	case "batch-final-authorial-drafts":
		return checkBatchFinalAuthorialDrafts(root)
	case "paid-intent":
		return checkPaidIntent(root)
	case "paid-intent-refinements":
		return checkPaidIntentRefinements(root)
	case "batch-draft-generation":
		return checkBatchDraftGeneration(root)
	case "batch-source-url-audits":
		return checkBatchSourceURLAudits(root)
	case "batch-source-matrix":
		return checkBatchSourceMatrix(root)
	case "sitemaps":
		return checkSitemaps(root)
	case "canonicals":
		return checkCanonicals(root)
	case "no-duplicate-content":
		return checkNoDuplicateContent(root)
	case "mechanical-content":
		return checkMechanicalContent(root)
	case "performance-budget":
		return checkPerformanceBudget(root)
	case "cpu-budget":
		return checkCPUBudget(root)
	default:
		return []string{"unknown_check=" + name}
	}
}

func checkAgentContextLedger(root string) []string {
	return agentcontext.Validate(root).Messages()
}

func checkArchitecture(root string) []string {
	report := architecture.Validate(root)
	errors := append([]string{}, report.Messages...)
	requiredDocs := []string{
		"docs/PROJECT_VISION.md",
		"docs/ARCHITECTURE.md",
		"docs/CONTENT_QUALITY.md",
		"docs/SEO_CRAWL_INDEXING.md",
		"docs/DATA_SOURCES.md",
		"docs/LAB_VALIDATION.md",
		"docs/ROADMAP_P0_P5.md",
		"docs/DECISIONS.md",
		"CHECKPOINT.md",
	}
	for _, doc := range requiredDocs {
		if _, err := os.Stat(filepath.Join(root, doc)); err != nil {
			errors = append(errors, "missing_required_doc="+doc)
		}
	}
	if _, err := os.Stat(filepath.Join(root, "docs", "adr")); err != nil {
		errors = append(errors, "missing_required_doc=docs/adr/")
	}
	if _, err := os.Stat(filepath.Join(root, "tools", "lab-cycle")); err != nil {
		errors = append(errors, "missing_required_tool=tools/lab-cycle")
	}
	if externalDependencies(root) {
		errors = append(errors, "external_dependency_present_without_adr")
	}
	if pythonFiles(root) {
		errors = append(errors, "python_file_present_after_go_decision")
	}
	return errors
}

func checkContentQuality(root string) []string {
	repo, err := content.LoadRepository(root)
	if err != nil {
		return []string{err.Error()}
	}
	registry, err := sources.LoadRegistry(root)
	if err != nil {
		return []string{err.Error()}
	}
	sourceReport := registry.ValidateForP0()
	if !sourceReport.Passed() {
		return sourceReport.Messages()
	}
	return quality.ValidatePagesWithSources(repo.Pages, registry).Messages()
}

func checkSEO(root string) []string {
	out, cleanup, err := buildTemp(root)
	if err != nil {
		return []string{err.Error()}
	}
	defer cleanup()

	errors := make([]string, 0)
	filepath.Walk(out, func(path string, info os.FileInfo, err error) error {
		if err != nil || info == nil || info.IsDir() || !strings.HasSuffix(path, ".html") {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			errors = append(errors, err.Error())
			return nil
		}
		text := string(data)
		for _, token := range []string{"<title>", `name="description"`, `rel="canonical"`, `name="robots"`, "<main"} {
			if !strings.Contains(text, token) {
				errors = append(errors, path+":missing_"+token)
			}
		}
		if strings.Contains(strings.ToLower(text), "<script") {
			errors = append(errors, path+":script_not_allowed")
		}
		return nil
	})
	return errors
}

func checkCrawlability(root string) []string {
	repo, err := content.LoadRepository(root)
	if err != nil {
		return []string{err.Error()}
	}
	policy, err := crawl.LoadDefaultPolicy(root)
	if err != nil {
		return []string{err.Error()}
	}
	robots := crawl.RenderRobotsTXT(policy, repo.BaseURL)
	errors := make([]string, 0)
	for _, bot := range []string{"Googlebot", "OAI-SearchBot", "GPTBot"} {
		if !strings.Contains(robots, "User-agent: "+bot) {
			errors = append(errors, "missing_bot_policy="+bot)
		}
	}
	if !strings.Contains(robots, "Disallow: /buscar/") {
		errors = append(errors, "missing_search_disallow")
	}
	if !strings.Contains(robots, "Sitemap: ") {
		errors = append(errors, "missing_sitemap_directive")
	}
	return errors
}

func checkGoogleSearchAppearance(root string) []string {
	repo, err := content.LoadRepository(root)
	if err != nil {
		return []string{err.Error()}
	}
	errors := make([]string, 0)
	for _, page := range repo.Pages {
		for _, issue := range seo.ValidateSearchAppearance(page) {
			errors = append(errors, page.Path+":"+issue.Code+":"+issue.Message)
		}
	}
	return errors
}

func checkSources(root string) []string {
	registry, err := sources.LoadRegistry(root)
	if err != nil {
		return []string{err.Error()}
	}
	return registry.ValidateForP0().Messages()
}

func checkStorageContract(root string) []string {
	return storage.Validate(root).Messages()
}

func checkManualKeywordResearch(root string) []string {
	return manualresearch.Validate(root).Messages()
}

func checkTermSeeds(root string) []string {
	return terms.ValidateSeeds(root).Messages()
}

func checkTermIntentCandidates(root string) []string {
	return termintents.Validate(root).Messages()
}

func checkTermSeedPromotions(root string) []string {
	return termpromotion.ValidatePromotedSeeds(root, 6).Messages()
}

func checkContentBriefs(root string) []string {
	return contentbriefs.Validate(root).Messages()
}

func checkAuthorialContentDrafts(root string) []string {
	return authorialdrafts.Validate(root).Messages()
}

func checkEditorialDrafts(root string) []string {
	return editorialdrafts.Validate(root).Messages()
}

func checkReviewQueue(root string) []string {
	return reviewqueue.Validate(root).Messages()
}

func checkApprovals(root string) []string {
	return approvals.Validate(root).Messages()
}

func checkPublicationBlockers(root string) []string {
	return publicationblockers.Validate(root).Messages()
}

func checkSourceSpecificityBlockers(root string) []string {
	return sourceblockers.Validate(root).Messages()
}

func checkSourceSpecificityResolutions(root string) []string {
	return sourceresolutions.Validate(root).Messages()
}

func checkPrepublicationGates(root string) []string {
	return prepublication.Validate(root).Messages()
}

func checkLegalEditorialReviews(root string) []string {
	return legalreviews.Validate(root).Messages()
}

func checkHumanContentScore(root string) []string {
	return humanscore.Validate(root).Messages()
}

func checkScalableContentBatches(root string) []string {
	return scalablebatches.Validate(root).Messages()
}

func checkBatchDrafts(root string) []string {
	return batchdrafts.Validate(root).Messages()
}

func checkBatchDraftExpansionArchive(root string) []string {
	return batchdraftarchive.Validate(root).Messages()
}

func checkBatchCandidateExpansionReadiness(root string) []string {
	return batchcandidateexpansion.Validate(root).Messages()
}

func checkBatchExpansionStrategy(root string) []string {
	return batchexpansionstrategy.Validate(root).Messages()
}

func checkBatchCandidateGates(root string) []string {
	return batchcandidategates.Validate(root).Messages()
}

func checkBatchCandidateReviews(root string) []string {
	return batchcandidatereviews.Validate(root).Messages()
}

func checkBatchPrepublicationGates(root string) []string {
	return batchprepublication.Validate(root).Messages()
}

func checkBatchSourceSpecificity(root string) []string {
	return batchsourcespecificity.Validate(root).Messages()
}

func checkBatchPublicManifestGates(root string) []string {
	return batchpublicmanifest.Validate(root).Messages()
}

func checkBatchFinalAuthorialDrafts(root string) []string {
	return batchfinaldrafts.Validate(root).Messages()
}

func checkPaidIntent(root string) []string {
	return paidintent.Validate(root).Messages()
}

func checkPaidIntentRefinements(root string) []string {
	return paidintentrefinement.Validate(root).Messages()
}

func checkBatchDraftGeneration(root string) []string {
	errors := make([]string, 0)
	_, generated := batchdraftgen.Generate(root, batchdraftgen.DefaultOptions())
	errors = append(errors, generated.Messages()...)
	errors = append(errors, batchdraftgen.ValidateStoredMetrics(root).Messages()...)
	return errors
}

func checkBatchSourceURLAudits(root string) []string {
	return batchsourceaudit.Validate(root).Messages()
}

func checkBatchSourceMatrix(root string) []string {
	return batchsourcematrix.Validate(root).Messages()
}

func checkSitemaps(root string) []string {
	out, cleanup, err := buildTemp(root)
	if err != nil {
		return []string{err.Error()}
	}
	defer cleanup()

	repo, err := content.LoadRepository(root)
	if err != nil {
		return []string{err.Error()}
	}
	indexData, err := os.ReadFile(filepath.Join(out, "sitemap.xml"))
	if err != nil {
		return []string{err.Error()}
	}
	pageData, err := os.ReadFile(filepath.Join(out, "sitemaps", "pages-0001.xml"))
	if err != nil {
		return []string{err.Error()}
	}
	index := string(indexData)
	pages := string(pageData)
	errors := make([]string, 0)
	if !strings.Contains(index, "<sitemapindex") {
		errors = append(errors, "sitemap_xml_not_index")
	}
	if !strings.Contains(index, "pages-0001.xml") {
		errors = append(errors, "sitemap_index_missing_partition")
	}
	for _, page := range repo.Pages {
		inSitemap := strings.Contains(pages, page.CanonicalURL)
		if editorial.IsIndexable(page) && !inSitemap {
			errors = append(errors, page.Path+":indexable_missing_from_sitemap")
		}
		if !editorial.IsIndexable(page) && inSitemap {
			errors = append(errors, page.Path+":noindex_in_sitemap")
		}
	}
	return errors
}

func checkCanonicals(root string) []string {
	repo, err := content.LoadRepository(root)
	if err != nil {
		return []string{err.Error()}
	}
	report := quality.ValidatePages(repo.Pages)
	errors := make([]string, 0)
	for _, message := range report.Messages() {
		if strings.Contains(message, "canonical") || strings.Contains(message, "unclean_url") {
			errors = append(errors, message)
		}
	}
	return errors
}

func checkNoDuplicateContent(root string) []string {
	repo, err := content.LoadRepository(root)
	if err != nil {
		return []string{err.Error()}
	}
	report := quality.ValidatePages(repo.Pages)
	errors := make([]string, 0)
	duplicateCodes := []string{
		"duplicate_intent",
		"duplicate_title",
		"duplicate_meta_description",
		"duplicate_canonical",
		"duplicate_content_hash",
		"near_duplicate_content",
	}
	for _, message := range report.Messages() {
		for _, code := range duplicateCodes {
			if strings.Contains(message, code) {
				errors = append(errors, message)
			}
		}
	}
	return errors
}

func checkMechanicalContent(root string) []string {
	repo, err := content.LoadRepository(root)
	if err != nil {
		return []string{err.Error()}
	}
	errors := make([]string, 0)
	mechanicalCodes := map[string]bool{
		"thin_content":                   true,
		"keyword_stuffing":               true,
		"low_lexical_diversity":          true,
		"repeated_phrase":                true,
		"repeated_sentence":              true,
		"mechanical_keyword_permutation": true,
	}
	for _, page := range repo.Pages {
		if !editorial.IsIndexable(page) {
			continue
		}
		analysis := quality.AnalyzePage(page)
		for _, issue := range analysis.Issues {
			if mechanicalCodes[issue.Code] {
				errors = append(errors, page.Path+":"+issue.Code+":"+issue.Message)
			}
		}
	}
	return errors
}

func checkPerformanceBudget(root string) []string {
	out, cleanup, err := buildTemp(root)
	if err != nil {
		return []string{err.Error()}
	}
	defer cleanup()

	errors := make([]string, 0)
	filepath.Walk(out, func(path string, info os.FileInfo, err error) error {
		if err != nil || info == nil || info.IsDir() {
			return nil
		}
		if generatedHeavyAsset(path) {
			errors = append(errors, path+":heavy_public_asset_not_allowed")
			return nil
		}
		if !strings.HasSuffix(path, ".html") {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			errors = append(errors, err.Error())
			return nil
		}
		errors = append(errors, htmlPerformanceIssues(path, data, info.Size())...)
		return nil
	})
	return errors
}

func checkCPUBudget(root string) []string {
	scanRoots := []string{
		"cmd/server",
		"internal/render",
		"internal/ondemand",
		"internal/router",
		"internal/seo",
		"internal/editorial",
	}
	errors := make([]string, 0)
	for _, scanRoot := range scanRoots {
		base := filepath.Join(root, scanRoot)
		if _, err := os.Stat(base); err != nil {
			continue
		}
		filepath.Walk(base, func(path string, info os.FileInfo, err error) error {
			if err != nil || info == nil || info.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
				return nil
			}
			data, err := os.ReadFile(path)
			if err != nil {
				errors = append(errors, err.Error())
				return nil
			}
			errors = append(errors, cpuBudgetIssuesForSource(path, string(data))...)
			return nil
		})
	}
	return errors
}

func cpuBudgetIssuesForSource(path string, source string) []string {
	text := strings.ToLower(source)
	errors := make([]string, 0)
	for _, marker := range []string{"exec.command", "\"os/exec\"", "'os/exec'"} {
		if strings.Contains(text, marker) {
			errors = append(errors, path+":public_runtime_exec_not_allowed")
			break
		}
	}
	for _, marker := range []string{"http.get(", "http.post(", "http.defaultclient", "net.dial"} {
		if strings.Contains(text, marker) {
			errors = append(errors, path+":public_runtime_network_call_not_allowed")
			break
		}
	}
	if strings.Contains(text, "time.sleep(") {
		errors = append(errors, path+":public_runtime_sleep_not_allowed")
	}
	if strings.Contains(text, "for {") || strings.Contains(text, "for{") {
		errors = append(errors, path+":unbounded_loop_not_allowed")
	}
	return errors
}

func htmlPerformanceIssues(path string, data []byte, size int64) []string {
	text := strings.ToLower(string(data))
	errors := make([]string, 0)
	if size > maxHTMLBytes {
		errors = append(errors, fmt.Sprintf("%s:html_size_over_budget=%d", path, size))
	}
	if strings.Contains(text, "<script") {
		errors = append(errors, path+":script_not_allowed")
	}
	if inlineStyleBytes(text) > maxInlineStyleBytes {
		errors = append(errors, fmt.Sprintf("%s:inline_style_over_budget=%d", path, inlineStyleBytes(text)))
	}
	for marker, code := range frontendRuntimeMarkers() {
		if strings.Contains(text, marker) {
			errors = append(errors, path+":frontend_runtime_marker_not_allowed="+code)
		}
	}
	for _, reference := range []string{".js", ".mjs", ".wasm"} {
		if containsRuntimeAssetReference(text, reference) {
			errors = append(errors, path+":runtime_asset_reference_not_allowed="+reference)
		}
	}
	return errors
}

func inlineStyleBytes(text string) int64 {
	var total int64
	cursor := 0
	for {
		start := strings.Index(text[cursor:], "<style")
		if start == -1 {
			return total
		}
		start += cursor
		openEnd := strings.Index(text[start:], ">")
		if openEnd == -1 {
			total += int64(len(text) - start)
			return total
		}
		contentStart := start + openEnd + 1
		closeStart := strings.Index(text[contentStart:], "</style>")
		if closeStart == -1 {
			total += int64(len(text) - contentStart)
			return total
		}
		total += int64(closeStart)
		cursor = contentStart + closeStart + len("</style>")
	}
}

func frontendRuntimeMarkers() map[string]string {
	return map[string]string{
		"__next_data__":  "__next_data__",
		"astro-island":   "astro-island",
		"client:load":    "client:load",
		"client:visible": "client:visible",
		"data-hydrate":   "data-hydrate",
		"data-reactroot": "data-reactroot",
		"data-svelte-h":  "data-svelte-h",
		"importmap":      "importmap",
		"modulepreload":  "modulepreload",
		"ng-version":     "ng-version",
		"vite":           "vite",
		"webpack":        "webpack",
	}
}

func containsRuntimeAssetReference(text string, reference string) bool {
	for _, suffix := range []string{`"` + reference, `'` + reference, reference + `"`, reference + `'`, reference + "?", reference + "#"} {
		if strings.Contains(text, suffix) {
			return true
		}
	}
	return false
}

func generatedHeavyAsset(path string) bool {
	for _, suffix := range []string{".js", ".mjs", ".wasm", ".map"} {
		if strings.HasSuffix(path, suffix) {
			return true
		}
	}
	return false
}

func buildTemp(root string) (string, func(), error) {
	out, err := os.MkdirTemp("", "portaljuridico-check-")
	if err != nil {
		return "", func() {}, err
	}
	repo, err := content.LoadRepository(root)
	if err != nil {
		os.RemoveAll(out)
		return "", func() {}, err
	}
	if _, err := build.Site(repo, out); err != nil {
		os.RemoveAll(out)
		return "", func() {}, err
	}
	return out, func() { os.RemoveAll(out) }, nil
}

func externalDependencies(root string) bool {
	data, err := os.ReadFile(filepath.Join(root, "go.mod"))
	if err != nil {
		return false
	}
	for _, line := range strings.Split(string(data), "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "require ") || trimmed == "require (" {
			return true
		}
	}
	return false
}

func pythonFiles(root string) bool {
	found := false
	filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil || info == nil || info.IsDir() {
			return nil
		}
		if strings.Contains(path, string(filepath.Separator)+".git"+string(filepath.Separator)) {
			return nil
		}
		if strings.HasSuffix(path, ".py") || strings.HasSuffix(path, ".pyc") {
			found = true
		}
		return nil
	})
	return found
}

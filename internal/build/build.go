package build

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"portaljuridico/internal/content"
	"portaljuridico/internal/crawl"
	"portaljuridico/internal/cta"
	"portaljuridico/internal/editorial"
	"portaljuridico/internal/quality"
	"portaljuridico/internal/render"
	"portaljuridico/internal/router"
	"portaljuridico/internal/sitemap"
	"portaljuridico/internal/sources"
)

type Result struct {
	GeneratedPages int
	IndexablePages int
	OutputDir      string
}

func Site(repo content.Repository, outputDir string) (Result, error) {
	report := quality.ValidatePages(repo.Pages)
	registry, registryErr := sources.LoadRegistry(repo.Root)
	hasRegistry := registryErr == nil
	if hasRegistry {
		report = quality.ValidatePagesWithSources(repo.Pages, registry)
	}
	if !report.Passed() {
		return Result{}, errors.New(strings.Join(report.Messages(), "\n"))
	}

	if err := os.RemoveAll(outputDir); err != nil {
		return Result{}, err
	}
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return Result{}, err
	}

	ctaPolicy, ctaPolicyErr := cta.LoadPolicy(repo.Root)
	hasCTAPolicy := ctaPolicyErr == nil
	for _, page := range repo.Pages {
		target := router.OutputFileForPath(page.Path, outputDir)
		if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
			return Result{}, err
		}
		html := render.Page(page)
		if hasRegistry && hasCTAPolicy {
			html = render.PageWithCTA(page, ctaPolicy, registry)
		}
		if err := os.WriteFile(target, []byte(html), 0644); err != nil {
			return Result{}, err
		}
	}

	indexable := make([]content.Page, 0)
	for _, page := range repo.Pages {
		if editorial.IsIndexable(page) {
			indexable = append(indexable, page)
		}
	}

	sitemapDir := filepath.Join(outputDir, "sitemaps")
	if err := os.MkdirAll(sitemapDir, 0755); err != nil {
		return Result{}, err
	}
	pagesSitemap, err := sitemap.RenderURLSet(indexable)
	if err != nil {
		return Result{}, err
	}
	if err := os.WriteFile(filepath.Join(sitemapDir, "pages-0001.xml"), pagesSitemap, 0644); err != nil {
		return Result{}, err
	}
	sitemapIndex, err := sitemap.RenderIndex(repo.BaseURL, []string{"/sitemaps/pages-0001.xml"})
	if err != nil {
		return Result{}, err
	}
	if err := os.WriteFile(filepath.Join(outputDir, "sitemap.xml"), sitemapIndex, 0644); err != nil {
		return Result{}, err
	}

	crawlPolicy, err := crawl.LoadDefaultPolicy(repo.Root)
	if err != nil {
		return Result{}, err
	}
	robots := crawl.RenderRobotsTXT(crawlPolicy, repo.BaseURL)
	if err := os.WriteFile(filepath.Join(outputDir, "robots.txt"), []byte(robots), 0644); err != nil {
		return Result{}, err
	}

	return Result{GeneratedPages: len(repo.Pages), IndexablePages: len(indexable), OutputDir: outputDir}, nil
}

func FormatResult(result Result) string {
	return fmt.Sprintf("generated_pages=%d indexable_pages=%d output_dir=%s", result.GeneratedPages, result.IndexablePages, result.OutputDir)
}

package ondemand

import (
	"errors"
	"os"
	"path/filepath"

	"portaljuridico/internal/content"
	"portaljuridico/internal/render"
	"portaljuridico/internal/router"
)

type GeneratedPage struct {
	Page      content.Page
	HTML      string
	CachePath string
	FromCache bool
}

type Generator struct {
	repository content.Repository
	cacheDir   string
	pagesByURL map[string]content.Page
}

func New(repository content.Repository, cacheDir string) Generator {
	pagesByURL := make(map[string]content.Page)
	for _, page := range repository.Pages {
		pagesByURL[page.Path] = page
	}
	return Generator{
		repository: repository,
		cacheDir:   cacheDir,
		pagesByURL: pagesByURL,
	}
}

func (g Generator) RenderPath(path string) (GeneratedPage, error) {
	page, ok := g.pagesByURL[path]
	if !ok {
		return GeneratedPage{}, errors.New("route_not_found")
	}
	if !router.IsCleanPublicPath(path) {
		return GeneratedPage{}, errors.New("unclean_route")
	}

	cachePath := router.OutputFileForPath(path, g.cacheDir)
	if data, err := os.ReadFile(cachePath); err == nil {
		return GeneratedPage{
			Page:      page,
			HTML:      string(data),
			CachePath: cachePath,
			FromCache: true,
		}, nil
	}

	html := render.Page(page)
	if err := os.MkdirAll(filepath.Dir(cachePath), 0755); err != nil {
		return GeneratedPage{}, err
	}
	if err := os.WriteFile(cachePath, []byte(html), 0644); err != nil {
		return GeneratedPage{}, err
	}
	return GeneratedPage{
		Page:      page,
		HTML:      html,
		CachePath: cachePath,
		FromCache: false,
	}, nil
}

package sitemap

import (
	"encoding/xml"
	"time"

	"portaljuridico/internal/content"
)

type sitemapIndex struct {
	XMLName  xml.Name       `xml:"sitemapindex"`
	XMLNS    string         `xml:"xmlns,attr"`
	Sitemaps []sitemapEntry `xml:"sitemap"`
}

type sitemapEntry struct {
	Loc     string `xml:"loc"`
	LastMod string `xml:"lastmod"`
}

type urlSet struct {
	XMLName xml.Name   `xml:"urlset"`
	XMLNS   string     `xml:"xmlns,attr"`
	URLs    []urlEntry `xml:"url"`
}

type urlEntry struct {
	Loc     string `xml:"loc"`
	LastMod string `xml:"lastmod"`
}

func RenderIndex(baseURL string, paths []string) ([]byte, error) {
	entries := make([]sitemapEntry, 0, len(paths))
	now := time.Now().UTC().Format("2006-01-02")
	for _, path := range paths {
		entries = append(entries, sitemapEntry{Loc: baseURL + path, LastMod: now})
	}
	index := sitemapIndex{
		XMLNS:    "http://www.sitemaps.org/schemas/sitemap/0.9",
		Sitemaps: entries,
	}
	return marshalXML(index)
}

func RenderURLSet(pages []content.Page) ([]byte, error) {
	urls := make([]urlEntry, 0, len(pages))
	for _, page := range pages {
		lastMod := page.ReviewedAt
		if lastMod == "" {
			lastMod = page.PublicationDate
		}
		urls = append(urls, urlEntry{Loc: page.CanonicalURL, LastMod: lastMod})
	}
	set := urlSet{
		XMLNS: "http://www.sitemaps.org/schemas/sitemap/0.9",
		URLs:  urls,
	}
	return marshalXML(set)
}

func marshalXML(value interface{}) ([]byte, error) {
	data, err := xml.MarshalIndent(value, "", "  ")
	if err != nil {
		return nil, err
	}
	output := append([]byte(xml.Header), data...)
	output = append(output, '\n')
	return output, nil
}

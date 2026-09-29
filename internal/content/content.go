package content

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

var LegalPageTypes = map[string]bool{
	"wiki":              true,
	"legislacao":        true,
	"dispositivo-legal": true,
	"jurisprudencia":    true,
	"precedente":        true,
	"noticia-juridica":  true,
	"artigo":            true,
	"pergunta":          true,
}

type SourceProvenance struct {
	SourceID    string `json:"source_id"`
	SourceName  string `json:"source_name"`
	SourceURL   string `json:"source_url"`
	CheckedAt   string `json:"checked_at"`
	LicenseNote string `json:"license_note"`
}

type Section struct {
	Title string `json:"title"`
	Body  string `json:"body"`
}

type Page struct {
	Path               string             `json:"path"`
	PageType           string             `json:"page_type"`
	Status             string             `json:"status"`
	IndexPolicy        string             `json:"index_policy"`
	UniqueIntentID     string             `json:"unique_intent_id"`
	CanonicalURL       string             `json:"canonical_url"`
	Title              string             `json:"title"`
	MetaDescription    string             `json:"meta_description"`
	Heading            string             `json:"heading"`
	Summary            string             `json:"summary"`
	BodySections       []Section          `json:"body_sections"`
	InternalLinks      []string           `json:"internal_links"`
	PublicationDate    string             `json:"publication_date"`
	ReviewedAt         string             `json:"reviewed_at"`
	Author             string             `json:"author"`
	Reviewer           string             `json:"reviewer"`
	SourceProvenance   []SourceProvenance `json:"source_provenance"`
	LegalNotice        string             `json:"legal_notice"`
	PublicationPurpose string             `json:"publication_purpose"`
}

func (p Page) IsLegalContent() bool {
	return LegalPageTypes[p.PageType]
}

func (p Page) PlainText() string {
	parts := []string{
		p.Title,
		p.MetaDescription,
		p.Heading,
		p.Summary,
		p.LegalNotice,
		p.PublicationPurpose,
	}
	for _, section := range p.BodySections {
		parts = append(parts, section.Title, section.Body)
	}
	return strings.Join(parts, " ")
}

type Repository struct {
	BaseURL           string
	BaseURLMode       string
	OfficialURLStatus string
	OfficialURLLocked bool
	EditorialIdentity EditorialIdentity
	Pages             []Page
	Root              string
}

type EditorialIdentity struct {
	AuthorName string `json:"author_name"`
	OABSection string `json:"oab_section"`
	OABNumber  string `json:"oab_number"`
	Source     string `json:"source"`
}

type siteConfig struct {
	BaseURL           string            `json:"base_url"`
	BaseURLMode       string            `json:"base_url_mode"`
	OfficialURLStatus string            `json:"official_url_status"`
	OfficialURLLocked bool              `json:"official_url_locked"`
	EditorialIdentity EditorialIdentity `json:"editorial_identity"`
}

func LoadRepository(root string) (Repository, error) {
	projectRoot, err := findProjectRoot(root)
	if err != nil {
		return Repository{}, err
	}

	var site siteConfig
	if err := readJSON(filepath.Join(projectRoot, "content", "site.json"), &site); err != nil {
		return Repository{}, err
	}

	var pages []Page
	if err := readJSON(filepath.Join(projectRoot, "content", "pages.json"), &pages); err != nil {
		return Repository{}, err
	}

	return Repository{
		BaseURL:           strings.TrimRight(site.BaseURL, "/"),
		BaseURLMode:       site.BaseURLMode,
		OfficialURLStatus: site.OfficialURLStatus,
		OfficialURLLocked: site.OfficialURLLocked,
		EditorialIdentity: site.EditorialIdentity,
		Pages:             pages,
		Root:              projectRoot,
	}, nil
}

func readJSON(path string, target interface{}) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, target)
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

package contract_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"portaljuridico/internal/content"
	"portaljuridico/internal/seo"
)

func TestProjectBaseURLUsesOfficialWikiJuridicaDomain(t *testing.T) {
	repo, err := content.LoadRepository(".")
	if err != nil {
		t.Fatalf("could not load repository: %v", err)
	}
	if !seo.IsAbsoluteHTTPSURL(repo.BaseURL) {
		t.Fatalf("base_url must be absolute HTTPS, got %q", repo.BaseURL)
	}
	if repo.BaseURL != "https://wikijuridica.com.br" {
		t.Fatalf("base_url=%q, want official project domain https://wikijuridica.com.br", repo.BaseURL)
	}
	if repo.BaseURLMode != "official_configured" {
		t.Fatalf("base_url_mode=%q, want official_configured", repo.BaseURLMode)
	}
	if repo.OfficialURLStatus != "locked" {
		t.Fatalf("official_url_status=%q, want locked", repo.OfficialURLStatus)
	}
	if !repo.OfficialURLLocked {
		t.Fatal("official project URL must be locked after wikijuridica.com.br was defined")
	}
	if strings.Contains(repo.BaseURL, ".example") && repo.BaseURLMode != "lab_placeholder" {
		t.Fatalf("example domain can only be used as lab placeholder, got mode=%q", repo.BaseURLMode)
	}
}

func TestEditorialIdentityComesFromSiteConfigNotRuntimeHardcode(t *testing.T) {
	repo, err := content.LoadRepository(".")
	if err != nil {
		t.Fatalf("could not load repository: %v", err)
	}
	if repo.EditorialIdentity.AuthorName != "Rafael Toledo" {
		t.Fatalf("author name=%q, want configured Rafael Toledo", repo.EditorialIdentity.AuthorName)
	}
	if repo.EditorialIdentity.OABSection != "RJ" || repo.EditorialIdentity.OABNumber != "227191" {
		t.Fatalf("OAB identity=%s/%s, want RJ/227191", repo.EditorialIdentity.OABSection, repo.EditorialIdentity.OABNumber)
	}
	if repo.EditorialIdentity.Source != "project_owner_declared_config" {
		t.Fatalf("identity source=%q, want project_owner_declared_config", repo.EditorialIdentity.Source)
	}

	root := projectRoot(t)
	for _, dir := range []string{"internal", "cmd", "tools"} {
		err := filepath.WalkDir(filepath.Join(root, dir), func(path string, entry os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if entry.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
				return nil
			}
			data, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			text := string(data)
			for _, forbidden := range []string{"Rafael Toledo", "227191", "OAB/RJ"} {
				if strings.Contains(text, forbidden) {
					t.Fatalf("runtime code hardcodes editorial identity %q in %s; use content/site.json", forbidden, path)
				}
			}
			return nil
		})
		if err != nil {
			t.Fatalf("runtime hardcode scan failed: %v", err)
		}
	}
}

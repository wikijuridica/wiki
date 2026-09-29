package contract_test

import (
	"strings"
	"testing"

	"portaljuridico/internal/content"
	"portaljuridico/internal/cta"
	"portaljuridico/internal/render"
	"portaljuridico/internal/sources"
)

func TestCurrentPublicPagesDoNotRenderWhatsAppCTA(t *testing.T) {
	repo, err := content.LoadRepository(".")
	if err != nil {
		t.Fatal(err)
	}
	policy, err := cta.LoadPolicy(".")
	if err != nil {
		t.Fatal(err)
	}
	registry, err := sources.LoadRegistry(".")
	if err != nil {
		t.Fatal(err)
	}

	for _, page := range repo.Pages {
		html := render.PageWithCTA(page, policy, registry)
		if strings.Contains(strings.ToLower(html), "whatsapp") {
			t.Fatalf("current page %s rendered WhatsApp CTA before approval", page.Path)
		}
	}
}

func TestApprovedFixtureRendersWhatsAppCTA(t *testing.T) {
	policy, err := cta.LoadPolicy(".")
	if err != nil {
		t.Fatal(err)
	}
	registry, err := sources.LoadRegistry(".")
	if err != nil {
		t.Fatal(err)
	}
	for index, source := range registry.Sources {
		if source.SourceID == "planalto" {
			registry.Sources[index].AuditStatus = "approved"
			registry.Sources[index].IngestionEnabled = false
		}
	}

	html := render.PageWithCTA(approvedLegalPage(), policy, registry)
	if !strings.Contains(html, "Contratar advogado pelo WhatsApp") {
		t.Fatal("approved fixture did not render WhatsApp CTA")
	}
	if strings.Contains(html, "CONFIGURAR_WHATSAPP_PROPRIO") {
		t.Fatal("CTA leaked raw WhatsApp placeholder")
	}
}

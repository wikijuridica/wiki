package render

import (
	"html"
	"net/url"
	"strings"

	"portaljuridico/internal/content"
	"portaljuridico/internal/cta"
	"portaljuridico/internal/editorial"
	"portaljuridico/internal/seo"
	"portaljuridico/internal/sources"
)

func Page(page content.Page) string {
	return pageHTML(page, "")
}

func PageWithCTA(page content.Page, policy cta.Policy, registry sources.Registry) string {
	ctaHTML := ""
	if cta.CanRender(policy, page, registry) {
		ctaHTML = renderWhatsAppCTA(page, policy)
	}
	return pageHTML(page, ctaHTML)
}

func pageHTML(page content.Page, ctaHTML string) string {
	var out strings.Builder
	out.WriteString("<!doctype html>\n")
	out.WriteString(`<html lang="pt-BR">` + "\n<head>\n")
	out.WriteString(seo.RenderHead(page))
	out.WriteString("\n<style>")
	out.WriteString(baseCSS(ctaHTML != ""))
	out.WriteString("</style>\n</head>\n<body>\n")
	out.WriteString(`<header class="site-header">`)
	out.WriteString(`<a href="/" class="brand">Portal Jurídico Brasileiro</a>`)
	out.WriteString(`<nav aria-label="Navegação principal">`)
	out.WriteString(`<a href="/fontes/planalto/">Fontes</a>`)
	out.WriteString(`<a href="/buscar/">Busca</a>`)
	out.WriteString(`</nav></header>` + "\n")
	out.WriteString(`<main id="conteudo">` + "\n")
	out.WriteString("<h1>" + html.EscapeString(page.Heading) + "</h1>\n")
	out.WriteString(`<p class="summary">` + html.EscapeString(page.Summary) + "</p>\n")
	for _, section := range page.BodySections {
		out.WriteString("<section>\n<h2>" + html.EscapeString(section.Title) + "</h2>\n")
		out.WriteString("<p>" + html.EscapeString(section.Body) + "</p>\n</section>\n")
	}
	out.WriteString(renderProvenance(page))
	if page.LegalNotice != "" {
		out.WriteString(`<aside class="legal-notice">` + html.EscapeString(page.LegalNotice) + "</aside>\n")
	}
	out.WriteString(ctaHTML)
	out.WriteString(`<section aria-labelledby="links-internos">` + "\n")
	out.WriteString(`<h2 id="links-internos">Links internos</h2>` + "\n<ul>")
	for _, link := range page.InternalLinks {
		out.WriteString(`<li><a href="` + html.EscapeString(link) + `">` + html.EscapeString(labelFromLink(link)) + `</a></li>`)
	}
	out.WriteString("</ul>\n</section>\n</main>\n")
	out.WriteString(`<footer class="site-footer">`)
	out.WriteString(`<p>Estado editorial: ` + html.EscapeString(page.Status) + `; política: ` + indexState(page) + `.</p>`)
	out.WriteString(`<p>Publicado em ` + html.EscapeString(page.PublicationDate) + `; revisado em ` + html.EscapeString(orPending(page.ReviewedAt)) + `.</p>`)
	out.WriteString(`<p>Autor: ` + html.EscapeString(page.Author) + `; revisor: ` + html.EscapeString(orPending(page.Reviewer)) + `.</p>`)
	out.WriteString(`</footer>` + "\n</body>\n</html>\n")
	return out.String()
}

func renderWhatsAppCTA(page content.Page, policy cta.Policy) string {
	message := cta.ContextMessage(policy, page)
	href := "/contato/advogado/?origem=" + url.QueryEscape(page.Path) + "&mensagem=" + url.QueryEscape(message)
	return `<section class="cta-whatsapp" aria-labelledby="cta-whatsapp-titulo">` +
		`<h2 id="cta-whatsapp-titulo">Contratar advogado pelo WhatsApp</h2>` +
		`<p>Converse com atendimento jurídico para avaliar seu caso com segurança.</p>` +
		`<a href="` + html.EscapeString(href) + `" rel="nofollow" data-context-message="` + html.EscapeString(message) + `">Iniciar atendimento jurídico</a>` +
		`</section>` + "\n"
}

func renderProvenance(page content.Page) string {
	if len(page.SourceProvenance) == 0 {
		return ""
	}
	var out strings.Builder
	out.WriteString(`<section aria-labelledby="proveniencia">` + "\n")
	out.WriteString(`<h2 id="proveniencia">Proveniência</h2>` + "\n<ul>")
	for _, source := range page.SourceProvenance {
		out.WriteString(`<li><a href="` + html.EscapeString(source.SourceURL) + `">` + html.EscapeString(source.SourceName) + `</a>`)
		out.WriteString(` verificado em ` + html.EscapeString(source.CheckedAt) + `. ` + html.EscapeString(source.LicenseNote) + `</li>`)
	}
	out.WriteString("</ul>\n</section>\n")
	return out.String()
}

func indexState(page content.Page) string {
	if editorial.IsIndexable(page) {
		return "indexável"
	}
	return "não indexável"
}

func orPending(value string) string {
	if value == "" {
		return "pendente"
	}
	return value
}

func labelFromLink(link string) string {
	if link == "/" {
		return "Home"
	}
	clean := strings.Trim(link, "/")
	clean = strings.ReplaceAll(clean, "-", " ")
	clean = strings.ReplaceAll(clean, "/", " / ")
	if clean == "" {
		return "Home"
	}
	return strings.Title(clean)
}

func baseCSS(includeCTA bool) string {
	css := "body{font-family:Arial,sans-serif;line-height:1.6;margin:0;color:#1f2933;background:#fff;}" +
		".site-header,.site-footer{padding:16px 24px;background:#f2f4f7;}" +
		".site-header{display:flex;gap:24px;align-items:center;justify-content:space-between;}" +
		".brand{font-weight:700;color:#14213d;text-decoration:none;}" +
		"nav a{margin-left:16px;color:#0b5cad;}" +
		"main{max-width:880px;margin:0 auto;padding:32px 24px;}" +
		"h1{font-size:2rem;margin:0 0 16px;}" +
		"h2{font-size:1.25rem;margin-top:28px;}" +
		".summary{font-size:1.05rem;color:#344054;}" +
		".legal-notice{border-left:4px solid #8a6d00;background:#fff8d6;padding:12px;margin-top:24px;}"
	if includeCTA {
		css += ".cta-whatsapp{border:1px solid #c7d7c9;background:#f6fbf7;padding:16px;margin-top:24px;}" +
			".cta-whatsapp a{display:inline-block;background:#0b6b43;color:#fff;padding:10px 14px;text-decoration:none;}"
	}
	return css
}

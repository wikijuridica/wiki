package seo

import (
	"fmt"
	"html"
	"net/url"

	"portaljuridico/internal/content"
	"portaljuridico/internal/editorial"
)

const (
	TitleMinCharacters           = 20
	TitleMaxCharacters           = 65
	MetaDescriptionMinCharacters = 70
	MetaDescriptionMaxCharacters = 160
	ProjectMaxSnippetCharacters  = 160
)

type AppearanceIssue struct {
	Code    string
	Message string
}

func IsAbsoluteHTTPSURL(value string) bool {
	parsed, err := url.Parse(value)
	return err == nil && parsed.Scheme == "https" && parsed.Host != ""
}

func RenderHead(page content.Page) string {
	robots := editorial.RobotsDirective(page)
	if editorial.IsIndexable(page) {
		robots += fmt.Sprintf(",max-snippet:%d", ProjectMaxSnippetCharacters)
	}
	return `<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>` + html.EscapeString(page.Title) + `</title>
<meta name="description" content="` + html.EscapeString(page.MetaDescription) + `">
<link rel="canonical" href="` + html.EscapeString(page.CanonicalURL) + `">
<meta name="robots" content="` + robots + `">`
}

func ValidateSearchAppearance(page content.Page) []AppearanceIssue {
	if !editorial.IsIndexable(page) {
		return nil
	}
	issues := make([]AppearanceIssue, 0)
	titleLength := runeLen(page.Title)
	if titleLength < TitleMinCharacters {
		issues = append(issues, AppearanceIssue{
			Code:    "title_too_short",
			Message: fmt.Sprintf("title com %d caracteres; minimo do projeto e %d", titleLength, TitleMinCharacters),
		})
	}
	if titleLength > TitleMaxCharacters {
		issues = append(issues, AppearanceIssue{
			Code:    "title_too_long",
			Message: fmt.Sprintf("title com %d caracteres; maximo conservador do projeto e %d", titleLength, TitleMaxCharacters),
		})
	}
	descriptionLength := runeLen(page.MetaDescription)
	if descriptionLength < MetaDescriptionMinCharacters {
		issues = append(issues, AppearanceIssue{
			Code:    "meta_description_too_short",
			Message: fmt.Sprintf("meta description com %d caracteres; minimo do projeto e %d", descriptionLength, MetaDescriptionMinCharacters),
		})
	}
	if descriptionLength > MetaDescriptionMaxCharacters {
		issues = append(issues, AppearanceIssue{
			Code:    "meta_description_too_long",
			Message: fmt.Sprintf("meta description com %d caracteres; maximo conservador do projeto e %d", descriptionLength, MetaDescriptionMaxCharacters),
		})
	}
	return issues
}

func runeLen(value string) int {
	return len([]rune(value))
}

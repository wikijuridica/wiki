package editorial

import "portaljuridico/internal/content"

var ValidStatuses = map[string]bool{
	"draft":        true,
	"needs_review": true,
	"approved":     true,
	"published":    true,
	"noindex":      true,
	"archived":     true,
}

var ValidIndexPolicies = map[string]bool{
	"index":   true,
	"noindex": true,
}

func IsIndexable(page content.Page) bool {
	return page.Status == "published" && page.IndexPolicy == "index"
}

func RobotsDirective(page content.Page) string {
	if IsIndexable(page) {
		return "index,follow"
	}
	return "noindex,follow"
}

package legal

import "portaljuridico/internal/content"

func RequiresLegalControls(page content.Page) bool {
	return page.IsLegalContent()
}

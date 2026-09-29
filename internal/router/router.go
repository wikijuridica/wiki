package router

import (
	"net/url"
	"path/filepath"
	"strings"
	"unicode"
)

func IsCleanPublicPath(path string) bool {
	if !strings.HasPrefix(path, "/") {
		return false
	}
	if strings.ContainsAny(path, "?#") {
		return false
	}
	if path != "/" && !strings.HasSuffix(path, "/") {
		return false
	}
	for _, r := range path {
		if r > 127 || unicode.IsUpper(r) {
			return false
		}
	}
	return true
}

func OutputFileForPath(path string, outputDir string) string {
	if path == "/" {
		return filepath.Join(outputDir, "index.html")
	}
	clean := strings.Trim(path, "/")
	return filepath.Join(outputDir, filepath.FromSlash(clean), "index.html")
}

func CanonicalPath(canonicalURL string) string {
	parsed, err := url.Parse(canonicalURL)
	if err != nil || parsed.RawQuery != "" || parsed.Fragment != "" {
		return ""
	}
	if parsed.Path == "" {
		return "/"
	}
	return parsed.Path
}

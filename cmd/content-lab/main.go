package main

import (
	"fmt"
	"os"
	"strings"

	"portaljuridico/internal/quality"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: content-lab natural:/tmp/natural.txt mechanical:/tmp/mechanical.txt")
		os.Exit(2)
	}

	failures := make([]string, 0)
	for _, spec := range os.Args[1:] {
		label, path, ok := parseSpec(spec)
		if !ok {
			failures = append(failures, "invalid_spec="+spec)
			continue
		}
		data, err := os.ReadFile(path)
		if err != nil {
			failures = append(failures, err.Error())
			continue
		}
		analysis := quality.AnalyzeText(string(data))
		fmt.Printf("%s: words=%d signal_words=%d diversity=%.2f issues=%s\n", label, analysis.WordCount, analysis.SignalWords, analysis.LexicalDiversity, strings.Join(analysis.Messages(), " | "))
		switch label {
		case "natural":
			if !analysis.Passed() {
				failures = append(failures, "natural_content_rejected="+strings.Join(analysis.Messages(), " | "))
			}
		case "mechanical":
			if !analysis.HasIssue("mechanical_keyword_permutation") {
				failures = append(failures, "mechanical_content_not_blocked="+strings.Join(analysis.Messages(), " | "))
			}
		default:
			failures = append(failures, "unknown_label="+label)
		}
	}

	if len(failures) > 0 {
		for _, failure := range failures {
			fmt.Println(failure)
		}
		os.Exit(1)
	}
}

func parseSpec(spec string) (string, string, bool) {
	parts := strings.SplitN(spec, ":", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", false
	}
	return parts[0], parts[1], true
}

package main

import (
	"fmt"
	"os"
	"path/filepath"

	"portaljuridico/internal/draftlab"
	"portaljuridico/internal/terms"
)

func main() {
	root := "."
	if len(os.Args) > 1 {
		root = os.Args[1]
	}
	entries, report := terms.LoadSeeds(root)
	if !report.Passed() {
		for _, message := range report.Messages() {
			fmt.Println(message)
		}
		os.Exit(1)
	}
	if len(entries) == 0 {
		fmt.Println("term_draft_lab: no seeds")
		return
	}

	outputDir, err := os.MkdirTemp("", "opt-wiki-term-draft-")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	failures := make([]string, 0)
	for _, entry := range entries {
		draft, err := draftlab.Build(entry.Seed)
		if err != nil {
			failures = append(failures, fmt.Sprintf("line=%d %s", entry.Line, err.Error()))
			continue
		}
		path := filepath.Join(outputDir, draft.TermID+".txt")
		if err := os.WriteFile(path, []byte(draft.Text+"\n"), 0644); err != nil {
			failures = append(failures, err.Error())
			continue
		}
		fmt.Printf("draft=%s status=%s index=%s source=%s path=%s\n", draft.TermID, draft.Status, draft.IndexPolicy, draft.SourceID, path)
	}
	if len(failures) > 0 {
		for _, failure := range failures {
			fmt.Println("failure=" + failure)
		}
		os.Exit(1)
	}
}

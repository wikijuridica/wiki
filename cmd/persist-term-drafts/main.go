package main

import (
	"fmt"
	"os"

	"portaljuridico/internal/draftlab"
	"portaljuridico/internal/editorialdrafts"
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
	failures := make([]string, 0)
	for _, entry := range entries {
		draft, err := draftlab.Build(entry.Seed)
		if err != nil {
			failures = append(failures, fmt.Sprintf("line=%d %s", entry.Line, err.Error()))
			continue
		}
		inserted, err := editorialdrafts.AppendIfMissing(root, draft)
		if err != nil {
			failures = append(failures, fmt.Sprintf("line=%d %s", entry.Line, err.Error()))
			continue
		}
		if inserted {
			fmt.Printf("persisted=%s status=%s index=%s source=%s\n", draft.TermID, draft.Status, draft.IndexPolicy, draft.SourceID)
		} else {
			fmt.Printf("existing=%s status=%s index=%s source=%s\n", draft.TermID, draft.Status, draft.IndexPolicy, draft.SourceID)
		}
	}
	if len(failures) > 0 {
		for _, failure := range failures {
			fmt.Println("failure=" + failure)
		}
		os.Exit(1)
	}
}

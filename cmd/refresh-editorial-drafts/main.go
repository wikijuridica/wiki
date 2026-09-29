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
	drafts := make([]draftlab.Draft, 0, len(entries))
	for _, entry := range entries {
		draft, err := draftlab.Build(entry.Seed)
		if err != nil {
			fmt.Printf("failure=line=%d %s\n", entry.Line, err.Error())
			os.Exit(1)
		}
		drafts = append(drafts, draft)
	}
	if err := editorialdrafts.UpsertAll(root, drafts); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	for _, draft := range drafts {
		fmt.Printf("refreshed=%s status=%s index=%s source=%s\n", draft.TermID, draft.Status, draft.IndexPolicy, draft.SourceID)
	}
}

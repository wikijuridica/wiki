package main

import (
	"fmt"
	"os"
	"time"

	"portaljuridico/internal/editorialdrafts"
	"portaljuridico/internal/reviewqueue"
)

func main() {
	root := "."
	if len(os.Args) > 1 {
		root = os.Args[1]
	}
	drafts, report := editorialdrafts.LoadRecords(root)
	if !report.Passed() {
		for _, message := range report.Messages() {
			fmt.Println(message)
		}
		os.Exit(1)
	}
	failures := make([]string, 0)
	for _, record := range drafts {
		inserted, err := reviewqueue.EnqueueDraft(root, record.AsDraft(), reviewqueue.Metadata{
			Author:    "Redacao tecnica",
			CreatedAt: time.Now().Format("2006-01-02"),
			Reason:    "Rascunho persistido em laboratorio pronto para revisao editorial; publicacao segue bloqueada.",
		})
		if err != nil {
			failures = append(failures, record.TermID+":"+err.Error())
			continue
		}
		if inserted {
			fmt.Printf("queued=%s status=needs_review publication_allowed=false\n", record.TermID)
		} else {
			fmt.Printf("existing=%s status=needs_review publication_allowed=false\n", record.TermID)
		}
	}
	if len(failures) > 0 {
		for _, failure := range failures {
			fmt.Println("failure=" + failure)
		}
		os.Exit(1)
	}
}

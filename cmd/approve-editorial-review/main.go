package main

import (
	"fmt"
	"os"
	"time"

	"portaljuridico/internal/approvals"
	"portaljuridico/internal/reviewqueue"
)

func main() {
	root := "."
	if len(os.Args) > 1 {
		root = os.Args[1]
	}
	queued, report := reviewqueue.LoadRecords(root)
	if !report.Passed() {
		for _, message := range report.Messages() {
			fmt.Println(message)
		}
		os.Exit(1)
	}
	failures := make([]string, 0)
	for _, record := range queued {
		inserted, err := approvals.Approve(root, record, approvals.Metadata{
			Reviewer:   "Revisao juridica",
			ApprovedAt: time.Now().Format("2006-01-02"),
			Reason:     "Aprovacao editorial de laboratorio; publicacao, URL e CTA seguem bloqueados.",
		})
		if err != nil {
			failures = append(failures, record.TermID+":"+err.Error())
			continue
		}
		if inserted {
			fmt.Printf("approved=%s editorial_status=approved publication_allowed=false\n", record.TermID)
		} else {
			fmt.Printf("existing=%s editorial_status=approved publication_allowed=false\n", record.TermID)
		}
	}
	if len(failures) > 0 {
		for _, failure := range failures {
			fmt.Println("failure=" + failure)
		}
		os.Exit(1)
	}
}

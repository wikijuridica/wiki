package main

import (
	"fmt"
	"os"
	"time"

	"portaljuridico/internal/approvals"
	"portaljuridico/internal/publicationblockers"
)

func main() {
	root := "."
	if len(os.Args) > 1 {
		root = os.Args[1]
	}
	approved, report := approvals.LoadRecords(root)
	if !report.Passed() {
		for _, message := range report.Messages() {
			fmt.Println(message)
		}
		os.Exit(1)
	}
	failures := make([]string, 0)
	for _, record := range approved {
		inserted, err := publicationblockers.AppendFromApproval(root, record, time.Now().Format("2006-01-02"))
		if err != nil {
			failures = append(failures, record.TermID+":"+err.Error())
			continue
		}
		if inserted {
			fmt.Printf("blocked=%s publication_allowed=false public_path=none\n", record.TermID)
		} else {
			fmt.Printf("existing=%s publication_allowed=false public_path=none\n", record.TermID)
		}
	}
	if len(failures) > 0 {
		for _, failure := range failures {
			fmt.Println("failure=" + failure)
		}
		os.Exit(1)
	}
}

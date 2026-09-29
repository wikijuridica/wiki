package main

import (
	"fmt"
	"os"

	"portaljuridico/internal/batchcandidategates"
	"portaljuridico/internal/batchcandidatepromotion"
)

func main() {
	root := "."
	if len(os.Args) > 1 {
		root = os.Args[1]
	}

	records, report := batchcandidatepromotion.ExpandFromReadiness(root)
	if !report.Passed() {
		for _, message := range report.Messages() {
			fmt.Println("expand-batch-candidate-gates: " + message)
		}
		os.Exit(1)
	}
	if err := batchcandidategates.WriteRecords(root, records); err != nil {
		fmt.Println("expand-batch-candidate-gates: " + err.Error())
		os.Exit(1)
	}

	total := 0
	for _, record := range records {
		total += len(record.SelectedUniqueIntentIDs)
	}
	fmt.Printf("expand-batch-candidate-gates: families=%d selected=%d\n", len(records), total)
}

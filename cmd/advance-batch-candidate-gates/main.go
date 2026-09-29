package main

import (
	"flag"
	"fmt"
	"os"

	"portaljuridico/internal/batchcandidategates"
	"portaljuridico/internal/batchcandidatepromotion"
)

func main() {
	flags := flag.NewFlagSet("advance-batch-candidate-gates", flag.ExitOnError)
	expectedTotal := flags.Int("expect-total", 0, "expected total selected candidates after explicit advance")
	if err := flags.Parse(os.Args[1:]); err != nil {
		fmt.Println("advance-batch-candidate-gates: " + err.Error())
		os.Exit(1)
	}
	root := "."
	if flags.NArg() > 0 {
		root = flags.Arg(0)
	}
	if *expectedTotal <= 0 {
		fmt.Println("advance-batch-candidate-gates: --expect-total is required")
		os.Exit(1)
	}

	currentRecords, currentReport := batchcandidategates.LoadRecords(root)
	if !currentReport.Passed() {
		for _, message := range currentReport.Messages() {
			fmt.Println("advance-batch-candidate-gates: " + message)
		}
		os.Exit(1)
	}
	if currentTotalSelected(currentRecords) == *expectedTotal {
		if err := batchcandidategates.WriteRecords(root, recordsFromEntries(currentRecords)); err != nil {
			fmt.Println("advance-batch-candidate-gates: " + err.Error())
			os.Exit(1)
		}
		fmt.Printf("advance-batch-candidate-gates: already selected=%d normalized=true\n", *expectedTotal)
		return
	}

	records, report := batchcandidatepromotion.AdvanceToNextTargets(root)
	if !report.Passed() {
		for _, message := range report.Messages() {
			fmt.Println("advance-batch-candidate-gates: " + message)
		}
		os.Exit(1)
	}
	if totalReport := batchcandidatepromotion.ValidateTotalSelected(records, *expectedTotal); !totalReport.Passed() {
		for _, message := range totalReport.Messages() {
			fmt.Println("advance-batch-candidate-gates: " + message)
		}
		os.Exit(1)
	}
	if err := batchcandidategates.WriteRecords(root, records); err != nil {
		fmt.Println("advance-batch-candidate-gates: " + err.Error())
		os.Exit(1)
	}

	total := 0
	for _, record := range records {
		total += len(record.SelectedUniqueIntentIDs)
	}
	fmt.Printf("advance-batch-candidate-gates: families=%d selected=%d\n", len(records), total)
}

func currentTotalSelected(entries []batchcandidategates.Entry) int {
	total := 0
	for _, entry := range entries {
		total += len(entry.Record.SelectedUniqueIntentIDs)
	}
	return total
}

func recordsFromEntries(entries []batchcandidategates.Entry) []batchcandidategates.Record {
	records := make([]batchcandidategates.Record, 0, len(entries))
	for _, entry := range entries {
		records = append(records, entry.Record)
	}
	return records
}

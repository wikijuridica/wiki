package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"portaljuridico/internal/batchcandidateexpansion"
	"portaljuridico/internal/batchexpansionapply"
)

func main() {
	root := "."
	if len(os.Args) > 1 {
		root = os.Args[1]
	}

	records, report := batchcandidateexpansion.RefreshRecords(root)
	if !report.Passed() {
		for _, message := range report.Messages() {
			fmt.Println("refresh-expansion-readiness: " + message)
		}
		os.Exit(1)
	}
	records, applyReport := batchexpansionapply.ApplyStrategy(root, records)
	if !applyReport.Passed() {
		for _, message := range applyReport.Messages() {
			fmt.Println("refresh-expansion-readiness: " + message)
		}
		os.Exit(1)
	}

	path := filepath.Join(root, "data", "editorial", "batch_candidate_expansion_readiness.jsonl")
	file, err := os.Create(path)
	if err != nil {
		fmt.Println("refresh-expansion-readiness: write failed: " + err.Error())
		os.Exit(1)
	}
	defer file.Close()

	for _, record := range records {
		encoded, err := json.Marshal(record)
		if err != nil {
			fmt.Println("refresh-expansion-readiness: encode failed: " + err.Error())
			os.Exit(1)
		}
		if _, err := file.Write(append(encoded, '\n')); err != nil {
			fmt.Println("refresh-expansion-readiness: write failed: " + err.Error())
			os.Exit(1)
		}
	}

	fmt.Printf("refresh-expansion-readiness: wrote %d records to %s\n", len(records), path)
}

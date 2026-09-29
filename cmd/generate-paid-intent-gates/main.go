package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"portaljuridico/internal/paidintent"
)

func main() {
	root := "."
	if len(os.Args) > 1 {
		root = os.Args[1]
	}

	records, report := paidintent.BuildRepositoryRecords(root)
	if !report.Passed() {
		for _, message := range report.Messages() {
			fmt.Println("generate-paid-intent-gates: " + message)
		}
		os.Exit(1)
	}

	path := filepath.Join(root, "data", "editorial", "batch_paid_intent_gates.jsonl")
	file, err := os.Create(path)
	if err != nil {
		fmt.Println("generate-paid-intent-gates: write failed: " + err.Error())
		os.Exit(1)
	}
	defer file.Close()

	for _, record := range records {
		encoded, err := json.Marshal(record)
		if err != nil {
			fmt.Println("generate-paid-intent-gates: encode failed: " + err.Error())
			os.Exit(1)
		}
		if _, err := file.Write(append(encoded, '\n')); err != nil {
			fmt.Println("generate-paid-intent-gates: write failed: " + err.Error())
			os.Exit(1)
		}
	}

	fmt.Printf("generate-paid-intent-gates: wrote %d records to %s\n", len(records), path)
}

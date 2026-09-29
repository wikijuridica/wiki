package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"portaljuridico/internal/batchexpansionstrategy"
)

func main() {
	root := "."
	if len(os.Args) > 1 {
		root = os.Args[1]
	}

	records, report := batchexpansionstrategy.RefreshRecords(root)
	if !report.Passed() {
		for _, message := range report.Messages() {
			fmt.Println("refresh-batch-expansion-strategy: " + message)
		}
		os.Exit(1)
	}

	path := filepath.Join(root, batchexpansionstrategy.Path)
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		fmt.Println("refresh-batch-expansion-strategy: mkdir failed: " + err.Error())
		os.Exit(1)
	}
	file, err := os.Create(path)
	if err != nil {
		fmt.Println("refresh-batch-expansion-strategy: write failed: " + err.Error())
		os.Exit(1)
	}
	defer file.Close()

	for _, record := range records {
		encoded, err := json.Marshal(record)
		if err != nil {
			fmt.Println("refresh-batch-expansion-strategy: encode failed: " + err.Error())
			os.Exit(1)
		}
		if _, err := file.Write(append(encoded, '\n')); err != nil {
			fmt.Println("refresh-batch-expansion-strategy: write failed: " + err.Error())
			os.Exit(1)
		}
	}
	fmt.Printf("refresh-batch-expansion-strategy: wrote %d records to %s\n", len(records), path)
}

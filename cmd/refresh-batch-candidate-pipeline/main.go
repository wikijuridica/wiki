package main

import (
	"fmt"
	"os"

	"portaljuridico/internal/batchcandidatepipeline"
)

func main() {
	root := "."
	if len(os.Args) > 1 {
		root = os.Args[1]
	}

	result, report := batchcandidatepipeline.Refresh(root)
	if !report.Passed() {
		for _, message := range report.Messages() {
			fmt.Println("refresh-batch-candidate-pipeline: " + message)
		}
		os.Exit(1)
	}

	fmt.Printf("refresh-batch-candidate-pipeline: selected=%d reviews=%d prepublication=%d source_specificity=%d manifest=%d final_drafts=%d locked=%d blocked=%d\n",
		result.SelectedCandidates,
		result.Reviews,
		result.Prepublication,
		result.SourceSpecificity,
		result.PublicManifest,
		result.FinalDrafts,
		result.SourceLocked,
		result.SourceBlocked,
	)
}

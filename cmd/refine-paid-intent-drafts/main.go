package main

import (
	"fmt"
	"os"

	"portaljuridico/internal/paidintentrefinement"
)

func main() {
	root := "."
	if len(os.Args) > 1 {
		root = os.Args[1]
	}

	result, report := paidintentrefinement.RefineRepository(root)
	if !report.Passed() {
		for _, message := range report.Messages() {
			fmt.Println("refine-paid-intent-drafts: " + message)
		}
		os.Exit(1)
	}

	fmt.Printf(
		"refine-paid-intent-drafts: refinements=%d archive_refined=%d final_refined=%d\n",
		len(result.Refinements),
		result.ArchiveRefined,
		result.FinalRefined,
	)
}

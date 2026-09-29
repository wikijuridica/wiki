package main

import (
	"fmt"
	"os"
	"strconv"

	"portaljuridico/internal/termpromotion"
)

func main() {
	limit := 6
	if len(os.Args) > 1 {
		parsed, err := strconv.Atoi(os.Args[1])
		if err != nil || parsed <= 0 {
			fmt.Fprintf(os.Stderr, "invalid_limit=%q\n", os.Args[1])
			os.Exit(1)
		}
		limit = parsed
	}
	result, err := termpromotion.PromoteTopCandidates(".", limit)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	for _, seed := range result.Added {
		fmt.Printf("promoted=%s status=%s mode=%s publication_allowed=false\n", seed.TermID, seed.QualityState, seed.OnlineServiceMode)
	}
	for _, seed := range result.Skipped {
		fmt.Printf("existing=%s status=%s mode=%s publication_allowed=false\n", seed.TermID, seed.QualityState, seed.OnlineServiceMode)
	}
}

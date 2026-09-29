package main

import (
	"fmt"
	"os"

	"portaljuridico/internal/checks"
)

func main() {
	selected := checks.Names
	if len(os.Args) > 1 && os.Args[1] != "all" {
		selected = []string{os.Args[1]}
	}

	failures := make([]string, 0)
	for _, name := range selected {
		errors := checks.Run(name, ".")
		if len(errors) == 0 {
			fmt.Println(name + ": pass")
			continue
		}
		for _, err := range errors {
			failures = append(failures, name+": "+err)
		}
	}

	if len(failures) > 0 {
		for _, failure := range failures {
			fmt.Println(failure)
		}
		os.Exit(1)
	}
}

package main

import (
	"fmt"
	"os"

	"portaljuridico/internal/build"
	"portaljuridico/internal/content"
)

func main() {
	outputDir := "public"
	if len(os.Args) > 1 {
		outputDir = os.Args[1]
	}
	repo, err := content.LoadRepository(".")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	result, err := build.Site(repo, outputDir)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Println(build.FormatResult(result))
}

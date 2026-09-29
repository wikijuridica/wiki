package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"portaljuridico/internal/batchdraftgen"
)

func main() {
	samples := flag.Int("samples-per-batch", 5, "quantidade de amostras por lote juridico")
	checkedAt := flag.String("checked-at", "2026-06-09", "data de validacao em YYYY-MM-DD")
	writeSamples := flag.Bool("write-samples", false, "persistir amostras validadas em data/editorial/batch_drafts.jsonl")
	writeMetrics := flag.Bool("write-metrics", false, "persistir metricas em data/editorial/batch_generation_metrics.jsonl")
	writeArchive := flag.Bool("write-archive", false, "persistir rascunhos validados em data/editorial/batch_draft_expansion_archive.jsonl")
	flag.Parse()

	root := "."
	if flag.NArg() > 0 {
		root = flag.Arg(0)
	}

	result, report := batchdraftgen.Generate(root, batchdraftgen.Options{SamplesPerBatch: *samples, CheckedAt: *checkedAt})
	if !report.Passed() {
		fmt.Println(strings.Join(report.Messages(), "\n"))
		os.Exit(1)
	}
	if *writeSamples {
		if err := batchdraftgen.WriteSamples(root, result.Drafts); err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
	}
	if *writeMetrics {
		if err := batchdraftgen.WriteMetrics(root, result.Metrics); err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
	}
	if *writeArchive {
		if err := batchdraftgen.WriteArchive(root, result.Drafts); err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
	}

	outputDir, err := os.MkdirTemp("", "opt-wiki-batch-draft-generation-")
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	if err := writeJSONL(filepath.Join(outputDir, "batch_drafts.jsonl"), result.Drafts); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	if err := writeJSONL(filepath.Join(outputDir, "batch_generation_metrics.jsonl"), result.Metrics); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	fmt.Printf("generated_drafts=%d rewritten=%d max_similarity=%.2f metrics=%d output_dir=%s\n", len(result.Drafts), result.RewrittenCount(), result.MaximumPairSimilarity(), len(result.Metrics), outputDir)
}

func writeJSONL(path string, records any) error {
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	data, err := json.Marshal(records)
	if err != nil {
		return err
	}
	var list []json.RawMessage
	if err := json.Unmarshal(data, &list); err != nil {
		return err
	}
	for _, record := range list {
		if _, err := file.Write(append(record, '\n')); err != nil {
			return err
		}
	}
	return nil
}

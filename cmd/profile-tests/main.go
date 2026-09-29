package main

import (
	"bytes"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"strconv"

	"portaljuridico/internal/testprofile"
)

func main() {
	pkg := flag.String("package", "./internal/contract", "go test package to profile")
	threshold := flag.Float64("threshold-seconds", 5, "minimum elapsed seconds to report")
	top := flag.Int("top", 12, "maximum slow tests to print")
	flag.Parse()

	cmd := exec.Command("go", "test", "-json", "-count=1", *pkg)
	cmd.Env = ensureGoCache(os.Environ())
	output, err := cmd.CombinedOutput()
	if err != nil {
		os.Stdout.Write(output)
		fmt.Fprintf(os.Stderr, "profile_test_command_failed=%v\n", err)
		os.Exit(1)
	}

	profile, err := testprofile.Parse(bytes.NewReader(output), *threshold)
	if err != nil {
		fmt.Fprintf(os.Stderr, "profile_parse_failed=%v\n", err)
		os.Exit(1)
	}
	slowest := profile.Slowest(*top)
	fmt.Printf("profile_package=%s threshold_seconds=%.2f total_observed_seconds=%.2f slow_tests=%d\n", *pkg, *threshold, profile.TotalObservedSeconds(), len(slowest))
	for i, test := range slowest {
		fmt.Printf("slow_test rank=%d seconds=%s package=%s test=%s\n", i+1, trimFloat(test.ElapsedSeconds), test.Package, test.Name)
	}
	fmt.Println("optimization_strategy=measure_before_commit,dedupe_lab_cycle,avoid_revalidating_upstream_indexes,shard_internal_contract_when_scale_tests_dominate")
}

func ensureGoCache(env []string) []string {
	for _, item := range env {
		if len(item) >= len("GOCACHE=") && item[:len("GOCACHE=")] == "GOCACHE=" {
			return env
		}
	}
	return append(env, "GOCACHE=/tmp/opt-wiki-go-cache")
}

func trimFloat(value float64) string {
	return strconv.FormatFloat(value, 'f', 2, 64)
}

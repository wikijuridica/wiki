package contract_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"portaljuridico/internal/testprofile"
)

func TestLabCycleTimesStagesWithoutDuplicateGlobalChecks(t *testing.T) {
	root := projectRoot(t)
	data, err := os.ReadFile(filepath.Join(root, "tools", "lab-cycle"))
	if err != nil {
		t.Fatalf("could not read lab-cycle: %v", err)
	}
	script := string(data)
	for _, token := range []string{"run_step", "TIMING start", "TIMING pass", "go test -count=1 ./...", "go run ./cmd/check all"} {
		if !strings.Contains(script, token) {
			t.Fatalf("lab-cycle missing timing/dedup token %q", token)
		}
	}
	if strings.Contains(script, "./tools/check-all") {
		t.Fatal("lab-cycle must not call check-all after go test; use go run ./cmd/check all to avoid duplicate test execution")
	}
	for _, duplicate := range []string{
		"./tools/check-batch-candidate-expansion-readiness",
		"./tools/check-batch-final-authorial-drafts",
		"./tools/check-paid-intent",
		"./tools/check-storage-contract",
	} {
		if strings.Count(script, duplicate) > 0 {
			t.Fatalf("lab-cycle repeats named check already covered by cmd/check all: %s", duplicate)
		}
	}
}

func TestContractTestProfileParserRanksSlowTests(t *testing.T) {
	events := strings.Join([]string{
		`{"Action":"pass","Package":"portaljuridico/internal/contract","Test":"Fast","Elapsed":0.03}`,
		`{"Action":"pass","Package":"portaljuridico/internal/contract","Test":"Slow","Elapsed":12.5}`,
		`{"Action":"pass","Package":"portaljuridico/internal/contract","Test":"Medium","Elapsed":3.2}`,
	}, "\n")

	profile, err := testprofile.Parse(strings.NewReader(events), 5)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	slow := profile.Slowest(2)
	if len(slow) != 1 || slow[0].Name != "Slow" || slow[0].ElapsedSeconds != 12.5 {
		t.Fatalf("unexpected slow tests: %#v", slow)
	}
	if profile.TotalObservedSeconds() != 15.73 {
		t.Fatalf("total seconds=%.2f, want 15.73", profile.TotalObservedSeconds())
	}
}

func TestProfileContractTestsToolExistsForPreCommitTiming(t *testing.T) {
	root := projectRoot(t)
	data, err := os.ReadFile(filepath.Join(root, "tools", "profile-contract-tests"))
	if err != nil {
		t.Fatalf("missing profile-contract-tests tool: %v", err)
	}
	script := string(data)
	for _, token := range []string{"go run ./cmd/profile-tests", "./internal/contract", "--threshold-seconds"} {
		if !strings.Contains(script, token) {
			t.Fatalf("profile-contract-tests missing token %q", token)
		}
	}
}

func projectRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("could not get cwd: %v", err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("could not find go.mod")
		}
		dir = parent
	}
}

package checks

import "testing"

func TestCPUBudgetRejectsHeavyPublicRuntimePatterns(t *testing.T) {
	source := `package render
import "os/exec"
func Bad() {
	for {
		exec.Command("node").Run()
	}
}`

	issues := cpuBudgetIssuesForSource("internal/render/bad.go", source)

	for _, want := range []string{"public_runtime_exec_not_allowed", "unbounded_loop_not_allowed"} {
		if !containsIssue(issues, want) {
			t.Fatalf("missing issue %q in %v", want, issues)
		}
	}
}

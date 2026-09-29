package testprofile

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"sort"
	"strings"
)

type TestTiming struct {
	Package        string
	Name           string
	ElapsedSeconds float64
}

type Profile struct {
	ThresholdSeconds float64
	Tests            []TestTiming
}

type event struct {
	Action  string  `json:"Action"`
	Package string  `json:"Package"`
	Test    string  `json:"Test"`
	Elapsed float64 `json:"Elapsed"`
}

func Parse(reader io.Reader, thresholdSeconds float64) (Profile, error) {
	profile := Profile{ThresholdSeconds: thresholdSeconds}
	scanner := bufio.NewScanner(reader)
	scanner.Buffer(make([]byte, 0, 4096), 1024*1024)
	lineNumber := 0
	for scanner.Scan() {
		lineNumber++
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var item event
		if err := json.Unmarshal([]byte(line), &item); err != nil {
			return profile, fmt.Errorf("line %d: %w", lineNumber, err)
		}
		if item.Action != "pass" || item.Test == "" {
			continue
		}
		profile.Tests = append(profile.Tests, TestTiming{
			Package:        item.Package,
			Name:           item.Test,
			ElapsedSeconds: item.Elapsed,
		})
	}
	if err := scanner.Err(); err != nil {
		return profile, err
	}
	return profile, nil
}

func (profile Profile) Slowest(limit int) []TestTiming {
	tests := make([]TestTiming, 0)
	for _, test := range profile.Tests {
		if test.ElapsedSeconds >= profile.ThresholdSeconds {
			tests = append(tests, test)
		}
	}
	sort.Slice(tests, func(i, j int) bool {
		if tests[i].ElapsedSeconds == tests[j].ElapsedSeconds {
			return tests[i].Name < tests[j].Name
		}
		return tests[i].ElapsedSeconds > tests[j].ElapsedSeconds
	})
	if limit > 0 && len(tests) > limit {
		tests = tests[:limit]
	}
	return tests
}

func (profile Profile) TotalObservedSeconds() float64 {
	total := 0.0
	for _, test := range profile.Tests {
		total += test.ElapsedSeconds
	}
	return math.Round(total*100) / 100
}

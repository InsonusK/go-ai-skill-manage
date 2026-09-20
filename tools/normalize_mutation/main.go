// Command normalize_mutation reads gremlins' own JSON mutation report (its
// path is argv[1]) and writes the normalized tmp/result/mutation-test.json
// the solution-conformance-testing report contract defines, plus a small
// human-readable tmp/report/mutation/index.html table.
//
// gremlins v0.6.0's report nests per-mutation status under files[].mutations
// (not a flat top-level list), and its status strings are "KILLED",
// "LIVED", "NOT COVERED", and "TIMED OUT" — verified against a real
// `gremlins unleash --output` run, not assumed.
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

type gremlinsReport struct {
	Files []struct {
		Mutations []struct {
			Status string `json:"status"`
		} `json:"mutations"`
	} `json:"files"`
}

type normalized struct {
	Killed     int     `json:"killed"`
	Survived   int     `json:"survived"`
	TimedOut   int     `json:"timedout"`
	NoCoverage int     `json:"noCoverage"`
	Score      float64 `json:"score"`
}

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintln(os.Stderr, "usage: normalize_mutation <gremlins-report.json>")
		os.Exit(1)
	}
	if err := run(os.Args[1]); err != nil {
		fmt.Fprintln(os.Stderr, "normalize_mutation:", err)
		os.Exit(1)
	}
}

func run(reportPath string) error {
	data, err := os.ReadFile(reportPath)
	if err != nil {
		return err
	}
	var report gremlinsReport
	if err := json.Unmarshal(data, &report); err != nil {
		return err
	}

	n := normalized{}
	for _, f := range report.Files {
		for _, m := range f.Mutations {
			switch strings.ToUpper(m.Status) {
			case "KILLED":
				n.Killed++
			case "LIVED", "SURVIVED":
				n.Survived++
			case "TIMED OUT", "TIMEDOUT", "TIMED_OUT":
				n.TimedOut++
			case "NOT COVERED", "NOTCOVERED", "NOT_COVERED", "NO COVERAGE":
				n.NoCoverage++
			}
		}
	}

	total := n.Killed + n.Survived + n.TimedOut + n.NoCoverage
	if total > 0 {
		n.Score = round1(float64(n.Killed) / float64(total) * 100)
	}

	if err := os.MkdirAll("tmp/result", 0o755); err != nil {
		return err
	}
	out, err := json.Marshal(n)
	if err != nil {
		return err
	}
	if err := os.WriteFile("tmp/result/mutation-test.json", out, 0o644); err != nil {
		return err
	}

	if err := os.MkdirAll("tmp/report/mutation", 0o755); err != nil {
		return err
	}
	return os.WriteFile("tmp/report/mutation/index.html", []byte(renderHTML(n)), 0o644)
}

func round1(v float64) float64 {
	return float64(int(v*10+0.5)) / 10
}

func renderHTML(n normalized) string {
	return fmt.Sprintf(`<!doctype html><html><head><meta charset="utf-8"><title>Mutation report</title></head>
<body><table border="1">
<tr><th>Killed</th><th>Survived</th><th>Timed out</th><th>No coverage</th><th>Score</th></tr>
<tr><td>%d</td><td>%d</td><td>%d</td><td>%d</td><td>%.1f%%</td></tr>
</table><p><a href="gremlins.json">Native report: every mutation, file, line and status</a></p></body></html>`, n.Killed, n.Survived, n.TimedOut, n.NoCoverage, n.Score)
}

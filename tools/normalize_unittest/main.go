// Command normalize_unittest reads `go test -json` events from stdin and
// writes the normalized tmp/result/unit-test.json the solution-conformance-testing
// report contract defines. It counts each leaf test exactly once: a godog
// scenario run as a Go subtest of TestFeatures reports its own pass/fail
// alongside TestFeatures' own - only the deepest name per branch is counted.
package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"html"
	"os"
	"sort"
	"strings"
)

type testEvent struct {
	Action  string
	Package string
	Test    string
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "normalize_unittest:", err)
		os.Exit(1)
	}
}

func run() error {
	packageFailures := map[string]bool{}
	results := map[string]string{} // "{package}/{test}" -> last pass/fail/skip action seen
	sc := bufio.NewScanner(os.Stdin)
	sc.Buffer(make([]byte, 64*1024), 1024*1024)
	for sc.Scan() {
		var ev testEvent
		if err := json.Unmarshal(sc.Bytes(), &ev); err != nil {
			continue // a non-JSON line (build output on stderr interleaved by a wrapper); ignore
		}
		if ev.Test == "" {
			if ev.Action == "fail" {
				packageFailures[ev.Package] = true
			}
			continue // package-level event, not a test result
		}
		switch ev.Action {
		case "pass", "fail", "skip":
			results[ev.Package+"/"+ev.Test] = ev.Action
		}
	}
	if err := sc.Err(); err != nil {
		return err
	}

	for pkg := range packageFailures {
		hasFailure := false
		for name, result := range results {
			if strings.HasPrefix(name, pkg+"/") && result == "fail" {
				hasFailure = true
			}
		}
		if !hasFailure {
			results[pkg+"/package failure"] = "fail"
		}
	}
	total, passed, failed := 0, 0, 0
	for _, name := range leafNames(results) {
		total++
		switch results[name] {
		case "pass":
			passed++
		case "fail":
			failed++
		}
	}

	if err := os.MkdirAll("tmp/result", 0o755); err != nil {
		return err
	}
	data, err := json.Marshal(struct {
		Total  int `json:"total"`
		Passed int `json:"passed"`
		Failed int `json:"failed"`
	}{total, passed, failed})
	if err != nil {
		return err
	}
	if err := os.WriteFile("tmp/result/unit-test.json", data, 0o644); err != nil {
		return err
	}
	if err := os.MkdirAll("tmp/report/tests", 0755); err != nil {
		return err
	}
	var page strings.Builder
	page.WriteString(`<!doctype html><html lang="en"><meta charset="utf-8"><title>Scenario results</title><style>body{font:16px system-ui;margin:2rem}td,th{padding:.4rem;text-align:left}tr:nth-child(even){background:#eee}</style><h1>Scenario results</h1><p><a href="go-test.json">Full Go test log</a></p><table><tr><th>Scenario</th><th>Status</th></tr>`)
	for _, name := range leafNames(results) {
		fmt.Fprintf(&page, "<tr><td>%s</td><td>%s</td></tr>", html.EscapeString(name), html.EscapeString(results[name]))
	}
	page.WriteString("</table></html>")
	return os.WriteFile("tmp/report/tests/index.html", []byte(page.String()), 0644)
}

// leafNames returns every key with no other key nested under it (no other
// key starts with "key/"), i.e. every test with no subtest of its own.
func leafNames(results map[string]string) []string {
	names := make([]string, 0, len(results))
	for k := range results {
		names = append(names, k)
	}
	leaves := make([]string, 0, len(names))
	for _, k := range names {
		prefix := k + "/"
		hasChild := false
		for _, other := range names {
			if strings.HasPrefix(other, prefix) {
				hasChild = true
				break
			}
		}
		if !hasChild {
			leaves = append(leaves, k)
		}
	}
	sort.Strings(leaves)
	return leaves
}

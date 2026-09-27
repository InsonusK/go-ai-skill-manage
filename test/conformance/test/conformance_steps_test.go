package conformance_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/InsonusK/go-ai-skill-manage/tools/testsupport"
	"github.com/cucumber/godog"
)

// Steps talk to the CLI only through its command line and the files it
// writes, so the same steps prove any implementation of aism.

var (
	markdownLink = regexp.MustCompile(`\[([^\]]*)\]\(([^)\s]*)\)`)
	wikiLink     = regexp.MustCompile(`\[\[([^\]|]*)(?:\|([^\]]*))?\]\]`)
)

func unescape(s string) string { return strings.NewReplacer(`\n`, "\n", `\"`, `"`).Replace(s) }

func initialize(sc *godog.ScenarioContext) {
	var dir string
	var exitCode int
	var output bytes.Buffer

	sc.Before(func(ctx context.Context, s *godog.Scenario) (context.Context, error) {
		var err error
		dir, err = os.MkdirTemp("", "aism-conformance-")
		exitCode = -1
		output.Reset()
		return ctx, err
	})
	sc.After(func(ctx context.Context, s *godog.Scenario, err error) (context.Context, error) {
		return ctx, os.RemoveAll(dir)
	})
	project := func(p string) string { return filepath.Join(dir, filepath.FromSlash(p)) }

	// the project: a JSON map of path -> content.
	sc.Step(`^the project$`, func(ctx context.Context, d *godog.DocString) error {
		var files map[string]string
		if err := json.Unmarshal([]byte(d.Content), &files); err != nil {
			return err
		}
		for p, content := range files {
			if err := os.MkdirAll(filepath.Dir(project(p)), 0o755); err != nil {
				return err
			}
			if err := os.WriteFile(project(p), []byte(content), 0o644); err != nil {
				return err
			}
		}
		testsupport.Log("project=%s files=%v", dir, files)
		return nil
	})
	sc.Step(`^the project file "([^"]*)" is executable$`, func(ctx context.Context, p string) error {
		return os.Chmod(project(p), 0o755)
	})
	sc.Step(`^I remove the project path "([^"]*)"$`, func(ctx context.Context, p string) error {
		return os.RemoveAll(project(p))
	})

	sc.Step(`^I run aism "([^"]*)"$`, func(ctx context.Context, args string) error {
		cli := os.Getenv("AISM_CLI")
		cmd := exec.CommandContext(ctx, cli, strings.Fields(args)...)
		cmd.Dir = dir
		output.Reset()
		cmd.Stdout, cmd.Stderr = &output, &output
		err := cmd.Run()
		exitCode = 0
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else if err != nil {
			return err
		}
		testsupport.Log("%s %s -> exit %d\n%s", cli, args, exitCode, output.String())
		return nil
	})
	sc.Step(`^aism (succeeds|fails)$`, func(ctx context.Context, want string) error {
		if (exitCode == 0) != (want == "succeeds") {
			return fmt.Errorf("exit code %d, want aism to %s; output:\n%s", exitCode, strings.TrimSuffix(want, "s"), output.String())
		}
		return nil
	})

	// holds exactly: every file below the folder, as paths from it.
	sc.Step(`^the project folder "([^"]*)" holds exactly$`, func(ctx context.Context, p string, d *godog.DocString) error {
		got := []string{}
		root := project(p)
		err := filepath.WalkDir(root, func(cur string, e os.DirEntry, err error) error {
			if err != nil || e.IsDir() {
				return err
			}
			rel, err := filepath.Rel(root, cur)
			got = append(got, filepath.ToSlash(rel))
			return err
		})
		if err != nil {
			return err
		}
		sort.Strings(got)
		var want []string
		if err := json.Unmarshal([]byte(d.Content), &want); err != nil {
			return err
		}
		sort.Strings(want)
		return testsupport.Equal(got, want)
	})
	sc.Step(`^the project path "([^"]*)" exists$`, func(ctx context.Context, p string) error {
		_, err := os.Lstat(project(p))
		return err
	})
	sc.Step(`^the project path "([^"]*)" does not exist$`, func(ctx context.Context, p string) error {
		if _, err := os.Lstat(project(p)); err == nil {
			return fmt.Errorf("%s exists", p)
		}
		return nil
	})
	sc.Step(`^the project file "([^"]*)" (contains|does not contain) "(.*)"$`, func(ctx context.Context, p, mode, text string) error {
		raw, err := os.ReadFile(project(p))
		if err != nil {
			return err
		}
		if strings.Contains(string(raw), unescape(text)) != (mode == "contains") {
			return fmt.Errorf("%s %s %q, but it is:\n%s", p, strings.Replace(mode, "contains", "should contain", 1), unescape(text), raw)
		}
		return nil
	})
	sc.Step(`^the project file "([^"]*)" is executable in the target too$`, func(ctx context.Context, p string) error {
		info, err := os.Stat(project(p))
		if err != nil {
			return err
		}
		if info.Mode().Perm()&0o111 == 0 {
			return fmt.Errorf("%s is %v, not executable", p, info.Mode())
		}
		return nil
	})

	// linkTarget is where the link with that label in file p leads, read
	// the way a markdown viewer reads it: relative to the file's folder.
	linkTarget := func(p, label string) (string, error) {
		raw, err := os.ReadFile(project(p))
		if err != nil {
			return "", err
		}
		target := ""
		for _, m := range markdownLink.FindAllStringSubmatch(string(raw), -1) {
			if m[1] == label {
				target = m[2]
			}
		}
		for _, m := range wikiLink.FindAllStringSubmatch(string(raw), -1) {
			if m[2] == label || m[2] == "" && m[1] == label {
				target = m[1]
			}
		}
		if target == "" {
			return "", fmt.Errorf("no link %q in %s:\n%s", label, p, raw)
		}
		target, _, _ = strings.Cut(target, "#")
		if target == "" {
			return p, nil
		}
		return path.Clean(path.Join(path.Dir(p), target)), nil
	}
	sc.Step(`^in "([^"]*)" the link "([^"]*)" leads to "([^"]*)"$`, func(ctx context.Context, p, label, want string) error {
		resolved, err := linkTarget(p, label)
		if err != nil {
			return err
		}
		if resolved != path.Clean(want) {
			return fmt.Errorf("link %q in %s leads, from the file, to %s, want %s", label, p, resolved, want)
		}
		if _, err := os.Stat(project(resolved)); err != nil {
			return fmt.Errorf("link %q in %s leads to %s, which doesn't exist", label, p, resolved)
		}
		return nil
	})
	sc.Step(`^in "([^"]*)" the link "([^"]*)" leads to a file containing "(.*)"$`, func(ctx context.Context, p, label, text string) error {
		resolved, err := linkTarget(p, label)
		if err != nil {
			return err
		}
		raw, err := os.ReadFile(project(resolved))
		if err != nil {
			return fmt.Errorf("link %q in %s leads, from the file, to %s: %v", label, p, resolved, err)
		}
		if !strings.Contains(string(raw), unescape(text)) {
			return fmt.Errorf("link %q in %s leads to %s, which lacks %q", label, p, resolved, unescape(text))
		}
		return nil
	})
}

package entity_test

import (
	"io/fs"
	"testing/fstest"

	"github.com/cucumber/godog"
)

// countingFS wraps fstest.MapFS, counting Open calls per path so a test can
// prove a lazy loader reads a file at most once even across repeat calls.
type countingFS struct {
	files  fstest.MapFS
	counts map[string]int
}

func (f countingFS) Open(name string) (fs.File, error) {
	f.counts[name]++
	return f.files.Open(name)
}

func initialize(sc *godog.ScenarioContext) {
	registerFileSteps(sc)
	registerSkillSteps(sc)
	registerDocumentSteps(sc)
	registerLinkSteps(sc)
}

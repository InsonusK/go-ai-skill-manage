package conformance_test

import (
	"os"
	"testing"

	"github.com/InsonusK/go-ai-skill-manage/tools/testsupport"
)

// TestFeatures runs the shared scenarios against the CLI in AISM_CLI (the
// Go bin/aism or the Python .venv/bin/aism): make conformance /
// make conformance-python. Without it the suite is skipped, so the known
// differences between the implementations don't fail go test ./...
func TestFeatures(t *testing.T) {
	if os.Getenv("AISM_CLI") == "" {
		t.Skip("set AISM_CLI to the aism executable to run the conformance scenarios")
	}
	testsupport.Run(t, initialize)
}

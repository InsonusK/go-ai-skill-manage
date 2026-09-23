package services_test

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"testing/fstest"

	"github.com/InsonusK/go-ai-skill-manage/internal/domain/interfaces"
	"github.com/InsonusK/go-ai-skill-manage/internal/domain/model"
	"github.com/InsonusK/go-ai-skill-manage/internal/domain/services"
	"github.com/InsonusK/go-ai-skill-manage/internal/domain/services/discovery"
	"github.com/InsonusK/go-ai-skill-manage/internal/domain/services/planning"
	"github.com/InsonusK/go-ai-skill-manage/internal/domain/services/relations"
	"github.com/InsonusK/go-ai-skill-manage/internal/domain/services/sourcing"
	"github.com/InsonusK/go-ai-skill-manage/internal/infrastructure/document"
	"github.com/InsonusK/go-ai-skill-manage/tools/testsupport"
	"github.com/cucumber/godog"
)

// source is a fake interfaces.SourceProvider, wrapped in a real
// sourcing.Manager so SyncService.Sources/Lookup and Detector's own
// SkillCatalog.Manager all share one cache, matching main.go's wiring: it
// hands back a fixed repo and records the acquisition options it received.
type source struct {
	repo    *model.Repository
	tempDir *string
}

func (s source) Acquire(_ context.Context, _ model.SourceKey, options model.AcquisitionOptions) (*model.Repository, error) {
	*s.tempDir = options.TempDir
	return s.repo, nil
}

type state struct{ fail bool }

func (s state) Snapshot(ctx context.Context, p string) (map[string]model.Managed, error) {
	if s.fail && p == "/project/two" {
		return nil, fmt.Errorf("state unavailable")
	}
	return map[string]model.Managed{}, nil
}

type writer struct{ calls *int }

func (w writer) Apply(context.Context, model.TargetPlan) error { *w.calls++; return nil }

// failingDetector proves SyncService.Run surfaces a discovery-stage error
// without needing real markdown parsing -- only possible to fake now that
// Detector is a port (services.SourceSelector) rather than a concrete
// discovery.Detector field.
type failingDetector struct{ message string }

func (d failingDetector) Select(context.Context, *sourcing.SkillCatalog, model.SourceSpec) ([]*model.SkillImpl, error) {
	return nil, fmt.Errorf("%s", d.message)
}

func initialize(sc *godog.ScenarioContext) {
	architectureSteps(sc)
	var request model.Request
	var repo *model.Repository
	var calls int
	var acquiredTempDir string
	var fail bool
	var failure error
	var detectorFailure string
	sc.Step(`^sync source content "([^"]*)" and dry run "([^"]*)"$`, func(ctx context.Context, content, dry string) error {
		body := "---\nname: sample\n---\n"
		if content == "broken" {
			body += "[bad](missing)"
		}
		repo = &model.Repository{ID: "repo", Root: "/source", FS: fstest.MapFS{"a.skill.md": &fstest.MapFile{Data: []byte(body), Mode: 0644}}}
		request = model.Request{Base: "/project", Sources: []model.SourceSpec{{Type: "local"}}, Targets: []model.Target{{Name: "one", Path: "/project/out"}}, DryRun: dry == "true", RemoveOrphans: true, Conflict: "error"}
		calls = 0
		acquiredTempDir = ""
		fail = false
		detectorFailure = ""
		testsupport.Log("content=%s dry=%s", content, dry)
		return nil
	})
	sc.Step(`^skill detection always fails with "([^"]*)"$`, func(ctx context.Context, message string) error {
		detectorFailure = message
		testsupport.Log("detector failure=%s", message)
		return nil
	})
	sc.Step(`^a second target fails state loading$`, func(ctx context.Context) error {
		fail = true
		request.Targets = append(request.Targets, model.Target{Name: "two", Path: "/project/two"})
		testsupport.Log("second target fails")
		return nil
	})
	sc.Step(`^request temporary directory is "([^"]*)"$`, func(ctx context.Context, tempDir string) error {
		request.TempDir = tempDir
		testsupport.Log("request temp dir=%s", tempDir)
		return nil
	})
	sc.Step(`^I synchronize$`, func(ctx context.Context) error {
		detector := discovery.SkillSelector{Codec: document.Codec{}}
		manager := sourcing.NewManager(map[string]interfaces.SourceProvider{"local": source{repo, &acquiredTempDir}}, request.TempDir)
		service := services.SyncService{Sources: manager, Codec: document.Codec{}, Lookup: manager, Detector: detector, Relations: relations.Expander{Detector: detector}, Planner: planning.Planner{State: state{fail}, Codec: document.Codec{}}, Writer: writer{&calls}}
		if detectorFailure != "" {
			service.Detector = failingDetector{message: detectorFailure}
		}
		_, failure = service.Run(ctx, request)
		testsupport.Log("sync error=%v", failure)
		return nil
	})
	sc.Step(`^writer calls equal "([^"]*)"$`, func(ctx context.Context, w string) error {
		return testsupport.Equal(strconv.Itoa(calls), w)
	})
	sc.Step(`^source acquisition temp dir equals "([^"]*)"$`, func(ctx context.Context, want string) error {
		testsupport.Log("source acquisition temp dir=%s", acquiredTempDir)
		return testsupport.Equal(acquiredTempDir, want)
	})
	sc.Step(`^sync error contains "([^"]*)"$`, func(ctx context.Context, w string) error {
		if w == "" {
			return failure
		}
		testsupport.Log("error=%v", failure)
		if failure == nil || !strings.Contains(failure.Error(), w) {
			return fmt.Errorf("error=%v want %s", failure, w)
		}
		return nil
	})
}

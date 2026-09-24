package handler

import (
	"context"
	"log/slog"

	"github.com/InsonusK/go-ai-skill-manage/internal/domain/model"
	"github.com/InsonusK/go-ai-skill-manage/internal/domain/services/sourcing"
)

// SyncService synchronizes the skills req's sources select into its
// targets. Only its first part exists yet: loading and checking the skills
// (FetchAndValidateSkills). Building the target catalog, planning and
// writing are being rebuilt; the previous pipeline is in git history.
type SyncService struct {
	Sources *sourcing.Manager
}

// Run loads and checks the skills and returns every problem found, without
// going further. With no problems it panics: the rest of the
// synchronization is not implemented yet.
func (s SyncService) Run(ctx context.Context, req model.Request) (model.Result, error) {
	slog.InfoContext(ctx, "synchronization started", "sources", len(req.Sources), "targets", len(req.Targets), "dry_run", req.DryRun)
	result := model.Result{DryRun: req.DryRun, Skills: []string{}, Plans: []model.TargetPlan{}}
	if _, problems := FetchAndValidateSkills(ctx, s.Sources, req); len(problems) > 0 {
		return result, problems
	}
	panic("sync: not implemented: target catalog, planning and writing")
}

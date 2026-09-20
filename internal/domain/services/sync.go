package services

import (
	"context"
	"errors"
	"github.com/InsonusK/go-ai-skill-manage/internal/domain/interfaces"
	"github.com/InsonusK/go-ai-skill-manage/internal/domain/model"
	"github.com/InsonusK/go-ai-skill-manage/internal/domain/services/discovery"
	"github.com/InsonusK/go-ai-skill-manage/internal/domain/services/planning"
	"github.com/InsonusK/go-ai-skill-manage/internal/domain/services/relations"
	"log/slog"
)

type SyncService struct {
	Sources   interfaces.SourceProvider
	Detector  discovery.Detector
	Relations relations.Expander
	Planner   planning.Planner
	Writer    interfaces.PlanWriter
}

func (s SyncService) Run(ctx context.Context, req model.Request) (result model.Result, err error) {
	slog.InfoContext(ctx, "synchronization started", "sources", len(req.Sources), "targets", len(req.Targets), "dry_run", req.DryRun)
	result.DryRun = req.DryRun
	result.Skills = []string{}
	result.Plans = []model.TargetPlan{}
	catalog := &discovery.Catalog{Conflict: req.Conflict}
	var cleanups []func() error
	defer func() {
		for i := len(cleanups) - 1; i >= 0; i-- {
			err = errors.Join(err, cleanups[i]())
		}
	}()
	var issues model.Issues
	for _, spec := range req.Sources {
		slog.DebugContext(ctx, "acquiring source", "type", spec.Type, "path", spec.Path)
		repo, close, acquireErr := s.Sources.Acquire(ctx, spec, model.AcquisitionOptions{TempDir: req.TempDir})
		if acquireErr != nil {
			return result, acquireErr
		}
		cleanups = append(cleanups, close)
		found, discoverErr := s.Detector.Select(ctx, repo)
		if discoverErr != nil {
			issues = append(issues, model.Issue{Code: "discovery", File: spec.Path, Message: discoverErr.Error()})
		}
		for _, skill := range found {
			if addErr := catalog.Add(ctx, skill); addErr != nil {
				issues = append(issues, model.Issue{Code: "duplicate-name", Skill: skill.Name, Message: addErr.Error()})
			}
		}
	}
	if len(issues) > 0 {
		return result, issues
	}
	if err = s.Relations.Expand(ctx, catalog, req.AddRelations, req.LinkSkipFolders); err != nil {
		return result, err
	}
	slog.DebugContext(ctx, "catalog expanded", "skills", len(catalog.Skills))
	for _, skill := range catalog.Skills {
		result.Skills = append(result.Skills, skill.Name)
	}
	for _, target := range req.Targets {
		plan, planErr := s.Planner.Plan(ctx, catalog, req, target)
		if planErr != nil {
			return result, planErr
		}
		result.Plans = append(result.Plans, plan)
	}
	slog.InfoContext(ctx, "plans validated", "skills", len(result.Skills), "targets", len(result.Plans))
	if req.DryRun {
		return result, nil
	}
	for _, plan := range result.Plans {
		slog.DebugContext(ctx, "applying target", "path", plan.Target.Path, "operations", len(plan.Operations))
		if err = s.Writer.Apply(ctx, plan); err != nil {
			return result, err
		}
	}
	return result, nil
}

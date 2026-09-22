package services

import (
	"context"
	"github.com/InsonusK/go-ai-skill-manage/internal/domain/interfaces"
	"github.com/InsonusK/go-ai-skill-manage/internal/domain/model"
	"github.com/InsonusK/go-ai-skill-manage/internal/domain/services/planning"
	"github.com/InsonusK/go-ai-skill-manage/internal/domain/services/relations"
	"github.com/InsonusK/go-ai-skill-manage/internal/domain/services/sourcing"
	"log/slog"
)

// SourceSelector selects the skills one source contributes: it resolves
// spec's configured subpaths through catalog (validating/adding each via
// GetOrAddByPath) and filters the result by spec's tags/name --
// implemented by discovery.Detector. Lives here rather than in
// interfaces because its signature needs *sourcing.SkillCatalog, and
// sourcing already imports interfaces (for SourceProvider/DocumentCodec),
// so interfaces importing sourcing back would cycle.
type SourceSelector interface {
	Select(ctx context.Context, catalog *sourcing.SkillCatalog, spec model.SourceSpec) ([]*model.Skill, error)
}

type SyncService struct {
	Sources   *sourcing.Manager
	Codec     interfaces.DocumentCodec
	Lookup    interfaces.RepositoryLookup
	Detector  SourceSelector
	Relations relations.Expander
	Planner   planning.Planner
	Writer    interfaces.PlanWriter
}

func (s SyncService) Run(ctx context.Context, req model.Request) (result model.Result, err error) {
	slog.InfoContext(ctx, "synchronization started", "sources", len(req.Sources), "targets", len(req.Targets), "dry_run", req.DryRun)
	result.DryRun = req.DryRun
	result.Skills = []string{}
	result.Plans = []model.TargetPlan{}
	catalog := &model.SkillCatalog{Conflict: req.Conflict}
	var issues model.Issues
	for _, spec := range req.Sources {
		slog.DebugContext(ctx, "acquiring source", "type", spec.Type, "path", spec.Path)
		skillCatalog := &sourcing.SkillCatalog{Manager: s.Sources, Codec: s.Codec, Conflict: req.Conflict, SkipFolders: sourcing.DefaultSkipFolders}
		found, discoverErr := s.Detector.Select(ctx, skillCatalog, spec)
		if discoverErr != nil {
			issues = append(issues, model.Issue{Code: "discovery", File: spec.Path, Message: discoverErr.Error()})
		}
		for _, skill := range found {
			if addErr := catalog.GetOrAdd(ctx, skill); addErr != nil {
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
		plan, planErr := s.Planner.Plan(ctx, catalog, s.Lookup, req, target)
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

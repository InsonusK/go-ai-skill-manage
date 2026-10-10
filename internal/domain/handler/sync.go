package handler

import (
	"context"
	"fmt"
	"log/slog"
	"slices"

	"github.com/InsonusK/go-ai-skill-manager/internal/domain/entity"
	"github.com/InsonusK/go-ai-skill-manager/internal/domain/interfaces"
	"github.com/InsonusK/go-ai-skill-manager/internal/domain/model"
	"github.com/InsonusK/go-ai-skill-manager/internal/domain/model/issues"
	"github.com/InsonusK/go-ai-skill-manager/internal/domain/services/planning"
	"github.com/InsonusK/go-ai-skill-manager/internal/domain/services/sourcing"
	"github.com/InsonusK/go-ai-skill-manager/internal/domain/services/transform"
	"github.com/InsonusK/go-ai-skill-manager/internal/domain/services/transform/transformers"
)

// ClaudeAdapter is the target adapter that adds Claude Code's
// transformers (ClaudeWhenToUseTransformer).
const ClaudeAdapter = "claude-property-adapter"

// SyncService synchronizes the skills req's sources select into its
// targets.
type SyncService struct {
	Sources *sourcing.Manager
	State   interfaces.StateReader
	Writer  interfaces.PlanWriter
}

// SyncResult is what a sync did, or, for a dry run, would do.
type SyncResult struct {
	// Skills are the names of the synchronized skills, in load order.
	Skills []string
	// Plans are the planned changes, one per target, in req.Targets order.
	Plans  []entity.TargetPlan
	DryRun bool
	// Warnings are the problems that didn't stop the sync, each once.
	Warnings issues.SkillIssues
}

// Run synchronizes req (which passed config/validator.Validate):
//  1. FetchAndValidateSkills -- skill errors stop it here, returned with
//     the warnings as issues.SkillIssues; warnings alone go on into
//     SyncResult.Warnings, as the transformers' warnings do;
//  2. a base TargetSkillCatalog of the loaded skills, laid out by
//     FlatTransformer -- the part every target shares, done once;
//  3. per target, a clone of the base transformed by the target's
//     transformers (ClaudeWhenToUseTransformer with ClaudeAdapter, then
//     always ManagedMarkerTransformer), then planned against a snapshot of
//     the target folder (planning.Plan);
//  4. only if every target planned without problems -- they are returned
//     together as issues.TargetIssues -- and it isn't a dry run, the plans
//     are written, in target order.
//
// So a problem anywhere leaves every target untouched; a failure while
// writing (the disk) can still stop after some targets are written.
func (s SyncService) Run(ctx context.Context, req model.Request) (result SyncResult, err error) {
	slog.InfoContext(ctx, "synchronization started", "sources", len(req.Sources), "targets", len(req.Targets), "dry_run", req.DryRun)
	result = SyncResult{DryRun: req.DryRun, Skills: []string{}, Plans: []entity.TargetPlan{}}
	catalog, problems := FetchAndValidateSkills(ctx, s.Sources, req)
	if issues.HasErrors(problems) {
		return result, problems
	}
	warnings := &warningSet{seen: map[string]bool{}}
	defer func() { result.Warnings = warnings.list }()
	warnings.add(problems)
	base := entity.NewTargetSkillCatalog(catalog.Skills())
	found, err := mustPipeline(transformers.FlatTransformer{Catalog: catalog}).Run(ctx, base)
	warnings.add(found)
	if err != nil {
		return result, err
	}
	for _, skill := range base.Skills() {
		result.Skills = append(result.Skills, skill.Name())
	}
	var targetProblems issues.TargetIssues
	for _, target := range req.Targets {
		plan, found, err := s.planTarget(ctx, base, target, req.RemoveOrphans, warnings)
		if err != nil {
			return result, err
		}
		result.Plans = append(result.Plans, plan)
		targetProblems = append(targetProblems, found...)
	}
	if len(targetProblems) > 0 {
		return result, targetProblems
	}
	slog.InfoContext(ctx, "plans ready", "skills", len(result.Skills), "targets", len(result.Plans))
	if req.DryRun {
		return result, nil
	}
	for _, plan := range result.Plans {
		slog.DebugContext(ctx, "writing target", "path", plan.Target.Path, "operations", len(plan.Operations))
		if err := s.Writer.Apply(ctx, plan); err != nil {
			return result, err
		}
	}
	return result, nil
}

// planTarget transforms a clone of base for target and plans it; the
// transformers' warnings go into warnings.
func (s SyncService) planTarget(ctx context.Context, base *entity.TargetSkillCatalog, target model.Target, removeOrphans bool, warnings *warningSet) (entity.TargetPlan, issues.TargetIssues, error) {
	catalog := base.Clone()
	list := []transform.Transformer{}
	if slices.Contains(target.Adapters, ClaudeAdapter) {
		list = append(list, transformers.ClaudeWhenToUseTransformer{})
	}
	list = append(list, transformers.ManagedMarkerTransformer{})
	found, err := mustPipeline(list...).Run(ctx, catalog)
	warnings.add(found)
	if err != nil {
		return entity.TargetPlan{}, nil, err
	}
	state, err := s.State.Snapshot(ctx, target.Path)
	if err != nil {
		return entity.TargetPlan{}, nil, err
	}
	plan, problems := planning.Plan(target, catalog, state, removeOrphans)
	return plan, problems, nil
}

// warningSet collects warnings, each once: every target's transformers
// warn about the same skill.
type warningSet struct {
	seen map[string]bool
	list issues.SkillIssues
}

func (w *warningSet) add(found issues.SkillIssues) {
	for _, i := range found {
		key := fmt.Sprintf("%#v", i)
		if !w.seen[key] {
			w.seen[key] = true
			w.list = append(w.list, i)
		}
	}
}

func mustPipeline(list ...transform.Transformer) *transform.Pipeline {
	p, err := transform.NewPipeline(list...)
	if err != nil {
		// The lists above are code, not configuration.
		panic(err)
	}
	return p
}

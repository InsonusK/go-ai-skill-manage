package command

import (
	"context"
	"fmt"
	"io"
	"log/slog"

	"github.com/InsonusK/go-ai-skill-manage/internal/command/common"
	"github.com/InsonusK/go-ai-skill-manage/internal/config"
	"github.com/InsonusK/go-ai-skill-manage/internal/domain/handler"
	"github.com/InsonusK/go-ai-skill-manage/internal/domain/services/sourcing"
)

// Sync is the sync command: load the skills of the sources, check them
// and write them into every target.
type Sync struct {
	Source   common.Source
	Override config.Overrides
	// Force is the deprecated --force: without a hash to skip unchanged
	// skills every managed folder is rewritten anyway.
	Force bool
}

func (s *Sync) Name() string { return "sync" }
func (s *Sync) Summary() string {
	return "Load the skills of the configured sources, check them and write\nthem into every target folder"
}

func (s *Sync) flags() []common.Flag {
	o := &s.Override
	return append(s.Source.Flags(),
		common.String(&o.Target, "PATH", "Write into this folder instead of the configured targets", "--target"),
		common.Bool(&o.DryRun, "Plan the changes and print them without writing", "--dry-run"),
		common.BoolFunc("Remove managed skill folders no longer synchronized", func(v bool) {
			if v {
				o.RemoveOrphans = &v
			}
		}, "--remove-orphans"),
		common.BoolFunc("Keep them (--remove-orphans wins over it)", func(v bool) {
			if v && (o.RemoveOrphans == nil || !*o.RemoveOrphans) {
				keep := false
				o.RemoveOrphans = &keep
			}
		}, "--keep-orphans"),
		common.AddRelationsFlag(o),
		common.Bool(&s.Force, "Deprecated, has no effect: every managed skill\nfolder is rewritten on each sync", "-f", "--force"),
	)
}

func (s *Sync) Parse(args []string, global []common.Flag) error {
	rest, err := common.ParseFlags(args, append(s.flags(), global...), "sync")
	if err != nil {
		return err
	}
	if len(rest) > 0 {
		return fmt.Errorf("unexpected argument %q", rest[0])
	}
	return nil
}

func (s *Sync) Help(global []common.Flag) string {
	return common.Help{
		Usage: []string{"aism sync [options]"},
		Description: "Load the skills of the configured sources, check them and write them into\n" +
			"every target folder. Nothing is written when the configuration, a skill or\n" +
			"a target has a problem: the problems are printed as a tree, exit code 1.",
		Flags:  s.flags(),
		Global: global,
	}.String()
}

func (s *Sync) Run(ctx context.Context, app *common.App, cwd string) (code int) {
	if s.Force {
		slog.WarnContext(ctx, "deprecated flag, remove it: every managed skill folder is rewritten on each sync", "flag", "--force")
	}
	req, ok := app.LoadRequest(ctx, s.Source, s.Override, cwd)
	if !ok {
		return 1
	}
	sources := sourcing.NewManager(app.Providers, req.TempDir)
	defer closeSources(ctx, app, sources, &code)
	result, err := handler.SyncService{Sources: sources, State: app.State, Writer: app.Writer}.Run(ctx, req)
	if err != nil {
		if rows := common.Reportables(err); rows != nil {
			common.PrintIssues(app.Err, rows, app.Color)
		} else {
			fmt.Fprintln(app.Err, err)
		}
		return 1
	}
	PrintResult(app.Out, result)
	return 0
}

// closeSources releases the acquired repositories; a failure fails the
// command.
func closeSources(ctx context.Context, app *common.App, sources *sourcing.Manager, code *int) {
	if err := sources.Close(ctx); err != nil {
		fmt.Fprintln(app.Err, "close sources:", err)
		*code = 1
	}
}

// PrintResult prints what a sync did (or, for a dry run, would do): each
// target with its operations, then a summary.
//
// Пример:
//
//	Target claude: /p/.claude/skills
//	  create review
//	  update guide
//	Synced 2 skill(s) to 1 target(s)
func PrintResult(out io.Writer, result handler.SyncResult) {
	for _, plan := range result.Plans {
		fmt.Fprintf(out, "Target %s: %s\n", plan.Target.Name, plan.Target.Path)
		for _, op := range plan.Operations {
			fmt.Fprintf(out, "  %s %s\n", op.Action, op.Name)
		}
	}
	if result.DryRun {
		fmt.Fprintf(out, "Dry run: %d skill(s), %d target(s); nothing written\n", len(result.Skills), len(result.Plans))
		return
	}
	fmt.Fprintf(out, "Synced %d skill(s) to %d target(s)\n", len(result.Skills), len(result.Plans))
}

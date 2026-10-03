package command

import (
	"context"
	"fmt"

	"github.com/InsonusK/go-ai-skill-manage/internal/command/common"
	"github.com/InsonusK/go-ai-skill-manage/internal/config"
	"github.com/InsonusK/go-ai-skill-manage/internal/domain/handler"
	"github.com/InsonusK/go-ai-skill-manage/internal/domain/services/sourcing"
)

// Validate is the validate command: check the configuration and the
// skills, write nothing.
type Validate struct {
	Source   common.Source
	Override config.Overrides
}

func (v *Validate) Name() string { return "validate" }
func (v *Validate) Summary() string {
	return "Check the configuration and the skills without writing anything"
}

func (v *Validate) flags() []common.Flag {
	return append(v.Source.Flags(), common.AddRelationsFlag(&v.Override))
}

func (v *Validate) Parse(args []string, global []common.Flag) error {
	rest, err := common.ParseFlags(args, append(v.flags(), global...), "validate")
	if err != nil {
		return err
	}
	if len(rest) > 0 {
		return fmt.Errorf("unexpected argument %q", rest[0])
	}
	return nil
}

func (v *Validate) Help(global []common.Flag) string {
	return common.Help{
		Usage: []string{"aism validate [options]"},
		Description: "Check the configuration and the skills of the sources, as sync does, without\n" +
			"writing anything. Problems are printed as a tree, exit code 1.",
		Flags:  v.flags(),
		Global: global,
	}.String()
}

func (v *Validate) Run(ctx context.Context, app *common.App, cwd string) (code int) {
	req, ok := app.LoadRequest(ctx, v.Source, v.Override, cwd)
	if !ok {
		return 1
	}
	sources := sourcing.NewManager(app.Providers, req.TempDir)
	defer closeSources(ctx, app, sources, &code)
	catalog, problems := handler.FetchAndValidateSkills(ctx, sources, req)
	if len(problems) > 0 {
		common.PrintIssues(app.Err, common.Reportables(problems), app.Color)
		return 1
	}
	fmt.Fprintf(app.Out, "Checked %d skill(s): no problems\n", len(catalog.Skills()))
	return 0
}

package command

import (
	"fmt"
	"github.com/InsonusK/go-ai-skill-manage/internal/domain/model"
	"io"
)

func PrintResult(out io.Writer, result model.Result) {
	for _, name := range result.Skills {
		fmt.Fprintf(out, "  %s\n", name)
	}
	for _, plan := range result.Plans {
		fmt.Fprintf(out, "Target %s: %s\n", plan.Target.Name, plan.Target.Path)
		for _, op := range plan.Operations {
			fmt.Fprintf(out, "  %-6s %s (%s)\n", op.Action, op.Name, op.Reason)
		}
	}
	if result.DryRun {
		fmt.Fprintf(out, "Dry run: %d skill(s), %d target(s); no target changes\n", len(result.Skills), len(result.Plans))
	} else {
		fmt.Fprintf(out, "Synced %d skill(s) to %d target(s)\n", len(result.Skills), len(result.Plans))
	}
}

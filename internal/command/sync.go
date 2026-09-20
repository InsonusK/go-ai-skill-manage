package command

import (
	"context"
	"fmt"
	"github.com/InsonusK/go-ai-skill-manage/internal/domain/services"
	"io"
)

type App struct {
	Sync     *services.SyncService
	ReadFile func(string) ([]byte, error)
	Out, Err io.Writer
	Version  string
}

func (a App) Execute(ctx context.Context, opts Options, cwd string) int {
	if opts.Help {
		fmt.Fprint(a.Out, Usage)
		return 0
	}
	if opts.Version {
		fmt.Fprintln(a.Out, a.Version)
		return 0
	}
	req, err := a.request(opts, cwd)
	if err != nil {
		fmt.Fprintln(a.Err, err)
		return 1
	}
	result, err := a.Sync.Run(ctx, req)
	if err != nil {
		fmt.Fprintln(a.Err, err)
		return 1
	}
	PrintResult(a.Out, result)
	return 0
}

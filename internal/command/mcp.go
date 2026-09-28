package command

import (
	"context"
	"errors"
	"fmt"
	"strings"

	configvalidator "github.com/InsonusK/go-ai-skill-manage/internal/config/validator"
	"github.com/InsonusK/go-ai-skill-manage/internal/mcpserver"
)

// serveMCP runs the MCP server on a.MCPTransport until the client leaves.
// Every tool call reads the configuration anew (Request, then Validate),
// so a changed config applies without restarting the server; stdout
// belongs to the protocol, so nothing else is printed there.
func (a App) serveMCP(ctx context.Context, opts Options, cwd string) int {
	open := func(ctx context.Context) (mcpserver.Project, error) {
		req, err := a.Request(opts, cwd)
		if err != nil {
			return mcpserver.Project{}, err
		}
		if problems := configvalidator.Validate(ctx, req); len(problems) > 0 {
			var b strings.Builder
			PrintIssues(&b, reportables(problems), false)
			return mcpserver.Project{}, errors.New("configuration problems:\n" + b.String())
		}
		service, dir := a.feedbackService(req)
		return mcpserver.Project{Service: service, Targets: req.Targets, DraftsDir: dir}, nil
	}
	if err := mcpserver.NewServer(open, a.Version).Run(ctx, a.MCPTransport); err != nil && ctx.Err() == nil {
		fmt.Fprintln(a.Err, "mcp:", err)
		return 1
	}
	return 0
}

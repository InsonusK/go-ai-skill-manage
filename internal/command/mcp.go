package command

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/InsonusK/go-ai-skill-manage/internal/command/common"
	"github.com/InsonusK/go-ai-skill-manage/internal/config"
	configvalidator "github.com/InsonusK/go-ai-skill-manage/internal/config/validator"
	"github.com/InsonusK/go-ai-skill-manage/internal/mcpserver"
)

// DefaultMCPServerName is the server's name in .mcp.json.
const DefaultMCPServerName = "ai-skills"

// MCP is the mcp command: Action "" serves the feedback tools over stdio,
// install/uninstall edit .mcp.json. ServerName is the server's key in .mcp.json;
// Replace lets install overwrite another entry under that name. Neither
// reaches the server.
type MCP struct {
	Action, ServerName string
	Replace            bool
	Source             common.Source
}

// mcpActions: name, usage after "aism mcp", summary.
var mcpActions = [][3]string{
	{"install", "install [-c FILE] [--name ai-skills] [--replace]", "Add this server to .mcp.json next to the config (Claude Code);\n-c, when given, is passed to the server too"},
	{"uninstall", "uninstall [-c FILE] [--name ai-skills]", "Remove it from .mcp.json"},
}

func (m *MCP) Name() string { return "mcp" }
func (m *MCP) Summary() string {
	return "Serve the feedback tools to an agent over MCP (stdio): the\nagent drafts, the user confirms in the client's dialog"
}

func (m *MCP) flags() []common.Flag {
	flags := []common.Flag{m.Source.ConfigFlag()}
	if m.Action != "" {
		flags = append(flags, common.String(&m.ServerName, "NAME", "The server's key in mcpServers (default: "+DefaultMCPServerName+")", "--name"))
	}
	if m.Action == "install" {
		flags = append(flags, common.Bool(&m.Replace, "Overwrite another server under this name", "--replace"))
	}
	return flags
}

// Parse: the action, if any, comes first, then its flags.
func (m *MCP) Parse(args []string, global []common.Flag) error {
	m.ServerName = DefaultMCPServerName
	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		if !slices.ContainsFunc(mcpActions, func(a [3]string) bool { return a[0] == args[0] }) {
			return fmt.Errorf("unknown mcp action %q: use install, uninstall, or none to serve", args[0])
		}
		m.Action, args = args[0], args[1:]
	}
	rest, err := common.ParseFlags(args, append(m.flags(), global...), strings.TrimSpace("mcp "+m.Action))
	if err != nil {
		return err
	}
	if len(rest) > 0 {
		return fmt.Errorf("unexpected argument %q", rest[0])
	}
	return nil
}

func (m *MCP) Help(global []common.Flag) string {
	i := slices.IndexFunc(mcpActions, func(a [3]string) bool { return a[0] == m.Action })
	if i < 0 {
		h := common.Help{
			Usage: []string{"aism mcp [-c FILE]", "aism mcp <action> [options]"},
			Description: "Without an action: serve the tools feedback_draft and feedback_submit over\n" +
				"stdio. The config is read anew on every call, from CLAUDE_PROJECT_DIR when\n" +
				"it is set; stdout belongs to the protocol.",
			EntriesTitle: "Actions",
			Flags:        m.flags(),
			Global:       global,
			Footer:       "Run \"aism mcp <action> --help\" for an action's options.",
		}
		for _, a := range mcpActions {
			h.Entries = append(h.Entries, [2]string{a[0], a[2]})
		}
		return h.String()
	}
	return common.Help{
		Usage:       []string{"aism mcp " + mcpActions[i][1]},
		Description: mcpActions[i][2] + ".",
		Flags:       m.flags(),
		Global:      global,
	}.String()
}

func (m *MCP) Run(ctx context.Context, app *common.App, cwd string) int {
	switch m.Action {
	case "install":
		return m.install(app, cwd)
	case "uninstall":
		return m.uninstall(app, cwd)
	}
	if dir := app.Env("CLAUDE_PROJECT_DIR"); dir != "" {
		// Claude Code gives the MCP server the project's root: the config
		// (and so the drafts) are found from there, whatever the server's
		// working directory.
		cwd = dir
	}
	return m.serve(ctx, app, cwd)
}

// serve runs the MCP server on app.MCPTransport until the client leaves.
// Every tool call reads the configuration anew (Request, then Validate),
// so a changed config applies without restarting the server; stdout
// belongs to the protocol, so nothing else is printed there.
func (m *MCP) serve(ctx context.Context, app *common.App, cwd string) int {
	open := func(ctx context.Context) (mcpserver.Project, error) {
		req, warnings, err := app.Request(m.Source, config.Overrides{}, cwd)
		if err != nil {
			return mcpserver.Project{}, err
		}
		app.PrintWarnings(common.Rows(warnings))
		if problems := configvalidator.Validate(ctx, req); len(problems) > 0 {
			var b strings.Builder
			common.PrintIssues(&b, common.Reportables(problems), false)
			return mcpserver.Project{}, errors.New("configuration problems:\n" + b.String())
		}
		service, dir := app.FeedbackService(req)
		return mcpserver.Project{Service: service, Targets: req.Targets, DraftsDir: dir}, nil
	}
	if err := mcpserver.NewServer(open, app.Version).Run(ctx, app.MCPTransport); err != nil && ctx.Err() == nil {
		fmt.Fprintln(app.Err, "mcp:", err)
		return 1
	}
	return 0
}

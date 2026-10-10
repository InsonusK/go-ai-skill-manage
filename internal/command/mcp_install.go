package command

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"
	"reflect"
	"strings"

	"github.com/InsonusK/go-ai-skill-manager/internal/command/common"
	"github.com/InsonusK/go-ai-skill-manager/internal/config"
)

// MCPConfigFile is the file Claude Code reads the project's MCP servers
// from, at the project's root.
const MCPConfigFile = ".mcp.json"

// mcpEntry is this server's entry in .mcp.json.
type mcpEntry struct {
	Command string   `json:"command"`
	Args    []string `json:"args"`
}

// install adds this server to .mcp.json in the project's root (the
// config's folder), keeping whatever else the file holds. The command is
// the name the tool runs under when it is found by that name in PATH (the
// file is committed: another machine has another absolute path), else the
// absolute path. A config other than ai-skills.yaml goes into the args, so
// the server reads the same one.
//
// Пример (aism в PATH, -c cfg.yaml) -> .mcp.json:
//
//	{"mcpServers": {"ai-skills": {"command": "aism", "args": ["mcp", "-c", "cfg.yaml"]}}}
func (m *MCP) install(app *common.App, cwd string) int {
	file, err := m.configFile(app, cwd)
	if err != nil {
		fmt.Fprintln(app.Err, err)
		return 1
	}
	command, inPath := app.Self()
	entry := mcpEntry{Command: command, Args: []string{"mcp"}}
	if name := filepath.Base(m.Source.Config); m.Source.Config != "" && name != common.DefaultConfigFile {
		entry.Args = append(entry.Args, "-c", name)
	}
	root, servers, err := readMCPConfig(app, file)
	if err != nil {
		fmt.Fprintln(app.Err, err)
		return 1
	}
	name := m.ServerName
	if old, ok := servers[name]; ok {
		if sameJSON(old, entry) {
			fmt.Fprintf(app.Out, "MCP server %s is already in %s\n", name, file)
			return 0
		}
		if !m.Replace {
			fmt.Fprintf(app.Err, "%s already has another MCP server %s: %s\nwould write: %s\nrun with --replace to overwrite it, or pick another --name\n", file, name, compact(old), compact(mustJSON(entry)))
			return 1
		}
	}
	servers[name] = mustJSON(entry)
	if err := writeMCPConfig(app, file, root, servers); err != nil {
		fmt.Fprintln(app.Err, err)
		return 1
	}
	fmt.Fprintf(app.Out, "Added MCP server %s to %s: %s %s\n", name, file, entry.Command, strings.Join(entry.Args, " "))
	if !inPath {
		fmt.Fprintf(app.Out, "Warning: %s is an absolute path of this machine: put the tool in PATH and run install again, or don't commit this entry\n", entry.Command)
	}
	fmt.Fprintf(app.Out, "Commit %s. Claude Code asks to approve the server %s the first time you run claude in this project.\n", MCPConfigFile, name)
	return 0
}

// uninstall removes the server named m.ServerName from .mcp.json;
// no file or no such server: nothing to do.
func (m *MCP) uninstall(app *common.App, cwd string) int {
	file, err := m.configFile(app, cwd)
	if err != nil {
		fmt.Fprintln(app.Err, err)
		return 1
	}
	root, servers, err := readMCPConfig(app, file)
	if err != nil {
		fmt.Fprintln(app.Err, err)
		return 1
	}
	name := m.ServerName
	if _, ok := servers[name]; !ok {
		fmt.Fprintf(app.Out, "No MCP server %s in %s: nothing to remove\n", name, file)
		return 0
	}
	delete(servers, name)
	if err := writeMCPConfig(app, file, root, servers); err != nil {
		fmt.Fprintln(app.Err, err)
		return 1
	}
	fmt.Fprintf(app.Out, "Removed MCP server %s from %s\n", name, file)
	return 0
}

// configFile is .mcp.json in the project's root: the config's folder.
func (m *MCP) configFile(app *common.App, cwd string) (string, error) {
	req, warnings, err := app.Request(m.Source, config.Overrides{}, cwd)
	if err != nil {
		return "", err
	}
	app.PrintWarnings(common.Rows(warnings))
	return filepath.Join(req.Base, MCPConfigFile), nil
}

// readMCPConfig reads file's top-level keys and its mcpServers by name;
// a missing file holds nothing.
func readMCPConfig(app *common.App, file string) (map[string]json.RawMessage, map[string]json.RawMessage, error) {
	root, servers := map[string]json.RawMessage{}, map[string]json.RawMessage{}
	raw, err := app.ReadFile(file)
	if errors.Is(err, fs.ErrNotExist) {
		return root, servers, nil
	}
	if err != nil {
		return nil, nil, err
	}
	if err := json.Unmarshal(raw, &root); err != nil {
		return nil, nil, fmt.Errorf("%s isn't a JSON object (%w): fix it by hand", file, err)
	}
	if list, ok := root["mcpServers"]; ok {
		if err := json.Unmarshal(list, &servers); err != nil || servers == nil {
			return nil, nil, fmt.Errorf("%s: mcpServers isn't an object: fix it by hand", file)
		}
	}
	return root, servers, nil
}

// writeMCPConfig writes root with servers as mcpServers, indented by two
// spaces. encoding/json sorts the keys and drops the file's own layout.
func writeMCPConfig(app *common.App, file string, root, servers map[string]json.RawMessage) error {
	root["mcpServers"] = mustJSON(servers)
	out, err := json.MarshalIndent(root, "", "  ")
	if err != nil {
		return err
	}
	return app.WriteFile(file, append(out, '\n'), 0o644)
}

func sameJSON(raw json.RawMessage, v any) bool {
	var a, b any
	if json.Unmarshal(raw, &a) != nil || json.Unmarshal(mustJSON(v), &b) != nil {
		return false
	}
	return reflect.DeepEqual(a, b)
}

func compact(raw json.RawMessage) string {
	var b bytes.Buffer
	if json.Compact(&b, raw) != nil {
		return string(raw)
	}
	return b.String()
}

func mustJSON(v any) json.RawMessage {
	raw, err := json.Marshal(v)
	if err != nil {
		// Strings, lists and raw JSON already read: can't fail.
		panic(err)
	}
	return raw
}

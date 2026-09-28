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
)

// MCPConfigFile is the file Claude Code reads the project's MCP servers
// from, at the project's root.
const MCPConfigFile = ".mcp.json"

// mcpEntry is this server's entry in .mcp.json.
type mcpEntry struct {
	Command string   `json:"command"`
	Args    []string `json:"args"`
}

// installMCP adds this server to .mcp.json in the project's root (the
// config's folder), keeping whatever else the file holds. The command is
// the name the tool runs under when it is found by that name in PATH (the
// file is committed: another machine has another absolute path), else the
// absolute path. A config other than ai-skills.yaml goes into the args, so
// the server reads the same one.
//
// Пример (aism в PATH, -c cfg.yaml) -> .mcp.json:
//
//	{"mcpServers": {"ai-skills": {"command": "aism", "args": ["mcp", "-c", "cfg.yaml"]}}}
func (a App) installMCP(opts Options, cwd string) int {
	file, err := a.mcpConfigFile(opts, cwd)
	if err != nil {
		fmt.Fprintln(a.Err, err)
		return 1
	}
	command, inPath := a.Self()
	entry := mcpEntry{Command: command, Args: []string{"mcp"}}
	if name := filepath.Base(opts.Config); opts.Config != "" && name != DefaultConfigFile {
		entry.Args = append(entry.Args, "-c", name)
	}
	root, servers, err := a.readMCPConfig(file)
	if err != nil {
		fmt.Fprintln(a.Err, err)
		return 1
	}
	name := opts.MCP.Name
	if old, ok := servers[name]; ok {
		if sameJSON(old, entry) {
			fmt.Fprintf(a.Out, "MCP server %s is already in %s\n", name, file)
			return 0
		}
		if !opts.MCP.Replace {
			fmt.Fprintf(a.Err, "%s already has another MCP server %s: %s\nwould write: %s\nrun with --replace to overwrite it, or pick another --name\n", file, name, compact(old), compact(mustJSON(entry)))
			return 1
		}
	}
	servers[name] = mustJSON(entry)
	if err := a.writeMCPConfig(file, root, servers); err != nil {
		fmt.Fprintln(a.Err, err)
		return 1
	}
	fmt.Fprintf(a.Out, "Added MCP server %s to %s: %s %s\n", name, file, entry.Command, strings.Join(entry.Args, " "))
	if !inPath {
		fmt.Fprintf(a.Out, "Warning: %s is an absolute path of this machine: put the tool in PATH and run install again, or don't commit this entry\n", entry.Command)
	}
	fmt.Fprintf(a.Out, "Commit %s. Claude Code asks to approve the server %s the first time you run claude in this project.\n", MCPConfigFile, name)
	return 0
}

// uninstallMCP removes the server named opts.MCP.Name from .mcp.json;
// no file or no such server: nothing to do.
func (a App) uninstallMCP(opts Options, cwd string) int {
	file, err := a.mcpConfigFile(opts, cwd)
	if err != nil {
		fmt.Fprintln(a.Err, err)
		return 1
	}
	root, servers, err := a.readMCPConfig(file)
	if err != nil {
		fmt.Fprintln(a.Err, err)
		return 1
	}
	name := opts.MCP.Name
	if _, ok := servers[name]; !ok {
		fmt.Fprintf(a.Out, "No MCP server %s in %s: nothing to remove\n", name, file)
		return 0
	}
	delete(servers, name)
	if err := a.writeMCPConfig(file, root, servers); err != nil {
		fmt.Fprintln(a.Err, err)
		return 1
	}
	fmt.Fprintf(a.Out, "Removed MCP server %s from %s\n", name, file)
	return 0
}

// mcpConfigFile is .mcp.json in the project's root: the config's folder.
func (a App) mcpConfigFile(opts Options, cwd string) (string, error) {
	req, err := a.Request(opts, cwd)
	if err != nil {
		return "", err
	}
	return filepath.Join(req.Base, MCPConfigFile), nil
}

// readMCPConfig reads file's top-level keys and its mcpServers by name;
// a missing file holds nothing.
func (a App) readMCPConfig(file string) (map[string]json.RawMessage, map[string]json.RawMessage, error) {
	root, servers := map[string]json.RawMessage{}, map[string]json.RawMessage{}
	raw, err := a.ReadFile(file)
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
func (a App) writeMCPConfig(file string, root, servers map[string]json.RawMessage) error {
	root["mcpServers"] = mustJSON(servers)
	out, err := json.MarshalIndent(root, "", "  ")
	if err != nil {
		return err
	}
	return a.WriteFile(file, append(out, '\n'), 0o644)
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

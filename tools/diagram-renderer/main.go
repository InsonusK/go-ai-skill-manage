// Command diagram-renderer renders a feature's depends_on packages and their
// actual Go imports as JSON Canvas and SVG. This repository-local renderer needs
// no external diagram application; regenerate it with make diagrams.
package main

import (
	"encoding/json"
	"fmt"
	"go.yaml.in/yaml/v3"
	"go/parser"
	"go/token"
	"html"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

type node struct {
	ID     string `json:"id"`
	Type   string `json:"type"`
	File   string `json:"file"`
	X      int    `json:"x"`
	Y      int    `json:"y"`
	Width  int    `json:"width"`
	Height int    `json:"height"`
}
type edge struct {
	ID       string `json:"id"`
	From     string `json:"fromNode"`
	FromSide string `json:"fromSide"`
	To       string `json:"toNode"`
	ToSide   string `json:"toSide"`
}
type canvas struct {
	Nodes []node `json:"nodes"`
	Edges []edge `json:"edges"`
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
func run() error {
	if len(os.Args) != 2 {
		return fmt.Errorf("usage: diagram-renderer docs/features/NAME.md")
	}
	feature := os.Args[1]
	raw, err := os.ReadFile(feature)
	if err != nil {
		return err
	}
	sections := strings.SplitN(string(raw), "---", 3)
	if len(sections) != 3 {
		return fmt.Errorf("feature must have YAML frontmatter")
	}
	var spec struct {
		Feature   string   `yaml:"feature"`
		DependsOn []string `yaml:"depends_on"`
	}
	if err := yaml.Unmarshal([]byte(sections[1]), &spec); err != nil {
		return err
	}
	link := regexp.MustCompile(`\[[^\]]+\]\(([^)]+)\)`)
	files := map[string]string{}
	for _, value := range spec.DependsOn {
		match := link.FindStringSubmatch(value)
		if match == nil {
			return fmt.Errorf("invalid depends_on link: %s", value)
		}
		filename := filepath.Clean(filepath.Join(filepath.Dir(feature), match[1]))
		if _, err := os.Stat(filename); err != nil {
			return err
		}
		files[filepath.ToSlash(filepath.Dir(filename))] = filename
	}
	packages := []string{}
	for p := range files {
		packages = append(packages, p)
	}
	sort.Strings(packages)
	c := canvas{Nodes: []node{}, Edges: []edge{}}
	positions := map[string]node{}
	rows := map[int]int{}
	for _, p := range packages {
		column := 2
		switch {
		case strings.HasPrefix(p, "cmd/"):
			column = 0
		case p == "internal/command" || p == "internal/config":
			column = 1
		case strings.HasPrefix(p, "internal/infrastructure/"):
			column = 4
		case strings.HasPrefix(p, "internal/domain/services/"):
			column = 3
		}
		n := node{ID: p, Type: "file", File: filepath.ToSlash(files[p]), X: column * 320, Y: rows[column] * 125, Width: 280, Height: 85}
		rows[column]++
		positions[p] = n
		c.Nodes = append(c.Nodes, n)
	}
	module, err := os.ReadFile("go.mod")
	if err != nil {
		return err
	}
	moduleName := strings.Fields(string(module))[1]
	seen := map[string]bool{}
	for _, p := range packages {
		sourceFiles, err := filepath.Glob(filepath.Join(p, "*.go"))
		if err != nil {
			return err
		}
		for _, f := range sourceFiles {
			if strings.HasSuffix(f, "_test.go") {
				continue
			}
			ast, err := parser.ParseFile(token.NewFileSet(), f, nil, parser.ImportsOnly)
			if err != nil {
				return err
			}
			for _, i := range ast.Imports {
				imported, err := strconv.Unquote(i.Path.Value)
				if err != nil {
					return err
				}
				target := strings.TrimPrefix(imported, moduleName+"/")
				if _, ok := positions[target]; !ok || target == p {
					continue
				}
				id := p + "->" + target
				if seen[id] {
					continue
				}
				seen[id] = true
				c.Edges = append(c.Edges, edge{ID: id, From: p, FromSide: "right", To: target, ToSide: "left"})
			}
		}
	}
	sort.Slice(c.Edges, func(i, j int) bool { return c.Edges[i].ID < c.Edges[j].ID })
	out := filepath.Join(filepath.Dir(feature), "diagrams")
	if err := os.MkdirAll(out, 0755); err != nil {
		return err
	}
	name := strings.TrimSuffix(filepath.Base(feature), filepath.Ext(feature))
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(out, name+".canvas"), append(data, '\n'), 0644); err != nil {
		return err
	}
	height := 125
	for _, n := range c.Nodes {
		if n.Y+125 > height {
			height = n.Y + 125
		}
	}
	var svg strings.Builder
	fmt.Fprintf(&svg, `<svg xmlns="http://www.w3.org/2000/svg" viewBox="-20 -40 1620 %d" role="img"><title>%s package dependencies</title><defs><marker id="arrow" viewBox="0 0 10 10" refX="9" refY="5" markerWidth="5" markerHeight="5" orient="auto-start-reverse"><path d="M0 0L10 5L0 10z" fill="#64748b"/></marker></defs><rect x="-20" y="-40" width="1620" height="%d" fill="#f8fafc"/><text x="0" y="-15" font-family="sans-serif" font-size="17">Arrow: package imports dependency</text>`, height+40, html.EscapeString(spec.Feature), height+40)
	for _, e := range c.Edges {
		a, b := positions[e.From], positions[e.To]
		x1, y1, x2, y2 := a.X+a.Width, a.Y+a.Height/2, b.X, b.Y+b.Height/2
		fmt.Fprintf(&svg, `<path d="M%d %d C%d %d %d %d %d %d" fill="none" stroke="#94a3b8" stroke-width="1.5" marker-end="url(#arrow)"/>`, x1, y1, x1+40, y1, x2-40, y2, x2, y2)
	}
	for _, n := range c.Nodes {
		label := strings.TrimPrefix(n.ID, "internal/")
		parts := strings.Split(label, "/")
		fmt.Fprintf(&svg, `<g><title>%s</title><rect x="%d" y="%d" width="%d" height="%d" rx="9" fill="white" stroke="#334155"/><text x="%d" y="%d" font-family="monospace" font-size="17">%s</text><text x="%d" y="%d" font-family="monospace" font-size="12" fill="#475569">%s</text></g>`, html.EscapeString(n.ID), n.X, n.Y, n.Width, n.Height, n.X+14, n.Y+32, html.EscapeString(parts[len(parts)-1]), n.X+14, n.Y+59, html.EscapeString(strings.Join(parts[:len(parts)-1], "/")))
	}
	svg.WriteString("</svg>\n")
	return os.WriteFile(filepath.Join(out, name+".svg"), []byte(svg.String()), 0644)
}

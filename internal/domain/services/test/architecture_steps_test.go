package services_test

import (
	"context"
	"github.com/InsonusK/go-ai-skill-manage/tools/testsupport"
	"github.com/cucumber/godog"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

// allowedExternal maps an external library the domain may import to the
// only domain folder allowed to import it.
var allowedExternal = map[string]string{
	// A skill's frontmatter is YAML: parsing it is part of what a skill is
	// (entity.MakeSkillDocument), not an infrastructure concern, and a port
	// in front of a pure parser would buy nothing.
	"go.yaml.in/yaml/v3": "/entity/",
}

func architectureSteps(sc *godog.ScenarioContext) {
	var forbidden []string
	sc.Step(`^I inspect domain package imports$`, func(ctx context.Context) error {
		forbidden = []string{}
		err := filepath.WalkDir("../..", func(p string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				if d.Name() == "test" {
					return fs.SkipDir
				}
				return nil
			}
			if !strings.HasSuffix(p, ".go") {
				return nil
			}
			file, err := parser.ParseFile(token.NewFileSet(), p, nil, 0)
			if err != nil {
				return err
			}
			for _, imp := range file.Imports {
				name, err := strconv.Unquote(imp.Path.Value)
				if err != nil {
					return err
				}
				local := strings.HasPrefix(name, "github.com/InsonusK/go-ai-skill-manage/")
				external := !local && strings.Contains(strings.Split(name, "/")[0], ".")
				if external && allowedExternal[name] != "" && strings.Contains(filepath.ToSlash(p), allowedExternal[name]) {
					external = false
				}
				if local && !strings.Contains(name, "/internal/domain/") || name == "os" || name == "os/exec" || name == "net/http" || external {
					forbidden = append(forbidden, filepath.ToSlash(p)+": "+name)
				}
			}
			// context.Context is required only where cancellation pays off
			// (I/O, providers, loops over unbounded data -- see AGENTS.md),
			// which a parser can't tell; what it can check is Go's
			// convention: a function taking a context takes it first.
			for _, decl := range file.Decls {
				fn, ok := decl.(*ast.FuncDecl)
				if !ok {
					continue
				}
				if i := contextParam(fn.Type.Params); i > 0 {
					forbidden = append(forbidden, filepath.ToSlash(p)+": "+fn.Name.Name+" must take context.Context first")
				}
			}

			return nil
		})
		sort.Strings(forbidden)
		testsupport.Log("forbidden imports=%v", forbidden)
		return err
	})
	sc.Step(`^forbidden domain imports are$`, func(ctx context.Context, d *godog.DocString) error { return testsupport.JSON(forbidden, d) })
}

// contextParam returns the position of the first context.Context
// parameter, or -1 if there is none.
func contextParam(params *ast.FieldList) int {
	i := 0
	for _, field := range params.List {
		n := max(len(field.Names), 1)
		if typ, ok := field.Type.(*ast.SelectorExpr); ok {
			if pkg, ok := typ.X.(*ast.Ident); ok && pkg.Name == "context" && typ.Sel.Name == "Context" {
				return i
			}
		}
		i += n
	}
	return -1
}

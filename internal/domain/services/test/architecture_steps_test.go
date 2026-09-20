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
				if local && !strings.Contains(name, "/internal/domain/") || name == "os" || name == "os/exec" || name == "net/http" || !local && strings.Contains(strings.Split(name, "/")[0], ".") {
					forbidden = append(forbidden, filepath.ToSlash(p)+": "+name)
				}
			}
			if strings.Contains(filepath.ToSlash(p), "/services/") {
				for _, decl := range file.Decls {
					method, ok := decl.(*ast.FuncDecl)
					if !ok || method.Recv == nil || !method.Name.IsExported() {
						continue
					}
					valid := false
					if len(method.Type.Params.List) > 0 {
						if typ, ok := method.Type.Params.List[0].Type.(*ast.SelectorExpr); ok {
							if pkg, ok := typ.X.(*ast.Ident); ok {
								valid = pkg.Name == "context" && typ.Sel.Name == "Context"
							}
						}
					}
					if !valid {
						forbidden = append(forbidden, filepath.ToSlash(p)+": "+method.Name.Name+" must accept context.Context first")
					}
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

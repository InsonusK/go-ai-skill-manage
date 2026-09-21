package planning

import (
	"context"
	"fmt"
	"github.com/InsonusK/go-ai-skill-manage/internal/domain/interfaces"
	"github.com/InsonusK/go-ai-skill-manage/internal/domain/model"
	"io/fs"
	"path"
	"sort"
	"strings"
)

type Layout struct {
	Paths  map[string]string
	Shared []model.OutputFile
}

func BuildLayout(ctx context.Context, cat *model.SkillCatalog, sources interfaces.RepositoryLookup) (Layout, error) {
	layout := Layout{Paths: cat.Destinations(), Shared: []model.OutputFile{}}
	external := map[string]bool{}
	scan := func(f *model.File) {
		for _, l := range f.Links {
			if _, ok := layout.Paths[l.Target]; ok {
				continue
			}
			repoID, p, _ := strings.Cut(l.Target, "\x00")
			if _, dest, ok := cat.Destination(repoID, p); ok {
				layout.Paths[l.Target] = dest
			} else {
				external[l.Target] = true
			}
		}
	}
	for _, s := range cat.Skills {
		scan(&s.MainFile)
		for i := range s.Files {
			scan(&s.Files[i])
		}
	}
	if len(external) > 0 {
		for _, s := range cat.Skills {
			if s.Name == "files" {
				return layout, fmt.Errorf("output-collision: skill name files is reserved when external attachments exist")
			}
		}
	}
	keys := []string{}
	for k := range external {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	used := map[string]bool{}
	for _, key := range keys {
		repoID, p, _ := strings.Cut(key, "\x00")
		repo, ok := sources.Lookup(ctx, repoID)
		if !ok {
			return layout, fmt.Errorf("source not found: %s", repoID)
		}
		base := path.Base(p)
		if base == "." {
			return layout, fmt.Errorf("external repository root cannot be copied")
		}
		dest := path.Join("files", base)
		ext := path.Ext(base)
		stem := strings.TrimSuffix(base, ext)
		for n := 1; used[dest]; n++ {
			dest = path.Join("files", fmt.Sprintf("%s_%d%s", stem, n, ext))
		}
		used[dest] = true
		layout.Paths[key] = dest
		err := fs.WalkDir(repo.FS, p, func(current string, entry fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			relative := ""
			if current != p {
				relative = strings.TrimPrefix(current, p+"/")
			}
			output := path.Join(dest, relative)
			info, err := entry.Info()
			if err != nil {
				return err
			}
			mode := info.Mode().Perm()
			var data []byte
			if entry.IsDir() {
				mode |= fs.ModeDir
			} else {
				data, err = fs.ReadFile(repo.FS, current)
				if err != nil {
					return err
				}
			}
			layout.Shared = append(layout.Shared, model.OutputFile{Path: output, Data: data, Mode: mode})
			return nil
		})
		if err != nil {
			return layout, err
		}
	}
	return layout, nil
}

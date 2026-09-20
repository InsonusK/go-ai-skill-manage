package planning

import (
	"context"
	"fmt"
	"github.com/InsonusK/go-ai-skill-manage/internal/domain/model"
	"github.com/InsonusK/go-ai-skill-manage/internal/domain/services/discovery"
	"io/fs"
	"path"
	"sort"
	"strings"
)

type Layout struct {
	Paths  map[string]string
	Shared []model.OutputFile
}

func BuildLayout(ctx context.Context, cat *discovery.Catalog) (Layout, error) {
	layout := Layout{Paths: map[string]string{}, Shared: []model.OutputFile{}}
	occupied := map[string]string{}
	repos := map[string]*model.Repository{}
	put := func(key, dest string) error {
		if old, ok := occupied[dest]; ok && old != key {
			return fmt.Errorf("output-collision: %s", dest)
		}
		occupied[dest] = key
		layout.Paths[key] = dest
		return nil
	}
	for _, s := range cat.Skills {
		repos[s.Repo.ID] = s.Repo
		for _, f := range s.Files {
			dest := path.Join(s.Name, discovery.Relative(s, f.Path))
			if err := put(s.Repo.ID+"\x00"+f.Path, dest); err != nil {
				return layout, err
			}
		}
		layout.Paths[s.Repo.ID+"\x00"+s.Root] = path.Join(s.Name, "SKILL.md")
	}
	external := map[string]bool{}
	for _, s := range cat.Skills {
		for _, f := range s.Files {
			for _, l := range f.Links {
				if _, ok := layout.Paths[l.Target]; ok {
					continue
				}
				repoID, p, _ := strings.Cut(l.Target, "\x00")
				if owner := cat.Owner(ctx, repoID, p); owner != nil {
					layout.Paths[l.Target] = path.Join(owner.Name, discovery.Relative(owner, p))
				} else {
					external[l.Target] = true
				}
			}
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
		repo := repos[repoID]
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

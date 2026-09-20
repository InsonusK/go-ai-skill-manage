package discovery

import (
	"github.com/InsonusK/go-ai-skill-manage/internal/domain/model"
	"path"
)

// BuildSkillMap indexes the finalized catalog once, after relations
// expansion stops growing it, so every target's planning.BuildLayout call
// can look up a file's owning skill and destination without re-deriving
// ownership from cat.Skills. Call it exactly once per sync run, between
// relation expansion and the per-target planning loop.
func BuildSkillMap(cat *model.Catalog) (*model.SkillMap, error) {
	entries := make([]model.SkillEntry, 0, len(cat.Skills))
	originals := make(map[string]map[string]string, len(cat.Skills))
	for _, s := range cat.Skills {
		entries = append(entries, model.SkillEntry{
			Name: s.Name, SourceKey: s.Repo.ID, Root: s.Root, Main: s.Main, Flat: s.Flat, Dest: s.Name,
		})
		files := make(map[string]string, len(s.Files)+1)
		for _, f := range s.Files {
			files[f.Path] = path.Join(s.Name, Relative(s, f.Path))
		}
		// A link pointing at the skill's own directory (not a specific file)
		// still resolves to its SKILL.md.
		files[s.Root] = path.Join(s.Name, "SKILL.md")
		originals[s.Name] = files
	}
	return model.NewSkillMap(entries, originals)
}

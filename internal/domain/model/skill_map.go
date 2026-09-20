package model

import (
	"fmt"
	"path"
)

// SkillEntry is one skill's provenance and destination, decoupled from
// fs.FS/Repository so it is constructible by hand in unit tests.
type SkillEntry struct {
	Name      string // output directory name (== the map key in SkillMap.entries)
	SourceKey string // Repository.ID the skill was discovered in
	Root      string // original root path (directory, or the flat *.skill.md file itself)
	Main      string // original path of the skill's SKILL.md / *.skill.md
	Flat      bool   // true for a flat *.skill.md skill (Root == Main)
	Dest      string // output path of this skill relative to the in-memory skills root; always == Name today, kept explicit for forward-compatibility
}

type skillPath struct {
	Skill       string
	Destination string
}

// SkillMap is the finalized skill-name -> provenance/destination registry,
// built once after relations expansion stops growing the catalog and reused
// unmodified by planning across every target: the mapping from an original
// (source, path) pair to which skill owns it and where its content is
// destined never varies per target -- only the per-target path prefix does,
// which stays planning's concern.
type SkillMap struct {
	entries map[string]SkillEntry
	paths   map[string]skillPath
}

// OriginalKey builds the lookup key SkillMap, Link.Target, and a target's
// resolved output layout all use: a NUL-separated (sourceKey, path) pair, so
// identical relative paths from different sources never collide.
func OriginalKey(sourceKey, path string) string { return sourceKey + "\x00" + path }

// NewSkillMap indexes entries and the destination of every original file
// each one owns. originalPaths maps a skill name to {originalPath: destPath}
// for every file the skill owns (its own inventory, plus any alias paths
// such as a directory skill's root resolving to its SKILL.md).
//
// Two different original paths belonging to the SAME skill are allowed to
// share a destination (e.g. a directory skill's root and its SKILL.md file
// both resolve to "<name>/SKILL.md"); an output-collision error is returned
// only when two DIFFERENT skills would write to the same destination.
func NewSkillMap(entries []SkillEntry, originalPaths map[string]map[string]string) (*SkillMap, error) {
	m := &SkillMap{entries: map[string]SkillEntry{}, paths: map[string]skillPath{}}
	for _, e := range entries {
		m.entries[e.Name] = e
	}
	occupied := map[string]string{} // destination -> owning skill name
	for name, files := range originalPaths {
		entry, ok := m.entries[name]
		if !ok {
			continue
		}
		for original, dest := range files {
			if owner, exists := occupied[dest]; exists && owner != name {
				return nil, fmt.Errorf("output-collision: %s", dest)
			}
			occupied[dest] = name
			m.paths[OriginalKey(entry.SourceKey, original)] = skillPath{Skill: name, Destination: dest}
		}
	}
	return m, nil
}

// Entry returns the entry registered for a skill name.
func (m *SkillMap) Entry(name string) (SkillEntry, bool) {
	e, ok := m.entries[name]
	return e, ok
}

// Owner returns the skill owning originalPath inside sourceKey's source, and
// its output destination. It first tries an exact inventoried-path match,
// then falls back to OwnsPath/RelativePath against each entry's
// Root/Main/Flat, so callers get one answer regardless of whether the path
// was ever read as a file.
func (m *SkillMap) Owner(sourceKey, originalPath string) (SkillEntry, string, bool) {
	if p, ok := m.paths[OriginalKey(sourceKey, originalPath)]; ok {
		return m.entries[p.Skill], p.Destination, true
	}
	for _, e := range m.entries {
		if e.SourceKey == sourceKey && OwnsPath(e.Main, e.Root, e.Flat, originalPath) {
			dest := path.Join(e.Dest, RelativePath(e.Main, e.Root, originalPath))
			return e, dest, true
		}
	}
	return SkillEntry{}, "", false
}

// Destinations returns a defensive copy of every OriginalKey -> destination
// pairing, so callers may extend their own copy (e.g. with per-target
// external-attachment destinations) without mutating the shared SkillMap.
func (m *SkillMap) Destinations() map[string]string {
	out := make(map[string]string, len(m.paths))
	for k, p := range m.paths {
		out[k] = p.Destination
	}
	return out
}

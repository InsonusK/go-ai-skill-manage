package model

// SourceKey identifies what to fetch (type/path/tree), excluding selection
// fields (Subpaths/Tags/Name) that vary per SourceSpec even when they point at
// the same underlying source. Used by SourceManager to dedup acquisitions.
type SourceKey struct{ Type, Path, Tree string }

func (s SourceSpec) Key() SourceKey { return SourceKey{Type: s.Type, Path: s.Path, Tree: s.Tree} }

func (s SourceKey) String() string {
	if s.Tree == "" {
		return s.Type + ":" + s.Path
	}
	return s.Type + ":" + s.Path + "@" + s.Tree
}

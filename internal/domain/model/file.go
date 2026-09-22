package model

import "io/fs"

type FileImpl struct {
	Path  string
	Data  []byte
	Mode  fs.FileMode
	Links []Link

	skill *SkillImpl // owning skill, for Content() to resolve Repo/Root; set by
	// FilesByPath/Find. Left nil for a File built outside this package
	// (discovery.Rooted's MainFile/s.Files construction) -- harmless for
	// MainFile (Data is always already populated, so Content() never needs
	// to resolve it), and patched on demand by the deprecated Skill.Data
	// for s.Files.
}

// Content returns f's own content, reading it from its owning skill's
// source repository and caching the result on f.Data. f must have been
// obtained from a Skill's FilesByPath/Find (or be that skill's own
// MainFile) -- a File with no owning skill wired in will panic.
func (f *FileImpl) Content() ([]byte, error) {
	if f.Data == nil {
		data, err := fs.ReadFile(f.skill.Repo.FS, NestedRepoPath(f.skill.SkillDirPath, f.Path))
		if err != nil {
			return nil, err
		}
		f.Data = data
	}
	return f.Data, nil
}

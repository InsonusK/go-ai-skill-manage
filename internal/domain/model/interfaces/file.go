package interfaces

type FilePathKind string

const (
	//Path in absolute format relative to filesystem root
	Absolute FilePathKind = "absolute"
	//Path in absolute format relative to repository folder
	RepoAbsolute FilePathKind = "repo-relative"
	//Path in relative format related to skill it attached
	SkillRelative FilePathKind = "skill-relative"
)

type File interface {
	Content() ([]byte, error)
	Path(kind FilePathKind) (string, error)
}
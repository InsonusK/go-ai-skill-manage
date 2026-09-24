package sourcing

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"path"
	"path/filepath"
	"strings"

	"github.com/InsonusK/go-ai-skill-manage/internal/domain/entity"
	"github.com/InsonusK/go-ai-skill-manage/internal/domain/model"
	"github.com/InsonusK/go-ai-skill-manage/internal/domain/model/issues"
)

func isNotExist(err error) bool { return errors.Is(err, fs.ErrNotExist) }

// SkillCatalog is the active, path-driven source of truth for skills found
// in acquired repositories. Get* only look at skills already loaded,
// fetch* find and validate skills in a Repository (acquired lazily via
// Manager), and addPath remembers what fetch* found.
type SkillCatalog struct {
	Manager *Manager
	// AddRelations selects what TryGetOrFetchByPath/TryGetOrFetchByPathUp
	// do on a cache miss: true fetches the skill, false fails with
	// entity.ErrSkillNotCached -- mirrors the add_relations setting.
	AddRelations bool
	// ExcludeFromChecks maps a source to the top-level folders of its
	// skills that are loaded and copied with the skill but never checked:
	// no nested-skill check here, no link checks in validators. A source
	// missing from the map excludes nothing. See IsExcludedFromChecks.
	ExcludeFromChecks map[model.SourceKey][]string
	// cachedByPath maps entity.GetSkillKey(repo, p) to the skills a lookup
	// by path p answers: every skill at or below a requested path p, and
	// for each loaded skill its own folder (or, for a flat skill, its
	// marker file) -> that one skill. See addPath.
	cachedByPath map[string][]*entity.Skill
	// loaded is every loaded skill once, in the order it was loaded.
	loaded []*entity.Skill
}

var _ entity.SkillResolver = (*SkillCatalog)(nil)

// GetByPath returns the skills an earlier GetOrFetchByPath found at or
// below start, or the skill whose own folder (or flat marker file) start
// is. It never acquires a Repository nor reads a file; a start never
// requested before -- even one with loaded skills below it -- fails with
// entity.ErrSkillNotCached, since only a fetch knows it found them all.
//
// Примеры (репозиторий: скил guide в папке "a/guide", flat-скил
// "b.skill.md"; ранее был вызов GetOrFetchByPath("."), см. addPath):
//   - GetByPath(".")          -> [guide, flat] (запрошенный ранее путь)
//   - GetByPath("a/guide")    -> [guide]       (папка скила)
//   - GetByPath("b.skill.md") -> [flat]        (файл-маркер flat-скила)
//   - GetByPath("a")          -> ErrSkillNotCached: "a" не запрашивали,
//     и это не папка скила, хотя guide лежит ниже него
func (c *SkillCatalog) GetByPath(ctx context.Context, key model.SourceKey, start string) ([]*entity.Skill, error) {
	start, err := c.cachedRepoPath(ctx, key, start)
	if err != nil {
		return nil, err
	}
	if skills, ok := c.cachedByPath[entity.GetSkillKey(key, start)]; ok {
		return skills, nil
	}
	return nil, fmt.Errorf("%w: %s in %s", entity.ErrSkillNotCached, start, key)
}

// GetByPathUp returns the loaded skill whose folder holds repo-relative
// path p -- unlike GetByPath, which answers for a path and below, it
// searches up: p is usually a file inside some skill. It checks p and each
// of its parent folders by key, without reading the filesystem. Nothing
// found fails with entity.ErrSkillNotCached.
//
// Примеры (загружены guide с папкой "a/guide" и flat-скил "b.skill.md";
// репозиторий в ОС лежит в "/home/u/skills"):
//   - GetByPathUp("a/guide/docs/x.md") -> guide: проверены ключи
//     "a/guide/docs/x.md", "a/guide/docs", "a/guide" -- найден на последнем
//   - GetByPathUp("a/guide/SKILL.md")  -> guide (файл-маркер в папке скила)
//   - GetByPathUp("a/guide")           -> guide (сама папка скила)
//   - GetByPathUp("b.skill.md")        -> flat (файл-маркер flat-скила)
//   - GetByPathUp("/home/u/skills/a/guide/docs/x.md") -> guide (путь ОС
//     сначала переводится в "a/guide/docs/x.md", см. cleanRelative)
//   - GetByPathUp("a")          -> ErrSkillNotCached ("a" выше скила, а не в нём)
//   - GetByPathUp("x/notes.md") -> ErrSkillNotCached (ни одна папка-родитель
//     не папка загруженного скила)
func (c *SkillCatalog) GetByPathUp(ctx context.Context, key model.SourceKey, p string) (*entity.Skill, error) {
	p, err := c.cachedRepoPath(ctx, key, p)
	if err != nil {
		return nil, err
	}
	for dir := p; ; dir = path.Dir(dir) {
		if skill := c.skillAt(key, dir); skill != nil && ownsPath(skill, p) {
			return skill, nil
		}
		if dir == "." {
			return nil, fmt.Errorf("%w: no skill holds %s in %s", entity.ErrSkillNotCached, p, key)
		}
	}
}

// GetOrFetchByPath is GetByPath; on a cache miss it acquires key's
// Repository, fetches every valid skill at or below start (FetchByPath)
// and remembers them (addPath). Invalid candidates are returned as
// model.Issues next to the valid skills.
func (c *SkillCatalog) GetOrFetchByPath(ctx context.Context, key model.SourceKey, start string) ([]*entity.Skill, error) {
	skills, err := c.GetByPath(ctx, key, start)
	if !errors.Is(err, entity.ErrSkillNotCached) {
		return skills, err
	}
	repo, start, err := c.acquire(ctx, key, start)
	if err != nil {
		return nil, err
	}
	skills, problems := c.FetchByPath(ctx, repo, start)
	if len(problems) > 0 {
		// Not a complete answer for start: remember only each valid skill
		// at its own location, so a repeat call fetches again and reports
		// the same issues.
		for _, skill := range skills {
			c.addSkill(skill)
		}
		return skills, problems
	}
	c.addPath(repo.Key, start, skills)
	return skills, nil
}

// GetOrFetchByPathUp is GetByPathUp; on a cache miss it acquires key's
// Repository, fetches the skill whose folder holds p (FetchByPathUp) and
// remembers it (addPath).
func (c *SkillCatalog) GetOrFetchByPathUp(ctx context.Context, key model.SourceKey, p string) (*entity.Skill, error) {
	skill, err := c.GetByPathUp(ctx, key, p)
	if !errors.Is(err, entity.ErrSkillNotCached) {
		return skill, err
	}
	repo, p, err := c.acquire(ctx, key, p)
	if err != nil {
		return nil, err
	}
	skill, err = c.FetchByPathUp(ctx, repo, p)
	if err != nil {
		return nil, err
	}
	c.addPath(repo.Key, skill.DirOrMarkerPath(), []*entity.Skill{skill})
	return skill, nil
}

// TryGetOrFetchByPath is GetOrFetchByPath when c.AddRelations is set, and
// GetByPath -- failing with entity.ErrSkillNotCached on a miss -- otherwise.
func (c *SkillCatalog) TryGetOrFetchByPath(ctx context.Context, key model.SourceKey, start string) ([]*entity.Skill, error) {
	if !c.AddRelations {
		return c.GetByPath(ctx, key, start)
	}
	return c.GetOrFetchByPath(ctx, key, start)
}

// TryGetOrFetchByPathUp is GetOrFetchByPathUp when c.AddRelations is set,
// and GetByPathUp -- failing with entity.ErrSkillNotCached on a miss --
// otherwise.
func (c *SkillCatalog) TryGetOrFetchByPathUp(ctx context.Context, key model.SourceKey, p string) (*entity.Skill, error) {
	if !c.AddRelations {
		return c.GetByPathUp(ctx, key, p)
	}
	return c.GetOrFetchByPathUp(ctx, key, p)
}

// cachedRepoPath cleans p against key's already-acquired Repository
// without acquiring it; a Repository not acquired yet holds no loaded
// skill, so it fails with entity.ErrSkillNotCached.
//
// Примеры: для загруженного репозитория -- как cleanRelative
// ("./a//b/" -> "a/b"); для незагруженного -- на любой путь
// ErrSkillNotCached "source local:repo is not acquired".
func (c *SkillCatalog) cachedRepoPath(ctx context.Context, key model.SourceKey, p string) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	repo, ok := c.Manager.LookupKey(ctx, key)
	if !ok {
		return "", fmt.Errorf("%w: source %s is not acquired", entity.ErrSkillNotCached, key)
	}
	return cleanRelative(repo, p)
}

// acquire gets key's Repository via Manager (fetched once, then cached)
// and resolves p inside it (see normalizePath). A Repository the provider
// can't give is a "source-acquire" issue.
//
// Пример: acquire(key, "/home/u/skills/a/guide") -> (репозиторий, "a/guide")
// -- первый вызов для key загружает репозиторий, следующие берут его из
// кэша Manager.
func (c *SkillCatalog) acquire(ctx context.Context, key model.SourceKey, p string) (*entity.Repository, string, error) {
	repo, err := c.Manager.Get(ctx, key)
	if err != nil {
		return nil, "", issues.SkillIssue{Code: "source-acquire", Source: key.String(), Message: err.Error()}
	}
	p, err = normalizePath(repo, p)
	if err != nil {
		return nil, "", err
	}
	return repo, p, nil
}

// addPath remembers skills as the answer for path p of repository key, and
// each skill as the answer for its own location (DirOrMarkerPath) -- so GetByPath
// finds them by p or by a skill's folder, and GetByPathUp by any path
// inside a skill's folder.
//
// Пример: addPath(key, ".", [guide (папка "a/guide"), flat ("b.skill.md")])
// записывает три ключа:
//   - "."          -> [guide, flat] (ответ на запрошенный путь)
//   - "a/guide"    -> [guide]       (собственное место guide)
//   - "b.skill.md" -> [flat]        (собственное место flat-скила)
func (c *SkillCatalog) addPath(key model.SourceKey, p string, skills []*entity.Skill) {
	if c.cachedByPath == nil {
		c.cachedByPath = map[string][]*entity.Skill{}
	}
	for _, skill := range skills {
		if c.skillAt(key, skill.DirOrMarkerPath()) == nil {
			c.loaded = append(c.loaded, skill)
		}
		c.cachedByPath[entity.GetSkillKey(key, skill.DirOrMarkerPath())] = []*entity.Skill{skill}
	}
	c.cachedByPath[entity.GetSkillKey(key, p)] = skills
}

// addSkill remembers skill at its own location only (see addPath).
func (c *SkillCatalog) addSkill(skill *entity.Skill) {
	c.addPath(skill.Repo.Key, skill.DirOrMarkerPath(), []*entity.Skill{skill})
}

// Skills returns every loaded skill once, in the order it was loaded: a
// skill loaded later -- e.g. fetched while following a link -- comes after
// all skills loaded before it, so a caller walking the list by index while
// loading more sees the new ones too.
func (c *SkillCatalog) Skills() []*entity.Skill {
	return append([]*entity.Skill(nil), c.loaded...)
}

// skillAt returns cached skill which register ad this SourceKey and repo path
// p, or nil -- an entry for a requested path above skills is not one.
//
// Примеры (после addPath из примера выше):
//   - skillAt("a/guide")      -> guide
//   - skillAt("b.skill.md")   -> flat
//   - skillAt(".")            -> nil: ключ есть, но это ответ на запрошенный
//     путь, а не место какого-то скила
//   - skillAt("a")            -> nil: ключа нет
//   - skillAt("a/guide/docs") -> nil: папка внутри скила -- не его место
//     (для этого есть GetByPathUp)
func (c *SkillCatalog) skillAt(key model.SourceKey, p string) *entity.Skill {
	for _, skill := range c.cachedByPath[entity.GetSkillKey(key, p)] {
		if skill.DirOrMarkerPath() == p {
			return skill
		}
	}
	return nil
}

// FetchByPath finds every valid skill at or below start inside repo --
// start is a path from the repository folder, already resolved (see
// normalizePath; GetOrFetchByPath does that before calling it): a
// folder that is a skill's folder becomes one skill and is not searched
// further, any other folder is searched into, a "*.skill.md" file outside
// a skill's folder is a flat skill. A skill already loaded at its own
// location is reused, not built again. Invalid candidates are returned as
// issues; nothing is remembered (see addPath).
//
// Примеры (в репозитории: "a/guide/SKILL.md", "a/guide/docs/x.md",
// "b.skill.md", "x/notes.md"):
//   - FetchByPath(".")            -> [guide, flat] ("x" обойдена, скилов нет)
//   - FetchByPath("a")            -> [guide]
//   - FetchByPath("a/guide")      -> [guide] (start сам -- папка скила)
//   - FetchByPath("b.skill.md")   -> [flat]
//   - FetchByPath("a/guide/docs") -> [] -- ищет только вниз; скил, в папке
//     которого лежит start, ищет FetchByPathUp
func (c *SkillCatalog) FetchByPath(ctx context.Context, repo *entity.Repository, start string) ([]*entity.Skill, issues.SkillIssues) {
	out := []*entity.Skill{}
	var problems issues.SkillIssues
	var scan func(string)
	scan = func(p string) {
		if err := ctx.Err(); err != nil {
			problems = append(problems, issues.SkillIssue{Code: "canceled", Source: repo.Key.String(), File: p, Message: err.Error()})
			return
		}
		if skill := c.skillAt(repo.Key, p); skill != nil {
			out = append(out, skill)
			return
		}
		info, err := fs.Stat(repo.FS, p)
		if err != nil {
			if !isNotExist(err) {
				problems = append(problems, issues.SkillIssue{Code: "source-read", Source: repo.Key.String(), File: p, Message: err.Error()})
			}
			return
		}
		if !info.IsDir() {
			if strings.HasSuffix(p, ".skill.md") {
				skill, err := entity.MakeSkill(repo, p, "", entity.FlatSkill)
				if err != nil {
					problems = append(problems, issues.SkillIssue{Code: "invalid-name", Source: repo.Key.String(), File: p, Message: err.Error()})
					return
				}
				if err := c.validate(skill); err != nil {
					problems = append(problems, *err)
					return
				}
				out = append(out, skill)
			}
			return
		}
		skill, err := c.isSkillDir(repo, p)
		if err != nil {
			problems = append(problems, issues.SkillIssue{Code: "invalid-skill", Source: repo.Key.String(), File: p, Message: err.Error()})
			return
		}
		if skill != nil {
			if err := c.validate(skill); err != nil {
				problems = append(problems, *err)
				return
			}
			out = append(out, skill)
			return
		}
		entries, err := fs.ReadDir(repo.FS, p)
		if err != nil {
			problems = append(problems, issues.SkillIssue{Code: "source-read", Source: repo.Key.String(), File: p, Message: err.Error()})
			return
		}
		for _, e := range entries {
			if e.Name() != ".git" {
				scan(path.Join(p, e.Name()))
			}
		}
	}
	scan(path.Clean(start))
	return out, problems
}

// FetchByPathUp finds the valid skill whose folder holds p inside repo
// -- p is a path from the repository folder, already resolved (see
// normalizePath; GetOrFetchByPathUp does that before calling it): it
// walks p's parent folders up to the repository folder until the first
// skill's folder (or takes p itself when it is a flat "*.skill.md" skill).
// Its neighbours are not loaded; nothing is remembered (see addPath).
//
// Примеры (в репозитории: "a/guide/SKILL.md", "a/guide/docs/x.md",
// "b.skill.md", "h.skill/h.skill.md", "f/f.skill.md", "x/one/SKILL.md",
// "x/notes.md"):
//   - FetchByPathUp("a/guide/docs/x.md") -> guide: проверены папки
//     "a/guide/docs" (не скил), "a/guide" (скил)
//   - FetchByPathUp("a/guide/SKILL.md") -> guide: первой проверяется папка
//     файла, "a/guide" -- agent-dir скил
//   - FetchByPathUp("a/guide")          -> guide (папка сама -- скил)
//   - FetchByPathUp("h.skill/h.skill.md") -> human-dir скил в папке
//     "h.skill": файл назван по папке, значит это её маркер, а не flat-скил
//   - FetchByPathUp("b.skill.md")   -> flat-скил "b.skill.md"
//   - FetchByPathUp("f/f.skill.md") -> flat-скил "f/f.skill.md": папка "f"
//     не "*.skill", значит "f.skill.md" в ней -- не маркер human-dir скила
//   - FetchByPathUp("x/notes.md")   -> "skill-not-found": проверены "x" и
//     "."; "x/one" -- соседняя папка, а не родитель
func (c *SkillCatalog) FetchByPathUp(ctx context.Context, repo *entity.Repository, p string) (*entity.Skill, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	info, err := fs.Stat(repo.FS, p)
	if err != nil {
		return nil, err
	}
	dir := p
	if !info.IsDir() {
		dir = path.Dir(p)
	}
	for {
		skill, err := c.isSkillDir(repo, dir)
		if err != nil {
			return nil, issues.SkillIssue{Code: "invalid-skill", Source: repo.Key.String(), File: dir, Message: err.Error()}
		}
		// A file that isn't claimed by its own folder's marker may still
		// be a flat skill of its own.
		if skill == nil && dir == path.Dir(p) && !info.IsDir() && strings.HasSuffix(p, ".skill.md") {
			if skill, err = entity.MakeSkill(repo, p, "", entity.FlatSkill); err != nil {
				return nil, issues.SkillIssue{Code: "invalid-name", Source: repo.Key.String(), File: p, Message: err.Error()}
			}
		}
		if skill != nil {
			if err := c.validate(skill); err != nil {
				return nil, *err
			}
			return skill, nil
		}
		if dir == "." {
			return nil, issues.SkillIssue{Code: "skill-not-found", Source: repo.Key.String(), File: p, Message: "no skill holds this path"}
		}
		dir = path.Dir(dir)
	}
}

// validate checks a skill fetch* just built: its files can be listed
// (FilesByPath(""), which also warms that cache) and none of them is
// another skill's marker outside the folders excluded from checks
// (nested-skill, see nestedSkillPath).
func (c *SkillCatalog) validate(skill *entity.Skill) *issues.SkillIssue {
	files, err := skill.FilesByPath("")
	if err != nil {
		return &issues.SkillIssue{Code: "source-read", Source: skill.Repo.Key.String(), Skill: skill.Name, SkillPath: skill.DirOrMarkerPath(), File: skill.MainFilePath, Message: err.Error()}
	}
	if nested := c.nestedSkillPath(skill, files); nested != "" {
		return &issues.SkillIssue{Code: "nested-skill", Source: skill.Repo.Key.String(), Skill: skill.Name, SkillPath: skill.DirOrMarkerPath(), File: nested, Message: fmt.Sprintf("nested-skill: %s", nested)}
	}
	return nil
}

// nestedSkillPath returns the skill-relative Path of the first file in
// files that looks like another skill's own marker (SKILL.md or
// *.skill.md) and isn't excluded from checks, or "" if none is found. A
// marker in an excluded folder is deliberately not flagged -- that folder
// is still part of the owning skill and copied along with it, not a
// validation failure.
//
// Примеры (пути файлов -- от папки скила, исключена папка "examples"):
//   - ["docs/a.md", "b/SKILL.md"]   -> "b/SKILL.md"
//   - ["docs/x.skill.md"]           -> "docs/x.skill.md"
//   - ["examples/demo/SKILL.md"]    -> "" (верхняя папка "examples" исключена)
//   - ["docs/examples/SKILL.md"]    -> "docs/examples/SKILL.md" (исключается
//     только папка верхнего уровня)
//   - ["notSKILL.md", "a.skill.md.bak"] -> "" (имена не маркеры)
func (c *SkillCatalog) nestedSkillPath(skill *entity.Skill, files []*entity.File) string {
	for _, f := range files {
		relPath, err := f.Path(model.SkillRelative)
		if err != nil {
			continue
		}
		name := path.Base(relPath)
		if name != "SKILL.md" && !strings.HasSuffix(name, ".skill.md") {
			continue
		}
		if !c.IsExcludedFromChecks(skill, relPath) {
			return relPath
		}
	}
	return ""
}

// IsExcludedFromChecks reports whether rel, a path from skill's folder,
// lies in a top-level folder excluded from checks for skill's source (see
// ExcludeFromChecks).
//
// Примеры (для источника скила исключена "examples"):
//   - "examples/app/README.md" -> true
//   - "examples"               -> true (сама папка)
//   - "docs/examples/page.md"  -> false: исключается только верхний уровень
//   - "examples.md"            -> false: файл, а не папка "examples"
func (c *SkillCatalog) IsExcludedFromChecks(skill *entity.Skill, rel string) bool {
	first := strings.SplitN(rel, "/", 2)[0]
	for _, folder := range c.ExcludeFromChecks[skill.Repo.Key] {
		if first == folder {
			return true
		}
	}
	return false
}

// isSkillDir tests whether dir is a directory skill's root -- the shallow,
// single-level marker check only (mirrors Detector.Rooted's first
// fs.ReadDir loop); the deep nested-file walk is Skill.FilesByPath's job
// (a pure lister) and nested-skill detection is validate's own job over its
// result (nestedSkillPath), both triggered by validate once the Skill
// exists.
//
// Примеры (содержимое папки dir -> результат):
//   - "a/guide" с "SKILL.md"           -> agent-dir скил, маркер "a/guide/SKILL.md"
//   - "h.skill" с "h.skill.md"         -> human-dir скил, маркер "h.skill/h.skill.md"
//   - "o.skill" только с "other.skill.md" -> nil: маркер human-dir должен
//     называться по папке ("o.skill.md")
//   - "f" с "f.skill.md"               -> nil: папка не "*.skill", это flat-скил,
//     его находит обход как файл
//   - "p" только с подпапкой "q/SKILL.md" -> nil: вглубь не смотрит
//   - "c1.skill" с "SKILL.md" и "c1.skill.md" -> "pattern-conflict: multiple
//     directory markers"
//   - "c2" с "SKILL.md" и "other.skill.md"    -> "pattern-conflict: directory
//     marker and flat skill"
func (c *SkillCatalog) isSkillDir(repo *entity.Repository, dir string) (*entity.Skill, error) {
	entries, err := fs.ReadDir(repo.FS, dir)
	if err != nil {
		return nil, err
	}

	markers := []string{}
	flats := []string{}
	human := path.Base(dir)
	human = strings.TrimSuffix(human, ".skill") + ".skill.md"
	// Lookup skil files in directory, but DON'T recurse into subdirectories
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		if e.Name() == "SKILL.md" || (strings.HasSuffix(dir, ".skill") && e.Name() == human) {
			markers = append(markers, e.Name())
		}
		if strings.HasSuffix(e.Name(), ".skill.md") {
			flats = append(flats, e.Name())
		}
	}
	if len(markers) == 0 {
		return nil, nil
	}
	if len(markers) > 1 {
		return nil, fmt.Errorf("pattern-conflict: multiple directory markers")
	}
	for _, flat := range flats {
		if flat != markers[0] {
			return nil, fmt.Errorf("pattern-conflict: directory marker and flat skill")
		}
	}
	main := path.Join(dir, markers[0])
	format := entity.AgentDirSkill
	if markers[0] != "SKILL.md" {
		format = entity.HumanDirSkill
	}
	return entity.MakeSkill(repo, main, dir, format)
}

// ownsPath отвечает на вопрос "принадлежит ли repo-relative путь p скилу s?"
// без чтения файлов -- только по уже известным идентичности/расположению
// скила (s.MainFilePath/s.SkillDirPath/s.Format).
//
// Правила:
//   - сам главный файл или сама корневая директория скила всегда принадлежат
//     ему;
//   - для flat-скила (SkillDirPath == "") этим всё и ограничивается -- у
//     него нет директории, в которую что-то ещё могло бы попасть;
//   - для directory-скила (agent-dir/human-dir) владение распространяется на
//     всё, что лежит внутри его директории, даже если этот файл ещё ни разу
//     не читался через FilesByPath.
//
// Примеры для скила с SkillDirPath="a/guide", MainFilePath="a/guide/SKILL.md",
// Format=AgentDirSkill:
//   - ownsPath(s, "a/guide/SKILL.md")   -> true  (главный файл)
//   - ownsPath(s, "a/guide")            -> true  (сама директория)
//   - ownsPath(s, "a/guide/notes.md")   -> true  (файл внутри директории)
//   - ownsPath(s, "a/other/notes.md")   -> false (другая директория)
//
// Для flat-скила с MainFilePath="guide.skill.md", SkillDirPath=""
// (Format=FlatSkill):
//   - ownsPath(s, "guide.skill.md")     -> true  (главный файл)
//   - ownsPath(s, "guide.skill.md.bak") -> false (директории нет, значит и
//     "владения" за пределами самого файла тоже нет)
func ownsPath(s *entity.Skill, p string) bool {
	if p == s.MainFilePath || p == s.SkillDirPath {
		return true
	}
	if s.Format == entity.FlatSkill {
		return false
	}
	return s.SkillDirPath == "." || strings.HasPrefix(p, strings.TrimSuffix(s.SkillDirPath, "/")+"/")
}

// relativePath переводит repo-relative путь p, уже принадлежащий скилу s
// (см. ownsPath), в путь внутри выходной директории этого скила -- то есть
// в то имя, под которым файл окажется после копирования/сборки скила.
//
// Правила:
//   - главный файл или сама директория скила всегда становятся "SKILL.md"
//     -- так унифицируются все три формата (flat/human-dir/agent-dir), у
//     которых главный файл на диске называется по-разному;
//   - если скил лежит прямо в корне репозитория (SkillDirPath == "."), путь
//     не меняется;
//   - иначе у пути отрезается префикс директории скила.
//
// Примеры для того же скила (SkillDirPath="a/guide", MainFilePath=
// "a/guide/SKILL.md"):
//   - relativePath(s, "a/guide/SKILL.md")     -> "SKILL.md"
//   - relativePath(s, "a/guide")               -> "SKILL.md"
//   - relativePath(s, "a/guide/docs/intro.md") -> "docs/intro.md"
//
// Для скила, лежащего в корне репозитория (SkillDirPath="."):
//   - relativePath(s, "notes.md") -> "notes.md" (без изменений)
func relativePath(s *entity.Skill, p string) string {
	if p == s.MainFilePath || p == s.SkillDirPath {
		return "SKILL.md"
	}
	if s.SkillDirPath == "." {
		return p
	}
	return strings.TrimPrefix(p, path.Clean(s.SkillDirPath)+"/")
}

// Owner ищет среди уже найденных (через GetOrFetchByPath) скилов репозитория
// repoID тот единственный, которому принадлежит repo-relative путь p (см.
// ownsPath), и возвращает его. Если ни один известный скил не владеет этим
// путём -- например путь ещё не был обнаружен через GetOrFetchByPath, или
// принадлежит другому репозиторию -- возвращает nil.
//
// repoID сравнивается со строковым представлением идентичности репозитория
// (model.SourceKey.String()), а не с сырыми полями SourceKey -- так
// вызывающему (например relations.Expander, резолвящему ссылку "куда-то в
// этот же репозиторий") не нужно знать структуру SourceKey, только её
// строковую форму, уже известную по Link.Target/OriginalKey.
//
// Пример: после GetOrFetchByPath нашёл скил "guide" с SkillDirPath="a/guide" в
// репозитории с Key.String()=="local:repo":
//   - Owner(ctx, "local:repo", "a/guide/docs/intro.md") -> скил "guide"
//   - Owner(ctx, "local:repo", "b/other.md")             -> nil (не найден)
//   - Owner(ctx, "other:repo", "a/guide/docs/intro.md")  -> nil (другой репозиторий)
func (c *SkillCatalog) Owner(ctx context.Context, repoID, p string) *entity.Skill {
	for _, s := range c.loaded {
		if s.Repo.Key.String() == repoID && ownsPath(s, p) {
			return s
		}
	}
	return nil
}

// Destination -- как Owner, но сразу возвращает и владеющий скил по имени,
// и путь p, переведённый в координаты выходной директории этого скила (см.
// relativePath), собранные в одну выходную строку "имя_скила/путь". Третье
// возвращаемое значение -- ok -- false, если владелец не найден (тогда имя
// и путь пустые).
//
// Пример (тот же скил "guide", SkillDirPath="a/guide"):
//   - Destination(ctx, "local:repo", "a/guide/docs/intro.md")
//     -> ("guide", "guide/docs/intro.md", true)
//   - Destination(ctx, "local:repo", "a/guide/SKILL.md")
//     -> ("guide", "guide/SKILL.md", true)
//   - Destination(ctx, "local:repo", "b/other.md")
//     -> ("", "", false)
func (c *SkillCatalog) Destination(ctx context.Context, repoID, p string) (name, dest string, ok bool) {
	s := c.Owner(ctx, repoID, p)
	if s == nil {
		return "", "", false
	}
	return s.Name, path.Join(s.Name, relativePath(s, p)), true
}

// cleanRelative resolves start into a clean, valid, repo-relative path --
// pure path arithmetic, no filesystem access, so a cache hit on the result
// costs nothing beyond it.
//
// Примеры (репозиторий в ОС лежит в "/home/u/skills"):
//   - "a/b"                -> "a/b"
//   - "./a//b/"            -> "a/b"
//   - "a/../b"             -> "b"
//   - "" или "."           -> "." (папка репозитория)
//   - "/home/u/skills/a/b" -> "a/b" (путь ОС внутри репозитория)
//   - "/home/u/skills"     -> "."
//   - "../x"               -> unsafe-subpath `unsafe subpath "../x"`
//   - "/home/u/other/x"    -> unsafe-subpath `unsafe subpath "../other/x"` (вне репозитория)
//   - `a\b` (на Linux)     -> unsafe-subpath `unsafe subpath "a\\b"`
//   - "/abs" при пустом пути репозитория в ОС -> unsafe-subpath с ошибкой filepath.Rel
func cleanRelative(repo *entity.Repository, start string) (string, error) {
	if filepath.IsAbs(start) {
		rel, err := filepath.Rel(repo.RootPath, start)
		if err != nil {
			return "", issues.SkillIssue{Code: "unsafe-subpath", Source: repo.Key.String(), File: start, Message: err.Error()}
		}
		start = rel
	}
	start = filepath.ToSlash(filepath.Clean(start))
	if !fs.ValidPath(start) || strings.Contains(start, "\\") {
		return "", issues.SkillIssue{Code: "unsafe-subpath", Source: repo.Key.String(), File: start, Message: fmt.Sprintf("unsafe subpath %q", start)}
	}
	return start, nil
}

// normalizePath resolves start into a clean, valid, repo-relative path safe to
// pass to repo.FS, additionally confirming it exists.
//
// Примеры (репозиторий "local:repo" в ОС лежит в "/home/u/skills", в нём
// есть "a/guide/docs/x.md"):
//   - "a/guide"                          -> "a/guide"
//   - "/home/u/skills/a/guide/docs/x.md" -> "a/guide/docs/x.md"
//   - "a/missing" -> missing-subpath `subpath "a/missing" does not exist in repository "local:repo"`
//   - "../x"      -> unsafe-subpath `unsafe subpath "../x"` (как у cleanRelative)
func normalizePath(repo *entity.Repository, start string) (string, error) {
	start, err := cleanRelative(repo, start)
	if err != nil {
		return "", err
	}
	if _, err := fs.Stat(repo.FS, start); err != nil {
		return "", issues.SkillIssue{Code: "missing-subpath", Source: repo.Key.String(), File: start, Message: fmt.Sprintf("subpath %q does not exist in repository %q", start, repo.Key)}
	}
	return start, nil
}

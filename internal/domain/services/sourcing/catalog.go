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
)

func isNotExist(err error) bool { return errors.Is(err, fs.ErrNotExist) }

// DefaultSkipFolders names top-level folders exempt from nested-skill
// validation -- they're still part of the owning skill and copied with it,
// just never walked into for a skill marker of their own (e.g. a packaged
// example app that happens to contain its own SKILL.md). Hardcoded for
// now and passed at SkillCatalog construction (SourceSpec.SkipFolders,
// read from each source's config, is not wired to this yet -- deferred,
// see AGENTS.md).
var defaultSkipFoldersInNestedChecker = []string{"examples"}

// SetDefaultLinkSearcher вызывается ОДИН раз при старте приложения
func SetSkipFoldersInNestedChecker(skipFolders []string) {
	defaultSkipFoldersInNestedChecker = skipFolders
}

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
	// cachedByPath maps entity.GetSkillKey(repo, p) to the skills a lookup
	// by path p answers: every skill at or below a requested path p, and
	// for each loaded skill its own folder (or, for a flat skill, its
	// marker file) -> that one skill. See addPath.
	cachedByPath map[string][]*entity.Skill
}

var _ entity.SkillResolver = (*SkillCatalog)(nil)

// GetByPath returns the skills an earlier GetOrFetchByPath found at or
// below start, or the skill whose own folder (or flat marker file) start
// is. It never acquires a Repository nor reads a file; a start never
// requested before -- even one with loaded skills below it -- fails with
// entity.ErrSkillNotCached, since only a fetch knows it found them all.
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
// Repository, fetches every valid skill at or below start (fetchByPath)
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
	skills, issues := c.fetchByPath(ctx, repo, start)
	if len(issues) > 0 {
		// Not a complete answer for start: remember only each valid skill
		// at its own location, so a repeat call fetches again and reports
		// the same issues.
		for _, skill := range skills {
			c.addPath(skill.Repo.Key, ownPath(skill), []*entity.Skill{skill})
		}
		return skills, issues
	}
	c.addPath(repo.Key, start, skills)
	return skills, nil
}

// GetOrFetchByPathUp is GetByPathUp; on a cache miss it acquires key's
// Repository, fetches the skill whose folder holds p (fetchByPathUp) and
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
	skill, err = c.fetchByPathUp(ctx, repo, p)
	if err != nil {
		return nil, err
	}
	c.addPath(repo.Key, ownPath(skill), []*entity.Skill{skill})
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
// and resolves p inside it (see normalizePath).
func (c *SkillCatalog) acquire(ctx context.Context, key model.SourceKey, p string) (*entity.Repository, string, error) {
	repo, err := c.Manager.Get(ctx, key)
	if err != nil {
		return nil, "", err
	}
	p, err = normalizePath(repo, p)
	if err != nil {
		return nil, "", err
	}
	return repo, p, nil
}

// addPath remembers skills as the answer for path p of repository key, and
// each skill as the answer for its own location (ownPath) -- so GetByPath
// finds them by p or by a skill's folder, and GetByPathUp by any path
// inside a skill's folder.
func (c *SkillCatalog) addPath(key model.SourceKey, p string, skills []*entity.Skill) {
	if c.cachedByPath == nil {
		c.cachedByPath = map[string][]*entity.Skill{}
	}
	c.cachedByPath[entity.GetSkillKey(key, p)] = skills
	for _, skill := range skills {
		c.cachedByPath[entity.GetSkillKey(key, ownPath(skill))] = []*entity.Skill{skill}
	}
}

// skillAt returns the loaded skill whose own location (ownPath) is exactly
// p, or nil -- an entry for a requested path above skills is not one.
func (c *SkillCatalog) skillAt(key model.SourceKey, p string) *entity.Skill {
	for _, skill := range c.cachedByPath[entity.GetSkillKey(key, p)] {
		if ownPath(skill) == p {
			return skill
		}
	}
	return nil
}

// ownPath is a skill's own location: its folder, or its marker file for a
// flat skill, which has no folder.
func ownPath(s *entity.Skill) string {
	if s.SkillDirPath != "" {
		return s.SkillDirPath
	}
	return s.MainFilePath
}

// fetchByPath finds every valid skill at or below start inside repo: a
// folder that is a skill's folder becomes one skill and is not searched
// further, any other folder is searched into, a "*.skill.md" file outside
// a skill's folder is a flat skill. A skill already loaded at its own
// location is reused, not built again. Invalid candidates are returned as
// issues; nothing is remembered (see addPath).
func (c *SkillCatalog) fetchByPath(ctx context.Context, repo *entity.Repository, start string) ([]*entity.Skill, model.Issues) {
	out := []*entity.Skill{}
	var issues model.Issues
	var scan func(string)
	scan = func(p string) {
		if err := ctx.Err(); err != nil {
			issues = append(issues, model.Issue{Code: "canceled", File: p, Message: err.Error()})
			return
		}
		if skill := c.skillAt(repo.Key, p); skill != nil {
			out = append(out, skill)
			return
		}
		info, err := fs.Stat(repo.FS, p)
		if err != nil {
			if !isNotExist(err) {
				issues = append(issues, model.Issue{Code: "source-read", File: p, Message: err.Error()})
			}
			return
		}
		if !info.IsDir() {
			if strings.HasSuffix(p, ".skill.md") {
				skill, err := entity.MakeSkill(repo, p, "", entity.FlatSkill)
				if err != nil {
					issues = append(issues, model.Issue{Code: "invalid-name", File: p, Message: err.Error()})
					return
				}
				if err := validate(skill); err != nil {
					issues = append(issues, *err)
					return
				}
				out = append(out, skill)
			}
			return
		}
		skill, err := c.isSkillDir(repo, p)
		if err != nil {
			issues = append(issues, model.Issue{Code: "invalid-skill", File: p, Message: err.Error()})
			return
		}
		if skill != nil {
			if err := validate(skill); err != nil {
				issues = append(issues, *err)
				return
			}
			out = append(out, skill)
			return
		}
		entries, err := fs.ReadDir(repo.FS, p)
		if err != nil {
			issues = append(issues, model.Issue{Code: "source-read", File: p, Message: err.Error()})
			return
		}
		for _, e := range entries {
			if e.Name() != ".git" {
				scan(path.Join(p, e.Name()))
			}
		}
	}
	scan(path.Clean(start))
	return out, issues
}

// fetchByPathUp finds the valid skill whose folder holds p inside repo: it
// walks p's parent folders up to the repository folder until the first
// skill's folder (or takes p itself when it is a flat "*.skill.md" skill).
// Its neighbours are not loaded; nothing is remembered (see addPath).
func (c *SkillCatalog) fetchByPathUp(ctx context.Context, repo *entity.Repository, p string) (*entity.Skill, error) {
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
			return nil, model.Issue{Code: "invalid-skill", File: dir, Message: err.Error()}
		}
		// A file that isn't claimed by its own folder's marker may still
		// be a flat skill of its own.
		if skill == nil && dir == path.Dir(p) && !info.IsDir() && strings.HasSuffix(p, ".skill.md") {
			if skill, err = entity.MakeSkill(repo, p, "", entity.FlatSkill); err != nil {
				return nil, model.Issue{Code: "invalid-name", File: p, Message: err.Error()}
			}
		}
		if skill != nil {
			if err := validate(skill); err != nil {
				return nil, *err
			}
			return skill, nil
		}
		if dir == "." {
			return nil, model.Issue{Code: "skill-not-found", File: p, Message: "no skill holds this path"}
		}
		dir = path.Dir(dir)
	}
}

// validate checks a skill fetch* just built: its files can be listed
// (FilesByPath(""), which also warms that cache) and none of them is
// another skill's marker outside the skip folders (nested-skill, see
// nestedSkillPath).
func validate(skill *entity.Skill) *model.Issue {
	files, err := skill.FilesByPath("")
	if err != nil {
		return &model.Issue{Code: "source-read", Skill: skill.Name, File: skill.MainFilePath, Message: err.Error()}
	}
	if nested := nestedSkillPath(files, defaultSkipFoldersInNestedChecker); nested != "" {
		return &model.Issue{Code: "nested-skill", Skill: skill.Name, File: nested, Message: fmt.Sprintf("nested-skill: %s", nested)}
	}
	return nil
}

// nestedSkillPath returns the skill-relative Path of the first file in
// files that looks like another skill's own marker (SKILL.md or
// *.skill.md) and whose top-level folder isn't one of skipFolders, or ""
// if none is found. A marker under skipFolders is deliberately not
// flagged -- that folder is still part of the owning skill and copied
// along with it (see DefaultSkipFolders), not a validation failure.
func nestedSkillPath(files []*entity.File, skipFolders []string) string {
	for _, f := range files {
		relPath, err := f.Path(model.SkillRelative)
		if err != nil {
			continue
		}
		name := relPath
		if idx := strings.LastIndex(name, "/"); idx >= 0 {
			name = name[idx+1:]
		}
		if name != "SKILL.md" && !strings.HasSuffix(name, ".skill.md") {
			continue
		}
		first := strings.SplitN(relPath, "/", 2)[0]
		exempt := false
		for _, skip := range skipFolders {
			if first == skip {
				exempt = true
				break
			}
		}
		if !exempt {
			return relPath
		}
	}
	return ""
}

// isSkillDir tests whether dir is a directory skill's root -- the shallow,
// single-level marker check only (mirrors Detector.Rooted's first
// fs.ReadDir loop); the deep nested-file walk is Skill.FilesByPath's job
// (a pure lister) and nested-skill detection is accept's own job over its
// result (nestedSkillPath), both triggered by accept once the Skill
// exists.
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
	for _, skills := range c.cachedByPath {
		for _, s := range skills {
			if s.Repo.Key.String() == repoID && ownsPath(s, p) {
				return s
			}
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
func cleanRelative(repo *entity.Repository, start string) (string, error) {
	if filepath.IsAbs(start) {
		rel, err := filepath.Rel(repo.RootPath, start)
		if err != nil {
			return "", err
		}
		start = rel
	}
	start = filepath.ToSlash(filepath.Clean(start))
	if !fs.ValidPath(start) || strings.Contains(start, "\\") {
		return "", fmt.Errorf("unsafe subpath %q", start)
	}
	return start, nil
}

// normalizePath resolves start into a clean, valid, repo-relative path safe to
// pass to repo.FS, additionally confirming it exists.
func normalizePath(repo *entity.Repository, start string) (string, error) {
	start, err := cleanRelative(repo, start)
	if err != nil {
		return "", err
	}
	if _, err := fs.Stat(repo.FS, start); err != nil {
		return "", fmt.Errorf("subpath %q does not exist in repository %q", start, repo.Key)
	}
	return start, nil
}

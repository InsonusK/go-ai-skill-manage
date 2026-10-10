package validators

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"strings"

	"github.com/InsonusK/go-ai-skill-manager/internal/domain/entity"
	"github.com/InsonusK/go-ai-skill-manager/internal/domain/model"
	"github.com/InsonusK/go-ai-skill-manager/internal/domain/model/issues"
	"github.com/InsonusK/go-ai-skill-manager/internal/domain/services/sourcing"
	"github.com/InsonusK/go-ai-skill-manager/internal/domain/services/validator"
)

// LinkValidator checks that every link in the markdown files of the loaded
// skills leads to an existing file -- and, when it names an anchor, to an
// existing anchor in it -- inside a loaded skill. A link to a skill not
// loaded yet is followed through catalog.TryGetOrFetchByPathUp: with
// AddRelations off it is an "unselected-skill" issue, with it on the skill
// is loaded and then checked the same way. Web links are not checked, nor
// are files the catalog excludes from checks (catalog.IsExcludedFromChecks,
// e.g. "examples" holding a packaged example app).
type LinkValidator struct{}

// CatalogValidator is a validator of the skills loaded in a SkillCatalog.
type CatalogValidator = validator.Validator[*sourcing.SkillCatalog, issues.SkillIssue]

var _ CatalogValidator = LinkValidator{}

func (LinkValidator) Name() string { return "link-validator" }

func (LinkValidator) DependsOn() []validator.Dependency { return nil }

// Validate checks every loaded skill in load order, including the skills
// loaded while following links -- catalog.Skills() lists them after the
// skills loaded before, so walking it by index reaches them too.
func (v LinkValidator) Validate(ctx context.Context, catalog *sourcing.SkillCatalog) []issues.SkillIssue {
	var problems issues.SkillIssues
	shared := &sharedFiles{seen: map[string]bool{}}
	for i := 0; i < len(catalog.Skills()); i++ {
		if err := ctx.Err(); err != nil {
			return append(problems, issues.SkillIssue{Code: issues.CodeCanceled, Message: err.Error()})
		}
		problems = append(problems, v.validateSkill(ctx, catalog, catalog.Skills()[i], shared)...)
	}
	return append(problems, shared.warnings...)
}

func (v LinkValidator) validateSkill(ctx context.Context, catalog *sourcing.SkillCatalog, skill *entity.Skill, shared *sharedFiles) issues.SkillIssues {
	files, err := skill.FilesByPath("")
	if err != nil {
		return issues.SkillIssues{skillIssue(skill, issues.SkillIssue{Code: issues.CodeSourceRead, Message: err.Error()})}
	}
	var problems issues.SkillIssues
	for _, file := range append([]*entity.File{skill.MainFile}, files...) {
		rel, err := file.Path(model.SkillRelative)
		if err != nil {
			problems = append(problems, skillIssue(skill, issues.SkillIssue{Code: issues.CodeSourceRead, Message: err.Error()}))
			continue
		}
		if !strings.HasSuffix(strings.ToLower(rel), ".md") || catalog.IsExcludedFromChecks(skill, rel) {
			continue
		}
		links, err := file.Links()
		if err != nil {
			problems = append(problems, skillIssue(skill, asIssue(err, issues.SkillIssue{File: rel})))
			continue
		}
		for _, link := range links {
			if link.External {
				continue
			}
			if err := validateLink(ctx, catalog, link, shared); err != nil {
				problems = append(problems, skillIssue(skill, asIssue(err, issues.SkillIssue{File: rel, Link: link.Raw})))
			}
		}
	}
	return problems
}

// validateLink checks that link's target exists and lies in a loaded skill
// (loading it when the catalog adds relations) -- or in no skill at all: a
// shared file of the source (e.g. a catalog's registry entry), which sync
// copies next to the skills that link to it -- and holds the anchor the
// link names, if any. A link into a skill no source selects is
// "unselected-skill"; a link to a folder outside every skill is
// "external-folder", only files are copied. A shared markdown file that
// holds links of its own gets a warning, once (shared): they are copied as
// written, not rewritten.
func validateLink(ctx context.Context, catalog *sourcing.SkillCatalog, link *entity.Link, shared *sharedFiles) error {
	repoPath, err := link.Path(ctx, model.RepoAbsolute, catalog)
	if err != nil {
		return err
	}
	repo := link.File().Skill().Repo
	target, err := link.Skill(ctx, catalog)
	if errors.Is(err, entity.ErrSkillNotCached) {
		// Not loaded: some skill no source selects holds it, or none does.
		if _, upErr := catalog.FetchByPathUp(ctx, repo, repoPath); !isSkillNotFound(upErr) {
			return issues.SkillIssue{Code: issues.CodeUnselectedSkill, Message: "the link leads outside the selected skills and add_relations is off"}
		}
		err = issues.SkillIssue{Code: issues.CodeSkillNotFound}
	}
	var content []byte
	switch {
	case isSkillNotFound(err):
		info, statErr := fs.Stat(repo.FS, repoPath)
		if statErr != nil {
			return statErr
		}
		if info.IsDir() {
			return issues.SkillIssue{Code: issues.CodeExternalFolder, Message: fmt.Sprintf("%s is a folder outside every skill: only files outside skills are copied, link a file", repoPath)}
		}
		shared.check(repo, repoPath)
		if link.Fragment == "" || link.Fragment == "#" {
			return nil
		}
		if !strings.HasSuffix(strings.ToLower(repoPath), ".md") {
			return issues.SkillIssue{Code: issues.CodeMissingAnchor, Message: fmt.Sprintf("anchor %s is looked for only in markdown files, not in %s", link.Fragment, repoPath)}
		}
		if content, err = fs.ReadFile(repo.FS, repoPath); err != nil {
			return err
		}
		if !hasAnchor(string(content), link.Fragment) {
			return issues.SkillIssue{Code: issues.CodeMissingAnchor, Message: fmt.Sprintf("anchor %s is not in %s", link.Fragment, repoPath)}
		}
		return nil
	case err != nil:
		return err
	}
	if link.Fragment == "" || link.Fragment == "#" {
		return nil
	}
	rel, err := link.Path(ctx, model.SkillRelative, catalog)
	if err != nil {
		return err
	}
	if !strings.HasSuffix(strings.ToLower(rel), ".md") {
		return issues.SkillIssue{Code: issues.CodeMissingAnchor, Message: fmt.Sprintf("anchor %s is looked for only in markdown files, not in %s", link.Fragment, rel)}
	}
	if content, err = entity.MakeFile(rel, target).Content(); err != nil {
		return err
	}
	if !hasAnchor(string(content), link.Fragment) {
		return issues.SkillIssue{Code: issues.CodeMissingAnchor, Message: fmt.Sprintf("anchor %s is not in %s", link.Fragment, rel)}
	}
	return nil
}

// sharedFiles collects the warnings about the shared files the checked
// links lead to, one per file whatever number of links lead to it.
type sharedFiles struct {
	seen     map[string]bool
	warnings issues.SkillIssues
}

// check adds a warning listing the links a shared markdown file holds
// ("shared-file-links"), or saying they can't be read
// ("shared-file-unread"): sync copies the file as written, so where they
// lead in the target is up to the reader to check.
func (f *sharedFiles) check(repo *entity.Repository, repoPath string) {
	key := entity.GetSkillKey(repo.Key, repoPath)
	if f.seen[key] || !strings.HasSuffix(strings.ToLower(repoPath), ".md") {
		return
	}
	f.seen[key] = true
	content, err := fs.ReadFile(repo.FS, repoPath)
	if err != nil {
		return
	}
	parsed, err := entity.SearchLinks(content)
	if err != nil {
		f.warnings = append(f.warnings, issues.SkillIssue{Code: issues.CodeSharedFileUnread, Source: repo.Key.String(), File: repoPath, Message: fmt.Sprintf("shared file outside skills is copied as written, its links can't be read: %v", err)})
		return
	}
	if len(parsed) == 0 {
		return
	}
	raws := make([]string, 0, len(parsed))
	for _, l := range parsed {
		raws = append(raws, string(content[l.Start:l.End]))
	}
	f.warnings = append(f.warnings, issues.SkillIssue{Code: issues.CodeSharedFileLinks, Source: repo.Key.String(), File: repoPath, Message: "shared file outside skills is copied as written, check where its links lead:", Details: raws})
}

// isSkillNotFound reports whether err says no skill holds the path.
func isSkillNotFound(err error) bool {
	var issue issues.SkillIssue
	return errors.As(err, &issue) && issue.Code == issues.CodeSkillNotFound
}

// asIssue turns err into an Issue carrying at's File and Link: an Issue
// keeps its own Code and Message (its own File, if any, goes into the
// Message -- e.g. where a nested skill was found), any other error becomes
// a "link-target" Issue.
func asIssue(err error, at issues.SkillIssue) issues.SkillIssue {
	var issue issues.SkillIssue
	if !errors.As(err, &issue) {
		at.Code, at.Message = issues.CodeLinkTarget, err.Error()
		return at
	}
	at.Code, at.Message = issue.Code, issue.Message
	if issue.File != "" && issue.File != at.File && !strings.Contains(issue.Message, issue.File) {
		at.Message = issue.File + ": " + issue.Message
	}
	return at
}

// skillIssue fills issue's skill context: its source, name and location.
func skillIssue(skill *entity.Skill, issue issues.SkillIssue) issues.SkillIssue {
	issue.Source = skill.Repo.Key.String()
	issue.Skill = skill.Name
	issue.SkillPath = skill.DirOrMarkerPath()
	return issue
}

package validator

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/InsonusK/go-ai-skill-manage/internal/domain/entity"
	"github.com/InsonusK/go-ai-skill-manage/internal/domain/model"
	"github.com/InsonusK/go-ai-skill-manage/internal/domain/services/sourcing"
	"github.com/InsonusK/go-ai-skill-manage/internal/domain/services/validators"
)

// LinkValidator checks that every link in the markdown files of the loaded
// skills leads to an existing file -- and, when it names an anchor, to an
// existing anchor in it -- inside a loaded skill. A link to a skill not
// loaded yet is followed through catalog.TryGetOrFetchByPathUp: with
// AddRelations off it is an "unselected-skill" issue, with it on the skill
// is loaded and then checked the same way. Web links are not checked, nor
// are files under SkipFolders.
type LinkValidator struct {
	// SkipFolders are top-level folders of a skill whose files' links are
	// not checked, e.g. "examples" holding a packaged example app.
	SkipFolders []string
}

var _ validators.Validator = LinkValidator{}

func (LinkValidator) Name() string { return "link-validator" }

func (LinkValidator) DependsOn() []validators.Dependency { return nil }

// Validate checks every loaded skill in load order, including the skills
// loaded while following links -- catalog.Skills() lists them after the
// skills loaded before, so walking it by index reaches them too.
func (v LinkValidator) Validate(ctx context.Context, catalog *sourcing.SkillCatalog) model.Issues {
	var issues model.Issues
	for i := 0; i < len(catalog.Skills()); i++ {
		if err := ctx.Err(); err != nil {
			return append(issues, model.Issue{Code: "canceled", Message: err.Error()})
		}
		issues = append(issues, v.validateSkill(ctx, catalog, catalog.Skills()[i])...)
	}
	return issues
}

func (v LinkValidator) validateSkill(ctx context.Context, catalog *sourcing.SkillCatalog, skill *entity.Skill) model.Issues {
	files, err := skill.FilesByPath("")
	if err != nil {
		return model.Issues{skillIssue(skill, model.Issue{Code: "source-read", Message: err.Error()})}
	}
	var issues model.Issues
	for _, file := range append([]*entity.File{skill.MainFile}, files...) {
		rel, err := file.Path(model.SkillRelative)
		if err != nil {
			issues = append(issues, skillIssue(skill, model.Issue{Code: "source-read", Message: err.Error()}))
			continue
		}
		if !strings.HasSuffix(strings.ToLower(rel), ".md") || v.skipped(rel) {
			continue
		}
		links, err := file.Links()
		if err != nil {
			issues = append(issues, skillIssue(skill, asIssue(err, model.Issue{File: rel})))
			continue
		}
		for _, link := range links {
			if link.External {
				continue
			}
			if err := validateLink(ctx, catalog, link); err != nil {
				issues = append(issues, skillIssue(skill, asIssue(err, model.Issue{File: rel, Link: link.Raw})))
			}
		}
	}
	return issues
}

// validateLink checks that link's target file exists, lies in a loaded
// skill (loading it when the catalog adds relations), and holds the anchor
// the link names, if any.
func validateLink(ctx context.Context, catalog *sourcing.SkillCatalog, link *entity.Link) error {
	if _, err := link.Path(ctx, model.RepoAbsolute, catalog); err != nil {
		return err
	}
	target, err := link.Skill(ctx, catalog)
	if errors.Is(err, entity.ErrSkillNotCached) {
		return model.Issue{Code: "unselected-skill", Message: "the link leads outside the selected skills and add_relations is off"}
	}
	if err != nil {
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
		return model.Issue{Code: "missing-anchor", Message: fmt.Sprintf("anchor %s is looked for only in markdown files, not in %s", link.Fragment, rel)}
	}
	content, err := entity.MakeFile(rel, target).Content()
	if err != nil {
		return err
	}
	if !hasAnchor(string(content), link.Fragment) {
		return model.Issue{Code: "missing-anchor", Message: fmt.Sprintf("anchor %s is not in %s", link.Fragment, rel)}
	}
	return nil
}

func (v LinkValidator) skipped(rel string) bool {
	first := strings.SplitN(rel, "/", 2)[0]
	for _, skip := range v.SkipFolders {
		if first == skip {
			return true
		}
	}
	return false
}

// asIssue turns err into an Issue carrying at's File and Link: an Issue
// keeps its own Code and Message (its own File, if any, goes into the
// Message -- e.g. where a nested skill was found), any other error becomes
// a "link-target" Issue.
func asIssue(err error, at model.Issue) model.Issue {
	var issue model.Issue
	if !errors.As(err, &issue) {
		at.Code, at.Message = "link-target", err.Error()
		return at
	}
	at.Code, at.Message = issue.Code, issue.Message
	if issue.File != "" && issue.File != at.File && !strings.Contains(issue.Message, issue.File) {
		at.Message = issue.File + ": " + issue.Message
	}
	return at
}

// skillIssue fills issue's skill context: its source, name and location.
func skillIssue(skill *entity.Skill, issue model.Issue) model.Issue {
	issue.Source = skill.Repo.Key.String()
	issue.Skill = skill.Name
	issue.SkillPath = skill.DirOrMarkerPath()
	return issue
}

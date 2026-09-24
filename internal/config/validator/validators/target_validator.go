package validators

import (
	"context"
	"fmt"

	"github.com/InsonusK/go-ai-skill-manage/internal/domain/model"
	"github.com/InsonusK/go-ai-skill-manage/internal/domain/model/issues"
	"github.com/InsonusK/go-ai-skill-manage/internal/domain/services/validator"
)

// TargetValidator checks that no two targets write into the same folder.
type TargetValidator struct{}

var _ ConfigValidator = TargetValidator{}

func (TargetValidator) Name() string { return "target-validator" }

func (TargetValidator) DependsOn() []validator.Dependency { return nil }

// Validate reports "target-overlap" on the later of two targets whose
// paths are the same or one lies inside the other.
//
// Примеры: "/p/out" и "/p/out" -> target-overlap; "/p/out" и
// "/p/out/nested" -> target-overlap; "/p/out" и "/p/out2" -> ok.
func (TargetValidator) Validate(ctx context.Context, req model.Request) []issues.ConfigIssue {
	var problems []issues.ConfigIssue
	for i, a := range req.Targets {
		for j, b := range req.Targets[i+1:] {
			if within(a.Path, b.Path) || within(b.Path, a.Path) {
				problems = append(problems, issues.ConfigIssue{Code: "target-overlap", Setting: fmt.Sprintf("targets[%d].path", i+1+j), Message: fmt.Sprintf("target %q path %s overlaps target %q path %s", b.Name, b.Path, a.Name, a.Path)})
			}
		}
	}
	return problems
}

// Package validator checks a resolved configuration with every config
// validator (see package validators). It is the only place a
// configuration is checked for meaning; config.Parse only checks its shape.
package validator

import (
	"context"

	"github.com/InsonusK/go-ai-skill-manage/internal/config/validator/validators"
	"github.com/InsonusK/go-ai-skill-manage/internal/domain/model"
	"github.com/InsonusK/go-ai-skill-manage/internal/domain/model/issues"
	domainvalidator "github.com/InsonusK/go-ai-skill-manage/internal/domain/services/validator"
)

// manager runs every config validator, in this order.
var manager = mustManager(
	validators.TagsValidator{},
	validators.SubpathValidator{},
	validators.SourceValidator{},
	validators.TargetValidator{},
)

// Validate checks req, resolved by config.Resolve, and returns every
// problem found. Code after it trusts req: a request with problems must
// not go further.
func Validate(ctx context.Context, req model.Request) issues.ConfigIssues {
	return manager.Validate(ctx, req)
}

func mustManager(list ...validators.ConfigValidator) *domainvalidator.Manager[model.Request, issues.ConfigIssue] {
	m, err := domainvalidator.NewManager(list...)
	if err != nil {
		// The list above is code, not configuration.
		panic(err)
	}
	return m
}

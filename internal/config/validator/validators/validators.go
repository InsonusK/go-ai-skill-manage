// Package validators holds the checks of a resolved configuration
// (model.Request). They run after config.Resolve -- paths are absolute --
// and are the only place a configuration is checked for meaning: code
// after them trusts the request and treats a violation as a bug.
package validators

import (
	"github.com/InsonusK/go-ai-skill-manager/internal/domain/model"
	"github.com/InsonusK/go-ai-skill-manager/internal/domain/model/issues"
	"github.com/InsonusK/go-ai-skill-manager/internal/domain/services/validator"
)

// ConfigValidator is a validator of a resolved configuration.
type ConfigValidator = validator.Validator[model.Request, issues.ConfigIssue]

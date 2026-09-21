// Package planning computes output layouts and changes without writing
// targets, consuming a model.SkillCatalog (which already indexes each
// skill's output destinations) and an interfaces.RepositoryLookup rather
// than re-deriving ownership on every target.
package planning

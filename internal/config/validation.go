package config

import (
	"fmt"
	"log/slog"

	"github.com/InsonusK/go-ai-skill-manage/internal/domain/model"
)

// globalExcludeFromChecks reads settings.validation.exclude_from_checks --
// or its deprecated name settings.validation.rules.link.skip_folder, with a
// warning. Neither set means model.DefaultExcludeFromChecks, with an info
// message; set to [] or left empty it excludes nothing. Both set is an
// error.
//
// Примеры:
//   - раздела нет                                -> ["examples"] + info
//   - exclude_from_checks: [demo]                -> ["demo"]
//   - exclude_from_checks: [] (или без значения) -> []
//   - rules.link.skip_folder: demo               -> ["demo"] + warning
func globalExcludeFromChecks(settings map[string]any) ([]string, error) {
	const name, legacyName = "settings.validation.exclude_from_checks", "settings.validation.rules.link.skip_folder"
	validation, err := mapping(settings["validation"], "validation")
	if err != nil {
		return nil, err
	}
	raw, hasNew := validation["exclude_from_checks"]
	legacy, hasLegacy, err := legacyLinkSkipFolder(validation)
	if err != nil {
		return nil, err
	}
	switch {
	case hasNew && hasLegacy:
		return nil, fmt.Errorf("%s cannot be defined both with deprecated %s", name, legacyName)
	case hasLegacy:
		slog.Warn("deprecated configuration key, rename it", "key", legacyName, "use", name)
		return listValue(legacy, "skip_folder", nil)
	case hasNew:
		return listValue(raw, "exclude_from_checks", nil)
	}
	slog.Info("configuration key is not set, using the default; set it to [] to check every folder", "key", name, "default", model.DefaultExcludeFromChecks)
	return append([]string{}, model.DefaultExcludeFromChecks...), nil
}

// legacyLinkSkipFolder finds validation.rules.link.skip_folder, if set.
func legacyLinkSkipFolder(validation map[string]any) (any, bool, error) {
	current := validation
	for _, key := range []string{"rules", "link"} {
		raw, exists := current[key]
		if !exists || raw == nil {
			return nil, false, nil
		}
		var err error
		if current, err = mapping(raw, key); err != nil {
			return nil, false, err
		}
	}
	raw, exists := current["skip_folder"]
	return raw, exists, nil
}

// sourceExcludeFromChecks reads a source's exclude_from_checks -- or its
// deprecated name skip_folder, with a warning. Neither set excludes nothing
// beyond the global settings.validation.exclude_from_checks. Both set is an
// error.
func sourceExcludeFromChecks(m map[string]any, source string) ([]string, error) {
	raw, hasNew := m["exclude_from_checks"]
	legacy, hasLegacy := m["skip_folder"]
	switch {
	case hasNew && hasLegacy:
		return nil, fmt.Errorf("source %s: exclude_from_checks cannot be defined both with deprecated skip_folder", source)
	case hasLegacy:
		slog.Warn("deprecated configuration key, rename it", "source", source, "key", "skip_folder", "use", "exclude_from_checks")
		return listValue(legacy, "skip_folder", nil)
	}
	return listValue(raw, "exclude_from_checks", nil)
}

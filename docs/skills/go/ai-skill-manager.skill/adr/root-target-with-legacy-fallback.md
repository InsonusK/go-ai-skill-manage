---
name: root-target-with-legacy-fallback
description: Place target configuration at the YAML root without silently breaking existing files.
problem: Choose how the CLI migrates target from settings.target to the root target key.
decision: Use root target as canonical, accept settings.target only as a fallback, and reject both together.
tags:
  - stack/go
  - app-type/cli
  - concern/documentation
  - concern/documentation/adr
---

# Problem

The configuration schema needs `target` beside `sources` while existing files may still define `settings.target`.

# Selected variant

[Root target with legacy fallback](#root-target-with-legacy-fallback-selected).

# Searched variants

## Root target with legacy fallback (selected)

### Description

Read root `target` as canonical, read `settings.target` only when the root key is absent, and reject configurations containing both.

### Benefits

- Existing configuration files continue to work.
- New files use the clearer root-level structure.
- Ambiguous files fail instead of silently selecting one target definition.

### Costs

- The parser temporarily supports two locations.
- Legacy removal requires a later deprecation decision.

## Strict root target only

### Description

Read only root `target` and reject `settings.target`.

### Benefits

- The parser and schema have one representation immediately.
- No legacy branch remains to maintain.

### Costs

- Every existing configuration must migrate before upgrading.
- A strict validation rule is required to prevent the old key from being silently ignored.

## Keep settings target

### Description

Leave `target` under `settings`.

### Benefits

- No migration or compatibility code is needed.

### Costs

- The configuration hierarchy remains less direct.
- The requested root-level schema is not delivered.


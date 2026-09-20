---
name: ai-skill-manager
description: Build and invoke the Go AI Skill Manager CLI to synchronize local or GitHub skills into project targets.
whenToUse: when invoking aism sync, preparing its configuration, or interpreting its exit status
updated: 20260920
tags:
  - stack/go
  - app-type/cli
  - concern/documentation
adr:
  - "[Single CLI skill](./adr/single-cli-skill.md)"
  - "[Root target with legacy fallback](./adr/root-target-with-legacy-fallback.md)"
---

# Goal

- A reproducible CLI invocation with explicit source and target settings.
- A checked synchronization result or actionable failure.

# Core Principle

- Resolve configuration paths relative to the configuration file.
- Use root `target` with the [documented legacy fallback](./adr/root-target-with-legacy-fallback.md).
- Treat directories carrying `.ai-skills-managed` as owned by this CLI.
- Use this single skill for the command's coherent synchronization surface.

# Workflow

1. Follow [installation.md](./installation.md) to obtain the binary.
2. Follow [method-sync.md](./method-sync.md) to prepare options and invoke synchronization.
3. Inspect the exit code and output before reporting completion.

# Rule

## MUST

### Preserve configuration context

Pass `--config /absolute/path/ai-skills.yaml` when operating outside the project's directory.
- Risk: implicit loading selects a different file or resolves direct paths against the wrong directory.
- Fix: choose the intended configuration explicitly.

### Check managed removals

Use the orphan policy described in [method-sync.md](./method-sync.md) when constructing a command.
- Risk: an empty selection can remove previously managed skills.
- Fix: use `--keep-orphans` when removals are outside the requested operation.

### Report failures

Interpret the exit codes in [method-sync.md](./method-sync.md) before claiming success.
- Risk: a partially applied I/O failure is mistaken for completed synchronization.
- Fix: include the failing target or source from stderr and rerun after resolving the cause.

# Check list

- [ ] Use the intended binary and configuration.
- [ ] Select source, target and orphan policy explicitly where needed.
- [ ] Check exit code and final summary.

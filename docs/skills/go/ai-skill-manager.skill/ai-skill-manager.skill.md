---
name: ai-skill-manager
description: Install, configure and run the aism CLI, which checks AI skills from local folders or GitHub repositories and writes them into a project's skill folders (.agents/skills, .claude/skills, ...).
whenToUse: when installing aism, writing or changing an ai-skills.yaml, checking skills with `aism validate`, copying skills into a project with `aism sync`, or reading aism's output and exit code
updated: 20260927
tags:
  - stack/go
  - app-type/cli
  - concern/documentation
adr:
  - "[Single CLI skill](./adr/single-cli-skill.md)"
  - "[Root target with legacy fallback](./adr/root-target-with-legacy-fallback.md)"
---

# Goal

- `aism` installed and answering `aism --version`.
- An `ai-skills.yaml` that names the skill sources and the target folders.
- The skills checked with `aism validate`, then written with `aism sync`, and the result confirmed from the exit code and output.

# Core Principle

- `aism` never writes anything while a problem remains: any configuration, skill or target problem stops it before the first write, with exit code 1 and a problem tree on stderr.
- A target skill folder that holds `.ai-skills-managed` belongs to `aism`: every `sync` rewrites it as a whole and, with `remove_orphans`, deletes it when its skill is gone. A folder without that marker is the user's and is never touched.
- Paths in `ai-skills.yaml` are relative to the folder that holds `ai-skills.yaml`.

# Installation and access

See [installation.md](./installation.md) for the install commands, prerequisites and a verification call.

# Workflow

1. Install `aism` ([installation.md](./installation.md)).
2. Write `ai-skills.yaml` next to the project ([configuration.md](./configuration.md)).
3. Run `aism validate` and fix every reported problem ([method-validate.md](./method-validate.md)).
4. Run `aism sync --dry-run` and read the planned operations ([method-sync.md](./method-sync.md)).
5. Run `aism sync` and check the exit code and the final `Synced N skill(s) to M target(s)` line.

# Rule

## MUST

### Validate before syncing

Run `aism validate` (and `aism sync --dry-run` when the target already holds skills) before `aism sync`.
- Risk: a first `sync` into a populated target plans `remove` for managed skills no longer in the sources, and the user sees the deletion only after it happened.
- Fix: read the dry-run's `remove <name>` lines; pass `--keep-orphans` when deletions are not part of the task.

### Pass the config explicitly outside its folder

Run `aism <command> --config /path/to/ai-skills.yaml` whenever the current directory is not the one holding `ai-skills.yaml`.
- Risk: without `--config`, `aism` reads `ai-skills.yaml` from the current directory and fails with "no such file", or reads another project's file.
- Fix: pass the intended file; its folder is the base for every relative path in it.

### Never delete a user's folder to clear `unmanaged-target`

Report `unmanaged-target` to the user and let them decide.
- Risk: the folder of that name was written by the user, not by `aism`; deleting it loses their skill.
- Fix: rename the skill in its source frontmatter, point the target elsewhere, or have the user remove the folder.

### Report problems from the tree, not from the summary

On exit code 1, read the problem tree on stderr and report each problem with its place (source, skill, file, link or target) and code.
- Risk: "sync failed" without the codes gives the user nothing to fix.
- Fix: map each code to its fix with the table in [method-validate.md](./method-validate.md#problem-codes).

# Check list

- [ ] `aism --version` prints a version.
- [ ] `aism validate` exits 0.
- [ ] The dry-run's `remove` lines were expected, or `--keep-orphans` was passed.
- [ ] `aism sync` exited 0 and printed `Synced N skill(s) to M target(s)`.

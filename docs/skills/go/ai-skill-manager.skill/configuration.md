# Configuration: `ai-skills.yaml`

Both commands read `ai-skills.yaml` from the current directory, or the file given with `--config`. JSON with the same keys works too. Every relative path in it is resolved from the file's folder. See the [annotated example](./examples/ai-skills.yaml) for every key.

## Minimal configuration

```yaml
sources:
  - path: ./my-skills
target:
  default: {}                        # writes into .agents/skills
  claude:                            # writes into .claude/skills
    adapters: [claude-property-adapter]
```

## `sources[]` — where skills come from

| Key | Type | Required | Default | Meaning |
| --- | --- | --- | --- | --- |
| `path` | string | yes | — | Local folder, or Git URL when `type: github`. |
| `type` | `local` \| `github` | no | `local` | Source kind. |
| `tree` | string | no | `master` | Git branch or tag (`github` only). |
| `subpath` | string or list | no | `github`: `[skills]`; `local`: the source folder | Folders (or single `.skill.md` files) inside the source to take skills from; every skill at or below them is taken. |
| `exclude_from_checks` | string or list | no | none | Top-level folders of each skill that are copied but not checked; added to `settings.validation.exclude_from_checks`. |

A skill is one of:
- a folder with `SKILL.md`;
- a folder `{name}.skill` with `{name}.skill.md`;
- a single file `{name}.skill.md`.

Its name is the `name` in its YAML frontmatter: lowercase letters, digits and single or double hyphens, e.g. `code-review`.

`tags` in a source is not supported yet: validation reports `unsupported-tags`. Remove it.

## `target` — where skills are written

- A string (`target: .agents/skills`) writes into one folder with no adapters.
- A mapping names the targets: `default` (path `.agents/skills` unless given), `claude` (path `.claude/skills` unless given), or any other name with a required `path`.
- `for_each.adapters` adds adapters to every named target.
- The only adapter is `claude-property-adapter`: it renames the frontmatter key `whenToUse` to `when_to_use`, the key Claude Code reads. Use it for Claude Code targets.
- Two targets must not share a folder or nest one in the other (`target-overlap`).

## `settings`

| Key | Type | Default | Meaning |
| --- | --- | --- | --- |
| `dry_run` | bool | `false` | Plan and print, write nothing (same as `--dry-run`). |
| `remove_orphans` | bool | `true` | Delete managed target folders whose skill is no longer in the sources. |
| `add_relations` | bool | `false` | Also load skills that the selected skills link to; with `false` such a link is the problem `unselected-skill`. |
| `temp_dir` | path | system temp folder | Existing, writable folder for downloaded GitHub sources. |
| `validation.exclude_from_checks` | string or list | `[examples, templates]` | Top-level skill folders copied but not checked (example apps, templates with placeholder links); `[]` checks everything. |

## Deprecated keys

They are accepted, have no effect, and log a warning on stderr — remove them:
- `sources[].name`;
- `sources[].skip_folder` (renamed `exclude_from_checks`);
- `settings.validation.rules.link.skip_folder` (renamed `settings.validation.exclude_from_checks`);
- `settings.on_conflict`: duplicate skill names are always the problem `duplicate-name`;
- the adapter `link-adapter`: links are always rewritten.

# `sync`

Checks the skills like [`validate`](./method-validate.md), then writes every skill into every target folder.

## Signature

```sh
aism [--debug] [--profile] sync [--config FILE | --type local|github --path SOURCE [--subpath PATH]...] [--target PATH] [--dry-run] [--remove-orphans | --keep-orphans] [--add-relations]
```

## Parameters

| Option | Type | Required | Default | Meaning |
| --- | --- | --- | --- | --- |
| `-c, --config` | path | no | `./ai-skills.yaml` | Configuration file ([configuration.md](./configuration.md)). |
| `-t, --type` | `local` \| `github` | no | — | Sync one source without a config file. |
| `-p, --path` | string | with `--type` | — | Local folder, or `"URL branch"` for `github`. |
| `--subpath` | path, repeatable | no | `skills` for `github` | Folders inside the `github` source. |
| `--target` | path | no | configured targets | Write into this one folder instead, with no adapters. |
| `--dry-run` | bool | no | `settings.dry_run` | Plan and print, write nothing. |
| `--remove-orphans` | bool | no | `settings.remove_orphans` (`true`) | Delete managed folders whose skill is gone. |
| `--keep-orphans` | bool | no | — | Keep them; `--remove-orphans` wins if both are given. |
| `--add-relations` | bool | no | `settings.add_relations` | Also load skills that selected skills link to. |
| `--profile`, `--profile-output` | bool, path | no | `false`, `ai-skill-manager.prof` | Write a Go CPU profile. |
| `-f, --force` | bool | no | — | Deprecated, no effect (warning). |

## What it writes

For every target and every skill `{name}`:

- `{target}/{name}/SKILL.md` — the skill's marker file, whatever its source form (`SKILL.md`, `{name}.skill.md`);
- `{target}/{name}/...` — the skill's other files at their paths inside the skill folder, with their file modes;
- `{target}/{name}/.ai-skills-managed` — JSON with the source, the skill's path there and the applied transformations.

Links in `.md` files are rewritten to the new places (`./x`, `../other/SKILL.md`), and wikilinks `[[...]]` become markdown links. For targets with `claude-property-adapter`, the frontmatter key `whenToUse` becomes `when_to_use`.

Per folder in the target:
- missing → `create`;
- has `.ai-skills-managed` → `update` (the folder is replaced as a whole);
- no marker → the problem `unmanaged-target`, nothing is written;
- managed but its skill is gone → `remove` (with `remove_orphans`).

All targets are planned before the first write.

## Output

- Exit 0, stdout: the operations per target, then the summary:

  ```text
  Target default: /project/.agents/skills
    create code-review
    update guide
    remove old-skill
  Synced 2 skill(s) to 1 target(s)
  ```

  With `--dry-run` the summary is `Dry run: 2 skill(s), 1 target(s); nothing written`.
- Exit 1: a problem tree on stderr, as for `validate`, plus the target problem `unmanaged-target`:

  ```text
  target /project/.agents/skills
    skill guide
      unmanaged-target: /project/.agents/skills/guide exists but was not written by this tool (no .ai-skills-managed): remove or rename it
  Found 1 problem(s)
  ```

  Exit 1 is also returned when writing fails (disk, permissions). Targets are written one after another, not as one transaction, so earlier targets may already be updated; rerun `sync` after fixing the cause.
- Exit 2: invalid arguments.

## Examples

```sh
aism sync --dry-run                      # plan with ./ai-skills.yaml
aism sync                                # write
aism sync --config /project/ai-skills.yaml --keep-orphans
aism sync --type local --path ./my-skills --target .agents/skills
aism sync --type github --path "https://github.com/InsonusK/ai-skills.git master" --subpath skills/go --target .agents/skills
```

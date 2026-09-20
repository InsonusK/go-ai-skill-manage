# sync

Signature: `aism [--debug] [--profile] sync [options]`.

## Required input

Use either a configuration file or `--type local|github --path SOURCE`.
Without these options, the CLI reads `ai-skills.yaml` in the current directory.

```yaml
sources:
  - type: local
    path: ./skills
target: .agents/skills
settings:
  remove_orphans: true
```

```sh
./bin/aism sync --config ./ai-skills.yaml --dry-run
./bin/aism sync --config ./ai-skills.yaml
```

Use the [complete annotated ai-skills.yaml](./examples/ai-skills.yaml) to see every supported field, whether it is required, its default, and its effect.

## Options

| Option | Type / default | Effect |
| --- | --- | --- |
| `--config, -c` | file / ai-skills.yaml | YAML or JSON; takes precedence over direct source flags |
| `--type, -t` | local or github | Direct source mode |
| `--path, -p` | string | Local path or quoted `URL branch` |
| `--subpath` | repeatable string / skills for GitHub | GitHub scan paths |
| `--target` | path | Replace all configured targets with one |
| `--dry-run` | bool / false | Validate and plan without target writes |
| `--force, -f` | bool / false | Recopy unchanged managed skills |
| `--remove-orphans` | bool / config default true | Delete managed skills missing from selection |
| `--keep-orphans` | bool | Disable orphan removal; remove wins if both are set |
| `--add-relations` | bool / config default false | Include referenced skills |
| `--debug` | bool / false | Structured debug logs on stderr |
| `--profile` | bool / false | CPU profiling |
| `--profile-output` | path / ai-skill-manager.prof | Go pprof output |

Configuration sources support `tree` (master), `subpath` (string/list),
`tags` (AND of expressions), `skip_folder` (examples), and `name`
(single selected skill only). Local relative paths resolve from the config
directory. Root `target` accepts a string path or named targets; legacy
`settings.target` remains accepted when the root key is absent. Settings support
`temp_dir`, `dry_run`, `remove_orphans`, `add_relations`,
`on_conflict: error|last_wins`, and validation rules.
A relative `temp_dir` resolves from the configuration file; omit it to use the operating system temp directory.
Claude requires `adapters: [claude-property-adapter]`.
Full schema: [reference](../../../api/reference.md).

## Output and failures

Stdout lists target operations: create, update, skip, remove, followed by e.g.
`Synced 1 skill(s) to 1 target(s)`.
Dry-run ends with `no target changes`; source acquisition may use temporary files.
Stderr carries diagnostics. Exit 0: success/help/version; 1: configuration,
validation, acquisition or I/O failure; 2: invalid arguments.

For `unselected-skill`, include the source or enable relations when intended.
For `missing-link`, fix the source path.
For `unmanaged-target`, choose another target or resolve its existing contents.
Do not delete personal directories merely to suppress the error.

All targets are planned before writes; writes across targets are not one
transaction. A failed write may follow successfully updated earlier targets.
Repeat with `--force` after repair if managed output needs rebuilding.

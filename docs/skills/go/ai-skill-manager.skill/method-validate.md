# `validate`

Checks the configuration and every skill it selects, and writes nothing.

## Signature

```sh
aism [--debug] validate [--config FILE | --type local|github --path SOURCE [--subpath PATH]...] [--add-relations]
```

## Parameters

| Option | Type | Required | Default | Meaning |
| --- | --- | --- | --- | --- |
| `-c, --config` | path | no | `./ai-skills.yaml` | Configuration file ([configuration.md](./configuration.md)). |
| `-t, --type` | `local` \| `github` | no | — | Check one source without a config file. |
| `-p, --path` | string | with `--type` | — | Local folder, or `"URL branch"` for `github`. |
| `--subpath` | path, repeatable | no | `skills` for `github` | Folders inside the `github` source. |
| `--add-relations` | bool | no | config value | Also load and check linked skills. |
| `--debug` | bool | no | `false` | Debug logs on stderr. |

## Output

- Exit 0, stdout: `Checked N skill(s): no problems`.
- Exit 1, stderr: every problem as a tree by place, then `Found N problem(s)`. Info and warning log lines (`level=INFO`, `level=WARN`) may precede it on stderr.
- Exit 2: invalid arguments (unknown flag or command, missing flag value).

```text
source local:/project/my-skills@master
  skill code-review (code-review)
    file SKILL.md
      link [the checklist](./docs/checklist.md)
        missing-link-target: code-review/docs/checklist.md: link target does not exist
Found 1 problem(s)
```

## Problem codes

| Code | Cause | Fix |
| --- | --- | --- |
| `unsupported-tags`, `invalid-tags` | `tags` in a source | Remove `tags`. |
| `unsafe-subpath` | `subpath` leads out of the source | Use a path inside the source. |
| `duplicate-source`, `conflicting-exclude` | Two sources of one repository repeat each other or differ in `exclude_from_checks` | Merge them, or give both the same list. |
| `target-overlap` | Two targets share or nest folders | Give each target its own folder. |
| `source-acquire` | The source can't be read or downloaded | Check the path, URL, branch and network. |
| `missing-subpath` | A `subpath` doesn't exist in the source | Fix or remove it. |
| `invalid-name`, `invalid-skill`, `pattern-conflict` | Bad frontmatter `name`, or two marker files in one folder | Fix the skill. |
| `nested-skill` | A skill folder holds another skill | Move it out, or add its folder to `exclude_from_checks`. |
| `duplicate-name` | Two selected skills have one name | Rename one in its frontmatter, or drop one source. |
| `missing-link-target`, `path-escape` | A link leads nowhere, or out of the repository | Fix the link. |
| `missing-anchor` | The linked heading/anchor doesn't exist | Fix the anchor. |
| `unselected-skill` | A link leads into a skill that no source selects | Add that skill's folder to `subpath`, or set `add_relations: true`. |
| `external-folder` | A link leads to a folder that no skill holds (only files outside skills are copied) | Link a file in it, or move the folder into a skill. |

## Example

```sh
cd /project
aism validate
echo "exit=$?"
```

Expected output for a valid project:

```text
Checked 3 skill(s): no problems
exit=0
```

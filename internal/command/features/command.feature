Feature: Command line synchronization
 Scenario: Sync a local directory from configuration
  Given CLI project
   """
   {"ai-skills.yaml":"sources:\n  - path: input\ntarget: output\n","input/a.skill.md":"---\nname: a\n---\nContent"}
   """
  When I run arguments "sync"
  Then exit code is "0"
  And stdout contains "Synced 1 skill"
  And project file "output/a/SKILL.md" contains "Content"
 Scenario: Config dry run does not create targets
  Given CLI project
   """
   {"ai-skills.yaml":"sources:\n  - path: input\ntarget: output\nsettings:\n  dry_run: true\n","input/a.skill.md":"---\nname: a\n---\nContent"}
   """
  When I run arguments "sync"
  Then exit code is "0"
  And stdout contains "Dry run"
  And project path "output" exists "false"
 Scenario: Direct mode skips config loading
  Given CLI project
   """
   {"input/a.skill.md":"---\nname: a\n---\nContent"}
   """
  When I run arguments "sync --type local --path input --target output"
  Then exit code is "0"
  And project file "output/a/SKILL.md" contains "Content"
 Scenario Outline: Usage and failures have stable exit codes
  Given CLI project
   """
   {}
   """
  When I run arguments "<args>"
  Then exit code is "<code>"
  And console contains "<message>"
  Examples:
   | args | code | message |
   | --help | 0 | Usage: |
   | sync --help | 0 | --add-relations |
   | --version | 0 | test-version |
   | sync | 1 | ai-skills.yaml |
   | sync --bad | 2 | unknown flag |
   | check | 2 | unknown command |
   | sync --type local | 1 | --path |
   | sync --config | 2 | requires a value |

 Scenario: Named targets share dependencies with different adapters
  Given CLI project
   """
   {"ai-skills.yaml":"sources:\n  - path: input\n    subpath: a.skill.md\ntarget:\n  for_each:\n    adapters: [link-adapter]\n  default:\n    path: out\n  claude:\n    path: claude\n    adapters: [claude-property-adapter]\nsettings:\n  add_relations: true\n","input/a.skill.md":"---\nname: a\ntags: [go]\n---\n[[b.skill.md#part|Read]]","input/b.skill.md":"---\nname: b\n---\nBody"}
   """
  When I run arguments "sync"
  Then exit code is "0"
  And project file "out/a/SKILL.md" contains "[Read](out/b/SKILL.md#part)"
  And project file "claude/a/SKILL.md" contains "[Read](claude/b/SKILL.md#part)"
  And project file "claude/a/SKILL.md" contains "## Metadata"
  When I run arguments "sync"
  Then exit code is "0"
  And stdout contains "(unchanged)"
  When I run arguments "sync -f"
  Then exit code is "0"
  And stdout contains "(forced)"
 Scenario: Tag filter and name override select one skill
  Given CLI project
   """
   {"ai-skills.yaml":"sources:\n  - path: input\n    tags: [go]\n    name: renamed\ntarget: out\n","input/a.skill.md":"---\nname: a\ntags: [go]\n---\nBody","input/b.skill.md":"---\nname: b\ntags: [python]\n---\n"}
   """
  When I run arguments "sync"
  Then exit code is "0"
  And project file "out/renamed/SKILL.md" contains "name: renamed"
  And project path "out/b" exists "false"
 Scenario: Last source wins a configured name conflict
  Given CLI project
   """
   {"ai-skills.yaml":"sources:\n  - path: first\n  - path: second\ntarget: out\nsettings:\n  on_conflict: last_wins\n","first/a.skill.md":"---\nname: same\n---\nFirst","second/a.skill.md":"---\nname: same\n---\nSecond"}
   """
  When I run arguments "sync"
  Then exit code is "0"
  And project file "out/same/SKILL.md" contains "Second"
 Scenario: Duplicate names block writes by default
  Given CLI project
   """
   {"ai-skills.yaml":"sources:\n  - path: input\ntarget: out\n","input/a.skill.md":"---\nname: same\n---\n","input/b.skill.md":"---\nname: same\n---\n"}
   """
  When I run arguments "sync"
  Then exit code is "1"
  And console contains "duplicate-name"
  And project path "out" exists "false"
 Scenario: Human directory attachments and external folders are copied
  Given CLI project
   """
   {"ai-skills.yaml":"sources:\n  - path: input\ntarget: out\n","input/a.skill/a.skill.md":"---\nname: a\n---\n[own](./notes.txt) [ext](assets)","input/a.skill/notes.txt":"Own","input/assets/image.png":"Image","input/assets/nested/info.txt":"Info"}
   """
  When I run arguments "sync"
  Then exit code is "0"
  And project file "out/a/notes.txt" contains "Own"
  And project file "out/files/assets/nested/info.txt" contains "Info"
  And project file "out/a/SKILL.md" contains "[own](out/a/notes.txt) [ext](out/files/assets)"
 Scenario: Missing subpaths produce an empty selection
  Given CLI project
   """
   {"ai-skills.yaml":"sources:\n  - path: input\n    subpath: missing\ntarget: out\n","input/readme.txt":"ordinary"}
   """
  When I run arguments "sync --keep-orphans"
  Then exit code is "0"
  And stdout contains "Synced 0 skill"

 Scenario: Whitespace GitHub path is a configuration error
  Given CLI project
   """
   {}
   """
  When I run argv
   """
   ["sync","--type","github","--path","   "]
   """
  Then exit code is "1"
  And console contains "--path is required"

Feature: Command line: sync and validate

 Scenario: Sync writes every skill into every target, laid out and transformed for it
  Given CLI project
   """
   {"ai-skills.yaml":"sources:\n  - path: skills\ntarget:\n  default: {}\n  claude:\n    adapters: [claude-property-adapter]\n",
    "skills/review/SKILL.md":"---\nname: review\nwhenToUse: on pull request\n---\nSee [guide](../guide.skill/guide.skill.md)\n",
    "skills/guide.skill/guide.skill.md":"---\nname: guide\n---\n[[review/SKILL.md|Review]]\n"}
   """
  When I run arguments "sync"
  Then exit code is "0"
  And stdout contains "Target default: "
  And stdout contains "  create guide\n  create review\n"
  And stdout contains "Synced 2 skill(s) to 2 target(s)"
  And project file ".agents/skills/review/SKILL.md" contains "whenToUse: on pull request\n---\nSee [guide](../guide/SKILL.md)"
  And project file ".claude/skills/review/SKILL.md" contains "when_to_use: on pull request"
  And project file ".claude/skills/guide/SKILL.md" contains "[Review](../review/SKILL.md)"
  And project file ".claude/skills/review/.ai-skills-managed" contains "\"transformers\": [\n    \"flat\",\n    \"claude-when-to-use\"\n  ]"

 Scenario: A second sync updates the managed folders and removes a skill no longer synchronized
  Given CLI project
   """
   {"ai-skills.yaml":"sources:\n  - path: skills\ntarget: out\n",
    "skills/a/SKILL.md":"---\nname: a\n---\n","skills/b/SKILL.md":"---\nname: b\n---\n"}
   """
  When I run arguments "sync"
  And project file "skills/b" is removed
  And I run arguments "sync"
  Then exit code is "0"
  And stdout contains "  update a\n  remove b\n"
  And project path "out/b" exists "false"
  And project path "out/a/SKILL.md" exists "true"

 Scenario: A dry run prints the plans and writes nothing
  Given CLI project
   """
   {"ai-skills.yaml":"sources:\n  - path: skills\ntarget: out\n","skills/a/SKILL.md":"---\nname: a\n---\n"}
   """
  When I run arguments "sync --dry-run"
  Then exit code is "0"
  And stdout contains "  create a\nDry run: 1 skill(s), 1 target(s); nothing written"
  And project path "out" exists "false"

 Scenario: Direct mode needs no config file
  Given CLI project
   """
   {"input/a.skill.md":"---\nname: a\n---\nContent"}
   """
  When I run arguments "sync --type local --path input --target output"
  Then exit code is "0"
  And project file "output/a/SKILL.md" contains "Content"

 Scenario: validate checks the skills and writes nothing
  Given CLI project
   """
   {"ai-skills.yaml":"sources:\n  - path: skills\ntarget: out\n","skills/a/SKILL.md":"---\nname: a\n---\n","skills/b/SKILL.md":"---\nname: b\n---\n"}
   """
  When I run arguments "validate"
  Then exit code is "0"
  And stdout contains "Checked 2 skill(s): no problems"
  And project path "out" exists "false"

 Scenario Outline: Skill problems are printed as a tree and nothing is written
  Given CLI project
   """
   {"ai-skills.yaml":"sources:\n  - path: skills\ntarget: out\n","skills/a/SKILL.md":"---\nname: a\n---\n[x](./gone.md)\n","skills/b/SKILL.md":"---\nname: a\n---\n"}
   """
  When I run arguments "<command>"
  Then exit code is "1"
  And console contains "  skill a (a)\n    duplicate-name: "
  And console contains "    file SKILL.md\n      link [x](./gone.md)\n        missing-link-target: "
  And console contains "Found 3 problem(s)"
  And console shows "  skill a (a)\n" once
  And project path "out" exists "false"
  Examples:
   | command  |
   | validate |
   | sync     |

 Scenario Outline: Configuration problems stop every command before the skills are loaded
  Given CLI project
   """
   {"ai-skills.yaml":"sources:\n  - path: skills\n    <setting>\ntarget: out\n","skills/a/SKILL.md":"---\nname: a\n---\n"}
   """
  When I run arguments "<command>"
  Then exit code is "1"
  And console contains "<code>"
  And project path "out" exists "false"
  Examples:
   | command  | setting            | code             |
   | validate | tags: [go]         | unsupported-tags |
   | sync     | subpath: ../escape | unsafe-subpath   |

 Scenario: A missing subpath is a problem
  Given CLI project
   """
   {"ai-skills.yaml":"sources:\n  - path: skills\n    subpath: missing\ntarget: out\n","skills/a/SKILL.md":"---\nname: a\n---\n"}
   """
  When I run arguments "sync"
  Then exit code is "1"
  And console contains "missing-subpath"

 Scenario: A target folder this tool didn't write is never overwritten
  Given CLI project
   """
   {"ai-skills.yaml":"sources:\n  - path: skills\ntarget: out\n","skills/a/SKILL.md":"---\nname: a\n---\nnew\n","out/a/SKILL.md":"mine"}
   """
  When I run arguments "sync"
  Then exit code is "1"
  And console contains "\n  skill a\n    unmanaged-target: "
  And project file "out/a/SKILL.md" contains "mine"

 Scenario: Deprecated options still work, with a warning
  Given CLI project
   """
   {"ai-skills.yaml":"sources:\n  - path: skills\ntarget:\n  default:\n    path: out\n    adapters: [link-adapter]\nsettings:\n  on_conflict: last_wins\n","skills/a/SKILL.md":"---\nname: a\n---\n"}
   """
  When I run arguments "sync --force"
  Then exit code is "0"
  And console contains "flag=--force"
  And console contains "adapter=link-adapter"
  And console contains "key=settings.on_conflict"
  And project path "out/a/SKILL.md" exists "true"

 Scenario Outline: Usage and failures have stable exit codes
  Given CLI project
   """
   {}
   """
  When I run arguments "<args>"
  Then exit code is "<code>"
  And console contains "<message>"
  Examples:
   | args              | code | message             |
   | --help            | 0    | Usage:              |
   | sync --help       | 0    | --add-relations     |
   | --version         | 0    | test-version        |
   | sync              | 1    | ai-skills.yaml      |
   | validate          | 1    | ai-skills.yaml      |
   | sync --bad        | 2    | unknown flag        |
   | check             | 2    | unknown command     |
   | sync --type local | 1    | --path              |
   | sync --config     | 2    | requires a value    |

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

Feature: Sync loads the skills, transforms them for each target, plans every target, then writes

 # Written targets are listed as "{skill}/{file}" -> content; the marker
 # file is checked by the transformers it lists.

 Background:
  Given a repository "repo" holding
   """
   {"a/guide/SKILL.md":"---\nname: guide\nwhenToUse: on review\n---\n[x](./docs/x.md)\n","a/guide/docs/x.md":"[back](../SKILL.md)\n",
    "h.skill/h.skill.md":"---\nname: h\n---\n[g](../a/guide/docs/x.md)\n"}
   """
  And the sources are
   | repository | subpath | tags | exclude_from_checks |
   | repo       |         |      |                     |

 Scenario: Every target gets the flat layout, its own transformers and the marker
  Given a target "agents" at "/p/.agents/skills" with adapters ""
  And a target "claude" at "/p/.claude/skills" with adapters "claude-property-adapter"
  When I run sync
  Then sync succeeds
  And the synchronized skills are "guide,h"
  And the written targets are "/p/.agents/skills,/p/.claude/skills"
  And the marker of "guide" in "/p/.agents/skills" lists transformers "flat"
  And the marker of "guide" in "/p/.claude/skills" lists transformers "flat,claude-when-to-use"
  And "/p/.claude/skills" is written with
   """
   {"guide/SKILL.md":"---\nname: guide\nwhen_to_use: on review\n---\n[x](./docs/x.md)\n","guide/docs/x.md":"[back](../SKILL.md)\n",
    "guide/.ai-skills-managed":"{\n  \"source\": \"local:repo\",\n  \"skill_path\": \"a/guide\",\n  \"transformers\": [\n    \"flat\",\n    \"claude-when-to-use\"\n  ],\n  \"version\": \"go-2\"\n}\n",
    "h/SKILL.md":"---\nname: h\n---\n[g](../guide/docs/x.md)\n",
    "h/.ai-skills-managed":"{\n  \"source\": \"local:repo\",\n  \"skill_path\": \"h.skill\",\n  \"transformers\": [\n    \"flat\",\n    \"claude-when-to-use\"\n  ],\n  \"version\": \"go-2\"\n}\n"}
   """
  And "/p/.agents/skills" is written with
   """
   {"guide/SKILL.md":"---\nname: guide\nwhenToUse: on review\n---\n[x](./docs/x.md)\n","guide/docs/x.md":"[back](../SKILL.md)\n",
    "guide/.ai-skills-managed":"{\n  \"source\": \"local:repo\",\n  \"skill_path\": \"a/guide\",\n  \"transformers\": [\n    \"flat\"\n  ],\n  \"version\": \"go-2\"\n}\n",
    "h/SKILL.md":"---\nname: h\n---\n[g](../guide/docs/x.md)\n",
    "h/.ai-skills-managed":"{\n  \"source\": \"local:repo\",\n  \"skill_path\": \"h.skill\",\n  \"transformers\": [\n    \"flat\"\n  ],\n  \"version\": \"go-2\"\n}\n"}
   """

 Scenario: Each target is planned against what its folder holds
  Given a target "agents" at "/p/.agents/skills" with adapters ""
  And the target folder "/p/.agents/skills" holds
   | name  | marker |
   | guide | yes    |
   | old   | yes    |
   | mine  | no     |
  And orphans are removed
  When I run sync
  Then sync succeeds
  And the plans are
   """
   {"/p/.agents/skills":[["update","guide"],["create","h"],["remove","old"]]}
   """

 Scenario: A dry run plans every target and writes nothing
  Given a target "agents" at "/p/.agents/skills" with adapters ""
  And it is a dry run
  When I run sync
  Then sync succeeds
  And the plans are
   """
   {"/p/.agents/skills":[["create","guide"],["create","h"]]}
   """
  And the written targets are ""

 Scenario: A folder this tool didn't write in one target stops writing every target
  Given a target "agents" at "/p/.agents/skills" with adapters ""
  And a target "claude" at "/p/.claude/skills" with adapters "claude-property-adapter"
  And the target folder "/p/.claude/skills" holds
   | name | marker |
   | h    | no     |
  When I run sync
  Then the target issues are
   """
   [["unmanaged-target","/p/.claude/skills","h"]]
   """
  And the written targets are ""

 Scenario: A target folder that can't be read stops writing every target
  Given a target "agents" at "/p/.agents/skills" with adapters ""
  And a target "claude" at "/p/.claude/skills" with adapters ""
  And reading the target folder "/p/.claude/skills" fails with "state unavailable"
  When I run sync
  Then sync fails with "state unavailable"
  And the written targets are ""

 Scenario: Skill problems stop sync before any target work
  Given a repository "broken" holding
   """
   {"b/SKILL.md":"---\nname: b\n---\n[gone](./gone.md)\n"}
   """
  And the sources are
   | repository | subpath | tags | exclude_from_checks |
   | broken     |         |      |                     |
   | gone       |         |      |                     |
  And a target "agents" at "/p/.agents/skills" with adapters ""
  When I run sync
  Then the issues are
   """
   [["source-acquire","local:gone","","",""],
    ["missing-link-target","local:broken","b","SKILL.md","[gone](./gone.md)"]]
   """
  And the written targets are ""

Feature: Select the skills one source contributes
 Scenario Outline: Subpaths and tag filters
  Given a source tree
   """
   <files>
   """
  When I select from subpaths "<subpaths>" with tags "<tags>"
  Then discovered names are "<names>" and discovery error contains "<error>"
  Examples:
   | files | subpaths | tags | names | error |
   | {"a.skill.md":"---\\nname: a\\n---\\n"} |  |  | a | |
   | {"skills/a.skill.md":"---\\nname: a\\n---\\n"} | skills |  | a | |
   | {"a.skill.md":"---\\nname: a\\ntags: [go]\\n---\\n"} |  | go | a | |
   | {"a.skill.md":"---\\nname: a\\ntags: [go]\\n---\\n"} |  | cli |  | |
   | {"a.skill.md":"---\\nname: a\\n---\\n"} | ../escape |  |  | unsafe subpath |

 Scenario: Every configured subpath is scanned and results are combined
  Given a source tree
   """
   {"a/SKILL.md":"---\nname: one\n---\n","b/SKILL.md":"---\nname: two\n---\n"}
   """
  When I select from subpaths "a,b" with tags ""
  Then discovered names are "one,two" and discovery error contains ""

 Scenario: A skill reachable from two overlapping subpaths is only selected once
  Given a source tree
   """
   {"a/SKILL.md":"---\nname: one\n---\n"}
   """
  When I select from subpaths ".,a" with tags ""
  Then discovered names are "one" and discovery error contains ""

 Scenario: A structural issue from the catalog surfaces as a discovery error
  Given a source tree
   """
   {"a/SKILL.md":"---\nname: one\n---\n","a/b/SKILL.md":"---\nname: two\n---\n"}
   """
  When I select from subpaths "." with tags ""
  Then discovered names are "" and discovery error contains "nested-skill"

 Scenario: Canceled discovery stops before acquiring the repository
  Given a source tree
   """
   {"a.skill.md":"---\nname: a\n---\n"}
   """
  And discovery is canceled
  When I select from subpaths "." with tags ""
  Then discovered names are "" and discovery error contains "context canceled"
  And the source provider was not acquired

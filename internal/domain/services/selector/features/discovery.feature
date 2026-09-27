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
  # Tags are not applied yet (see AGENTS.md): back when the tag filter is.
  @todo
  Examples:
   | files | subpaths | tags | names | error |
   | {"a.skill.md":"---\\nname: a\\ntags: [go]\\n---\\n"} |  | cli |  | |

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
  When I select from subpaths ".,a" with tags ""
  Then discovered names are "" and discovery error contains "context canceled"
  And the selection issues are
   """
   [["canceled","local:repo","."]]
   """
  And the source provider was not acquired

 Scenario: A source that can't be acquired is one issue, not one per subpath
  Given a source tree
   """
   {"a/SKILL.md":"---\nname: one\n---\n","b/SKILL.md":"---\nname: two\n---\n"}
   """
  And the source provider fails with "clone failed"
  When I select from subpaths "a,b" with tags ""
  Then discovered names are "" and discovery error contains "clone failed"
  And the selection issues are
   """
   [["source-acquire","local:repo",""]]
   """
  And the source provider was acquired 1 time

 Scenario: A missing subpath is reported and the other subpaths are still selected
  Given a source tree
   """
   {"a/SKILL.md":"---\nname: one\n---\n","b/SKILL.md":"---\nname: two\n---\n"}
   """
  When I select from subpaths "a,missing,b" with tags ""
  Then discovered names are "one,two" and discovery error contains "does not exist"
  And the selection issues are
   """
   [["missing-subpath","local:repo","missing"]]
   """

 @todo
 Scenario: An invalid tag expression is a bug in the caller: the config validator rejects it
  Given a source tree
   """
   {"a/SKILL.md":"---\nname: one\n---\n"}
   """
  When I select from subpaths "a" with tags "(go"
  Then selection panics with "invalid tags"
  And the source provider was not acquired

 Scenario: A subpath leading out of the source is a bug in the caller: the config validator rejects it
  Given a source tree
   """
   {"a.skill.md":"---\nname: a\n---\n"}
   """
  When I select from subpaths "../escape" with tags ""
  Then selection panics with "leads out of it"

 Scenario: Issues found in a skill carry the source they come from
  Given a source tree
   """
   {"a/SKILL.md":"---\nname: one\n---\n","a/b/SKILL.md":"---\nname: two\n---\n"}
   """
  When I select from subpaths "." with tags ""
  Then the selection issues are
   """
   [["nested-skill","local:repo","b/SKILL.md"]]
   """

 @todo
 Scenario: A skill the tags reject is not loaded into the catalog, nor checked
  Given a source tree
   """
   {"a/SKILL.md":"---\nname: one\ntags: [go]\n---\n","b/SKILL.md":"---\nname: two\n---\n","b/x/SKILL.md":"---\nname: nested\n---\n"}
   """
  When I select from subpaths "." with tags "go"
  Then discovered names are "one" and discovery error contains ""
  And the catalog holds "one"

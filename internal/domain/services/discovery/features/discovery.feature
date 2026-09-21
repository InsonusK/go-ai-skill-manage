Feature: Discover skill definitions
 Scenario Outline: Skill formats and structural validation
  Given a source tree
   """
   <files>
   """
  When I discover skills at "."
  Then discovered names are "<names>" and discovery error contains "<error>"
  Examples:
   | files | names | error |
   | {"one.skill.md":"---\\nname: one\\n---\\nBody"} | one | |
   | {"a/SKILL.md":"---\\nname: one\\n---\\nBody","a/data.txt":"data"} | one | |
   | {"a/a.skill.md":"---\\nname: one\\n---\\nBody"} | one | |
   | {"a/SKILL.md":"---\\nname: one\\n---\\n","a/a.skill.md":"---\\nname: two\\n---\\n"} | | pattern-conflict |
   | {"a/SKILL.md":"---\\nname: one\\n---\\n","a/b/SKILL.md":"---\\nname: two\\n---\\n"} | | nested-skill |
   | {"a/SKILL.md":"---\\nname: one\\n---\\n","a/examples/b/SKILL.md":"---\\nname: example\\n---\\n"} | one | |
   | {"a.skill.md":"---\\nname: Bad Name\\n---\\n"} | | invalid-name |
   | {"a.skill.md":"no header"} | | invalid-name |
   | {"a.skill.md":"---\\nname: a--b\\n---\\n"} | a--b | |
   | {"a.skill.md":"---\\nname: a---b\\n---\\n"} | | invalid-name |
   | {"README.md":"ordinary"} | | |

 Scenario Outline: Discovered skill format, root and nested files
  Given a source tree
   """
   <files>
   """
  When I discover skills at "."
  Then discovered skill "<name>" has format "<format>" root "<root>" and nested files "<nested>"
  Examples:
   | files | name | format | root | nested |
   | {"one.skill.md":"---\\nname: one\\n---\\nBody"} | one | flat |  |  |
   | {"a.skill/a.skill.md":"---\\nname: one\\n---\\nBody","a.skill/data.txt":"data"} | one | human-dir | a.skill | data.txt |
   | {"a/SKILL.md":"---\\nname: one\\n---\\nBody","a/data.txt":"data"} | one | agent-dir | a | data.txt |
   | {"a/SKILL.md":"---\\nname: one\\n---\\nBody","a/sub/data.txt":"data"} | one | agent-dir | a | sub/data.txt |

 Scenario: A flat example belongs to its ancestor skill
  Given a source tree
   """
   {"guide/SKILL.md":"---\nname: guide\n---\nBody","guide/examples/sample.skill.md":"---\nname: sample\n---\nSample"}
   """
  When I find the owner of "guide/examples/sample.skill.md"
  Then discovered names are "guide" and discovery error contains ""

 Scenario: Canceled discovery stops before reading the tree
  Given a source tree
   """
   {"a.skill.md":"---\nname: a\n---\n"}
   """
  And discovery is canceled
  When I discover skills at "."
  Then discovered names are "" and discovery error contains "context canceled"

 Scenario Outline: Select resolves subpaths, tags and name override
  Given a source tree
   """
   <files>
   """
  When I select from subpath "<subpath>" with tags "<tags>" and name "<name>"
  Then discovered names are "<names>" and discovery error contains "<error>"
  Examples:
   | files | subpath | tags | name | names | error |
   | {"a.skill.md":"---\\nname: a\\n---\\n"} |  |  |  | a | |
   | {"skills/a.skill.md":"---\\nname: a\\n---\\n"} | skills |  |  | a | |
   | {"a.skill.md":"---\\nname: a\\ntags: [go]\\n---\\n"} |  | go |  | a | |
   | {"a.skill.md":"---\\nname: a\\ntags: [go]\\n---\\n"} |  | cli |  |  | |
   | {"a.skill.md":"---\\nname: a\\n---\\n"} |  |  | renamed | renamed | |
   | {"a.skill.md":"---\\nname: a\\n---\\n"} | ../escape |  |  |  | unsafe subpath |

 Scenario: A single-file source ignores a configured subpath
  Given a source tree
   """
   {"a.skill.md":"---\nname: a\n---\n"}
   """
  When I select the single file "a.skill.md" with subpath "elsewhere"
  Then discovered names are "a" and discovery error contains ""

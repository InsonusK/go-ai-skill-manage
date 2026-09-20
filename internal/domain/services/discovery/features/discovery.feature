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

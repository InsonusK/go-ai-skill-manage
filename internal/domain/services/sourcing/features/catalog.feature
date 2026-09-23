Feature: SkillCatalog finds and validates skills by path
 Scenario Outline: Skill formats and structural validation
  Given a source tree
   """
   <files>
   """
  When I get or add skills at "."
  Then found names are "<names>" and catalog error contains "<error>"
  Examples:
   | files | names | error |
   | {"one.skill.md":"---\\nname: one\\n---\\nBody"} | one | |
   | {"a/SKILL.md":"---\\nname: one\\n---\\nBody","a/data.txt":"data"} | one | |
   | {"a/a.skill.md":"---\\nname: one\\n---\\nBody"} | one | |
   | {"a/SKILL.md":"---\\nname: one\\n---\\n","a/a.skill.md":"---\\nname: two\\n---\\n"} | | pattern-conflict |
   | {"a/SKILL.md":"---\\nname: one\\n---\\n","a/b/SKILL.md":"---\\nname: two\\n---\\n"} | | nested-skill |
   | {"a.skill.md":"---\\nname: Bad Name\\n---\\n"} | | invalid-name |
   | {"README.md":"ordinary"} | | |

 Scenario: SkipFolders exempts a folder from nested-skill
  Given a source tree
   """
   {"a/SKILL.md":"---\nname: one\n---\n","a/examples/b/SKILL.md":"---\nname: example\n---\n"}
   """
  And skip folders are "examples"
  When I get or add skills at "."
  Then found names are "one" and catalog error contains ""

 Scenario: Recursive discovery finds multiple skills and does not search inside a found skill's own directory
  Given a source tree
   """
   {"a/SKILL.md":"---\nname: one\n---\n","b/SKILL.md":"---\nname: two\n---\n"}
   """
  When I get or add skills at "."
  Then found names are "one,two" and catalog error contains ""

 Scenario: The found skill's file-list cache is already warm
  Given a source tree
   """
   {"a/SKILL.md":"---\nname: one\n---\n","a/notes.md":"hi"}
   """
  When I get or add skills at "."
  Then total directory reads so far are remembered
  When I list files by path "" for skill "one"
  Then no additional directories were read

 Scenario: A repeated GetByPath call for the same path reuses the already-resolved skill
  Given a source tree
   """
   {"a/SKILL.md":"---\nname: one\n---\n"}
   """
  When I get or add skills at "a"
  Then total directory reads so far are remembered
  When I get or add skills at "a"
  Then no additional directories were read
  And found names are "one" and catalog error contains ""

 Scenario: Manager acquisition is lazy and cached across GetByPath calls
  Given a source tree
   """
   {"a/SKILL.md":"---\nname: one\n---\n","b/SKILL.md":"---\nname: two\n---\n"}
   """
  When I get or add skills at "a"
  And I get or add skills at "b"
  Then the source manager acquired "1" times

 Scenario: Owner finds the skill owning a path inside its root
  Given a source tree
   """
   {"a/SKILL.md":"---\nname: one\n---\n","a/notes.md":"hi"}
   """
  When I get or add skills at "."
  Then catalog owner of "a/notes.md" is "one"
  And catalog owner of "a/SKILL.md" is "one"
  And catalog owner of "a" is "one"

 Scenario: Owner reports no owner for a path outside every known skill
  Given a source tree
   """
   {"a/SKILL.md":"---\nname: one\n---\n"}
   """
  When I get or add skills at "."
  Then catalog owner of "b/notes.md" is not found

 Scenario: Owner does not extend a flat skill's ownership beyond its own file
  Given a source tree
   """
   {"one.skill.md":"---\nname: one\n---\n"}
   """
  When I get or add skills at "."
  Then catalog owner of "one.skill.md" is "one"
  And catalog owner of "one.skill.md.bak" is not found

 Scenario: Destination resolves a path to its skill's output location
  Given a source tree
   """
   {"a/SKILL.md":"---\nname: one\n---\n","a/notes.md":"hi"}
   """
  When I get or add skills at "."
  Then catalog destination of "a/notes.md" is "one" at "one/notes.md"
  And catalog destination of "a/SKILL.md" is "one" at "one/SKILL.md"

 Scenario: Destination reports nothing for a path outside every known skill
  Given a source tree
   """
   {"a/SKILL.md":"---\nname: one\n---\n"}
   """
  When I get or add skills at "."
  Then catalog destination of "b/notes.md" is not found

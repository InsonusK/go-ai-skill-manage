Feature: SkillCatalog.FetchByPath finds every valid skill at or below a path

 # FetchByPath searches down from the path: a folder that is a skill's
 # folder becomes one skill and is not searched further, any other folder
 # is searched into, a "*.skill.md" file outside a skill's folder is a flat
 # skill. It remembers nothing -- that is addPath's job.

 Scenario Outline: Skill formats and structural validation
  Given a source tree
   """
   <files>
   """
  When I fetch skills at "."
  Then found names are "<names>" and catalog error contains "<error>"
  Examples:
   | files                                                                               | names | error            |
   | {"one.skill.md":"---\\nname: one\\n---\\nBody"}                                     | one   |                  |
   | {"a/SKILL.md":"---\\nname: one\\n---\\nBody","a/data.txt":"data"}                   | one   |                  |
   | {"a/a.skill.md":"---\\nname: one\\n---\\nBody"}                                     | one   |                  |
   | {"a/SKILL.md":"---\\nname: one\\n---\\n","a/a.skill.md":"---\\nname: two\\n---\\n"} |       | pattern-conflict |
   | {"a/SKILL.md":"---\\nname: one\\n---\\n","a/b/SKILL.md":"---\\nname: two\\n---\\n"} |       | nested-skill     |
   | {"a.skill.md":"---\\nname: Bad Name\\n---\\n"}                                      |       | invalid-name     |
   | {"README.md":"ordinary"}                                                            |       |                  |

 Scenario Outline: A folder excluded from checks of the skill's source is exempt from nested-skill
  Given a source tree
   """
   {"a/SKILL.md":"---\nname: one\n---\n","a/examples/b/SKILL.md":"---\nname: example\n---\n"}
   """
  And folders excluded from checks of source "<source>" are "<folders>"
  When I fetch skills at "."
  Then found names are "<names>" and catalog error contains "<error>"
  Examples:
   | source | folders  | names | error        |
   | repo   | examples | one   |              |
   | repo   |          |       | nested-skill |
   | other  | examples |       | nested-skill |

 Scenario: Skills in several folders are all found
  Given a source tree
   """
   {"a/SKILL.md":"---\nname: one\n---\n","b/SKILL.md":"---\nname: two\n---\n"}
   """
  When I fetch skills at "."
  Then found names are "one,two" and catalog error contains ""

 Scenario: The found skill's file-list cache is already warm
  Given a source tree
   """
   {"a/SKILL.md":"---\nname: one\n---\n","a/notes.md":"hi"}
   """
  When I fetch skills at "."
  Then total directory reads so far are remembered
  When I list files by path "" for skill "one"
  Then no additional directories were read

 Scenario Outline: The start path decides where the search goes down from
  Given a source tree
   """
   {"a/guide/SKILL.md"  :"---\nname: guide\n---\n",
    "a/guide/docs/x.md" :"x",
    "b.skill.md"        :"---\nname: flat\n---\n",
    "h.skill/h.skill.md":"---\nname: human\n---\n",
    "x/notes.md"        :"n"}
   """
  When I fetch skills at "<start>"
  Then found names are "<names>" and catalog error contains ""
  Examples:
   | start              | names            |
   | .                  | guide,flat,human |
   | a                  | guide            |
   | a/guide            | guide            |
   | h.skill            | human            |
   | b.skill.md         | flat             |
   | a/guide/docs       |                  |
   | a/guide/SKILL.md   |                  |
   | x                  |                  |
   | x/notes.md         |                  |
   | missing            |                  |

 Scenario Outline: A found skill has the format and location of its marker
  Given a source tree
   """
   <files>
   """
  When I fetch skills at "."
  Then the found skill is "one" at "<location>" in format "<format>"
  Examples:
   | files                                                       | location       | format    |
   | {"a/guide/SKILL.md":"---\\nname: one\\n---\\n"}             | a/guide        | agent-dir |
   | {"a/guide.skill/SKILL.md":"---\\nname: one\\n---\\n"}       | a/guide.skill  | agent-dir |
   | {"a/guide.skill/guide.skill.md":"---\\nname: one\\n---\\n"} | a/guide.skill  | human-dir |
   | {"a/one.skill.md":"---\\nname: one\\n---\\n"}               | a/one.skill.md | flat      |
   | {"SKILL.md":"---\\nname: one\\n---\\n"}                     | .              | agent-dir |

 Scenario: The .git folder is not searched
  Given a source tree
   """
   {".git/SKILL.md":"---\nname: git\n---\n","a/SKILL.md":"---\nname: one\n---\n"}
   """
  When I fetch skills at "."
  Then found names are "one" and catalog error contains ""

 Scenario: Valid skills are found next to invalid ones, which are reported as issues
  Given a source tree
   """
   {"a/SKILL.md":"---\nname: one\n---\n","b/SKILL.md":"---\nname: Bad Name\n---\n","c/SKILL.md":"---\nname: three\n---\n"}
   """
  When I fetch skills at "."
  Then found names are "one,three" and catalog error contains "invalid skill name"

 Scenario: A skill already loaded at its own folder is reused, not built again
  Given a source tree
   """
   {"a/SKILL.md":"---\nname: one\n---\n","b/SKILL.md":"---\nname: two\n---\n"}
   """
  When I get or add skills at "a"
  And the found skills are remembered
  And I fetch skills at "."
  Then found names are "one,two" and catalog error contains ""
  And the found skill "one" is the one remembered

 Scenario: FetchByPath remembers nothing
  Given a source tree
   """
   {"a/SKILL.md":"---\nname: one\n---\n"}
   """
  When I fetch skills at "."
  And I get cached skills at "."
  Then the catalog reports the skill is not cached
  When I get cached skills at "a"
  Then the catalog reports the skill is not cached

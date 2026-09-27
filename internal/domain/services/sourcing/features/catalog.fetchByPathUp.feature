Feature: SkillCatalog.FetchByPathUp finds the valid skill whose folder holds a path

 # FetchByPathUp searches up from the path: it checks the path's own folder
 # (the path itself when it is a folder) and then each parent folder up to
 # the repository folder, and returns the first skill's folder it meets --
 # or the path itself when it is a flat "*.skill.md" skill. It never loads
 # neighbour skills and remembers nothing -- that is addPath's job.

 Scenario Outline: The skill whose folder holds the path is found
  Given a source tree
   """
   {"a/guide/SKILL.md":"---\nname: guide\n---\n","a/guide/docs/deep/x.md":"x","a/human.skill/human.skill.md":"---\nname: human\n---\n","a/human.skill/docs/x.md":"x","b.skill.md":"---\nname: flat\n---\n","f/f.skill.md":"---\nname: plain\n---\n"}
   """
  When I fetch the skill holding "<path>"
  Then the found skill is "<name>" at "<location>" in format "<format>"
  Examples:
   | path                           | name  | location                     | format    |
   | a/guide/docs/deep/x.md         | guide | a/guide                      | agent-dir |
   | a/guide/docs                   | guide | a/guide                      | agent-dir |
   | a/guide/SKILL.md               | guide | a/guide                      | agent-dir |
   | a/guide                        | guide | a/guide                      | agent-dir |
   | a/human.skill/human.skill.md   | human | a/human.skill                | human-dir |
   | a/human.skill/docs/x.md        | human | a/human.skill                | human-dir |
   | a/human.skill                  | human | a/human.skill                | human-dir |
   | b.skill.md                     | flat  | b.skill.md                   | flat      |
   | f/f.skill.md                   | plain | f/f.skill.md                 | flat      |

 Scenario: A skill in the repository folder itself holds every other path
  Given a source tree
   """
   {"SKILL.md":"---\nname: root\n---\n","docs/x.md":"x"}
   """
  When I fetch the skill holding "docs/x.md"
  Then the found skill is "root" at "." in format "agent-dir"

 Scenario: The nearest skill folder wins
  Given a source tree
   """
   {"a/b/SKILL.md":"---\nname: inner\n---\n","a/b/x.md":"x"}
   """
  When I fetch the skill holding "a/b/x.md"
  Then the found skill is "inner" at "a/b" in format "agent-dir"

 Scenario Outline: No skill, or an invalid one, is reported
  Given a source tree
   """
   <files>
   """
  When I fetch the skill holding "<path>"
  Then found names are "" and catalog error contains "<error>"
  Examples:
   | files | path | error |
   | {"README.md":"hi"} | README.md | skill-not-found |
   | {"x/one/SKILL.md":"---\\nname: one\\n---\\n","x/notes.md":"n"} | x/notes.md | skill-not-found |
   | {"a/SKILL.md":"---\\nname: one\\n---\\n","a/b/SKILL.md":"---\\nname: two\\n---\\n","a/x.md":"hi"} | a/x.md | nested-skill |
   | {"a/SKILL.md":"---\\nname: one\\n---\\n","a/a.skill.md":"---\\nname: two\\n---\\n","a/x.md":"hi"} | a/x.md | pattern-conflict |
   | {"a/SKILL.md":"---\\nname: Bad Name\\n---\\n","a/x.md":"hi"} | a/x.md | invalid skill name |
   | {"a/SKILL.md":"---\\nname: one\\n---\\n"} | a/missing.md | not exist |

 Scenario: Neighbour skills are not loaded and nothing is remembered
  Given a source tree
   """
   {"a/SKILL.md":"---\nname: one\n---\n","a/x.md":"x","b/SKILL.md":"---\nname: two\n---\n"}
   """
  When I fetch the skill holding "a/x.md"
  Then total directory reads so far are remembered
  And no directory under "b" was read
  When I get the cached skill holding "a/x.md"
  Then the catalog reports the skill is not cached

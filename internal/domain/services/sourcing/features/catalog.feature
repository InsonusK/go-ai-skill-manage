Feature: SkillCatalog answers from its cache and fetches on a miss
 Scenario: A repeated GetOrFetchByPath call for the same path reuses the already-resolved skill
  Given a source tree
   """
   {"a/SKILL.md":"---\nname: one\n---\n"}
   """
  When I get or add skills at "a"
  Then total directory reads so far are remembered
  When I get or add skills at "a"
  Then no additional directories were read
  And found names are "one" and catalog error contains ""

 Scenario: Manager acquisition is lazy and cached across GetOrFetchByPath calls
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

 Scenario Outline: GetByPath answers by key -- a requested path or a skill's own folder -- without reading anything
  Given a source tree
   """
   {"a/SKILL.md":"---\nname: one\n---\n","b/c/SKILL.md":"---\nname: two\n---\n"}
   """
  When I get or add skills at "."
  Then total directory reads so far are remembered
  When I get cached skills at "<path>"
  Then no additional directories were read
  And found names are "<names>" and catalog error contains ""
  Examples:
   | path | names   |
   | .    | one,two |
   | a    | one     |
   | b/c  | two     |

 Scenario: GetByPath fails with not-cached for a path never requested, even with loaded skills below it
  Given a source tree
   """
   {"a/SKILL.md":"---\nname: one\n---\n","b/c/SKILL.md":"---\nname: two\n---\n"}
   """
  When I get or add skills at "."
  And I get cached skills at "b"
  Then the catalog reports the skill is not cached

 Scenario: GetOrFetchByPath fetches a wider path even when a skill below it is already loaded
  Given a source tree
   """
   {"a/SKILL.md":"---\nname: one\n---\n","b/SKILL.md":"---\nname: two\n---\n"}
   """
  When I get or add skills at "a"
  And I get or add skills at "."
  Then found names are "one,two" and catalog error contains ""

 Scenario: A fetch that found an invalid skill is not remembered as complete, so it reports the issue again
  Given a source tree
   """
   {"a/SKILL.md":"---\nname: one\n---\n","b/SKILL.md":"---\nname: two\n---\n","b/c/SKILL.md":"---\nname: three\n---\n"}
   """
  When I get or add skills at "."
  Then found names are "one" and catalog error contains "nested-skill"
  When I get or add skills at "."
  Then found names are "one" and catalog error contains "nested-skill"
  When I get cached skills at "a"
  Then found names are "one" and catalog error contains ""

 Scenario: GetByPath fails with not-cached when the path was never resolved
  Given a source tree
   """
   {"a/SKILL.md":"---\nname: one\n---\n","b/SKILL.md":"---\nname: two\n---\n"}
   """
  When I get or add skills at "a"
  And I get cached skills at "b"
  Then the catalog reports the skill is not cached

 Scenario: GetByPath fails with not-cached before the source is acquired
  Given a source tree
   """
   {"a/SKILL.md":"---\nname: one\n---\n"}
   """
  When I get cached skills at "."
  Then the catalog reports the skill is not cached
  And the source manager acquired "0" times

 Scenario Outline: TryGetOrFetchByPath fetches only when add relations is on
  Given a source tree
   """
   {"a/SKILL.md":"---\nname: one\n---\n"}
   """
  And add relations is "<relations>"
  When I try to get or fetch skills at "a"
  Then the source manager acquired "<acquired>" times
  Examples:
   | relations | acquired |
   | true      | 1        |
   | false     | 0        |

 Scenario: TryGetOrFetchByPath with add relations on returns the fetched skill
  Given a source tree
   """
   {"a/SKILL.md":"---\nname: one\n---\n"}
   """
  And add relations is "true"
  When I try to get or fetch skills at "a"
  Then found names are "one" and catalog error contains ""

 Scenario: TryGetOrFetchByPath with add relations off fails with not-cached
  Given a source tree
   """
   {"a/SKILL.md":"---\nname: one\n---\n"}
   """
  And add relations is "false"
  When I try to get or fetch skills at "a"
  Then the catalog reports the skill is not cached

 Scenario: TryGetOrFetchByPathUp returns a cached skill without reading anything
  Given a source tree
   """
   {"a/SKILL.md":"---\nname: one\n---\n","a/docs/x.md":"hi"}
   """
  And add relations is "false"
  When I get or add skills at "."
  Then total directory reads so far are remembered
  When I try to get or fetch the skill holding "a/docs/x.md"
  Then no additional directories were read
  And found names are "one" and catalog error contains ""

 Scenario: TryGetOrFetchByPathUp with add relations off fails with not-cached
  Given a source tree
   """
   {"a/SKILL.md":"---\nname: one\n---\n","a/docs/x.md":"hi"}
   """
  And add relations is "false"
  When I try to get or fetch the skill holding "a/docs/x.md"
  Then the catalog reports the skill is not cached

 Scenario: TryGetOrFetchByPathUp with add relations on fetches the skill whose folder holds the path
  Given a source tree
   """
   {"a/SKILL.md":"---\nname: one\n---\n","a/docs/x.md":"hi"}
   """
  And add relations is "true"
  When I try to get or fetch the skill holding "a/docs/x.md"
  Then found names are "one" and catalog error contains ""
  When I get the cached skill holding "a/docs/x.md"
  Then found names are "one" and catalog error contains ""

 Scenario: TryGetOrFetchByPathUp with add relations on refuses a path missing from the repository
  Given a source tree
   """
   {"a/SKILL.md":"---\nname: one\n---\n"}
   """
  And add relations is "true"
  When I try to get or fetch the skill holding "a/missing.md"
  Then found names are "" and catalog error contains "does not exist"

 Scenario: TryGetOrFetchByPathUp fetches only the skill holding the path, not its neighbours
  Given a source tree
   """
   {"a/SKILL.md":"---\nname: one\n---\n","a/x.md":"hi","b/SKILL.md":"---\nname: two\n---\n"}
   """
  And add relations is "true"
  When I try to get or fetch the skill holding "a/x.md"
  And I get cached skills at "a"
  Then found names are "one" and catalog error contains ""
  When I get cached skills at "b"
  Then the catalog reports the skill is not cached

 Scenario: GetByPathUp returns the cached skill whose folder holds a path, without reading anything
  Given a source tree
   """
   {"a/SKILL.md":"---\nname: one\n---\n","a/docs/x.md":"hi"}
   """
  When I get or add skills at "."
  Then total directory reads so far are remembered
  When I get the cached skill holding "a/docs/x.md"
  Then no additional directories were read
  And found names are "one" and catalog error contains ""

 Scenario: GetByPathUp fails with not-cached even when add relations is on
  Given a source tree
   """
   {"a/SKILL.md":"---\nname: one\n---\n","a/x.md":"hi","b/SKILL.md":"---\nname: two\n---\n","b/x.md":"hi"}
   """
  And add relations is "true"
  When I get or add skills at "a"
  And I get the cached skill holding "b/x.md"
  Then the catalog reports the skill is not cached
  And the source manager acquired "1" times

 Scenario Outline: GetByPathUp finds a loaded skill from any path inside its folder or its flat marker file
  Given a source tree
   """
   {"a/SKILL.md":"---\nname: one\n---\n","a/docs/deep/x.md":"hi","two.skill.md":"---\nname: two\n---\n"}
   """
  When I get or add skills at "."
  And I get the cached skill holding "<path>"
  Then found names are "<names>" and catalog error contains ""
  Examples:
   | path             | names |
   | a/docs/deep/x.md | one   |
   | a                | one   |
   | a/SKILL.md       | one   |
   | two.skill.md     | two   |

 Scenario: GetByPathUp does not take a requested path above a skill for that skill's folder
  Given a source tree
   """
   {"x/y/SKILL.md":"---\nname: one\n---\n","x/z.md":"hi"}
   """
  When I get or add skills at "x"
  Then found names are "one" and catalog error contains ""
  When I get the cached skill holding "x/z.md"
  Then the catalog reports the skill is not cached

 Scenario: A wider fetch searches into a requested path above skills instead of taking it for one skill
  Given a source tree
   """
   {"x/y1/SKILL.md":"---\nname: one\n---\n","x/y2/SKILL.md":"---\nname: two\n---\n"}
   """
  When I get or add skills at "x"
  And I get or add skills at "."
  Then found names are "one,two" and catalog error contains ""

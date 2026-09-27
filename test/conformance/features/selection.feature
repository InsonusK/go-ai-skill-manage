Feature: What gets selected and what stops sync before writing

 Scenario: Without add_relations a link to a skill no source selects stops sync
  Given the project
   """
   {"ai-skills.yaml":"sources:\n  - path: skills\n    subpath: a\nsettings:\n  target: out\n  add_relations: false\n",
    "skills/a/SKILL.md":"---\nname: a\n---\n[b](../b/SKILL.md)\n","skills/b/SKILL.md":"---\nname: b\n---\n"}
   """
  When I run aism "sync"
  Then aism fails
  And the project path "out" does not exist

 Scenario: With add_relations the linked skill is written too
  Given the project
   """
   {"ai-skills.yaml":"sources:\n  - path: skills\n    subpath: a\nsettings:\n  target: out\n  add_relations: true\n",
    "skills/a/SKILL.md":"---\nname: a\n---\n[b](../b/SKILL.md)\n","skills/b/SKILL.md":"---\nname: b\n---\n"}
   """
  When I run aism "sync"
  Then aism succeeds
  And in "out/a/SKILL.md" the link "b" leads to "out/b/SKILL.md"

 Scenario: A skill inside another skill's folder stops sync
  Given the project
   """
   {"ai-skills.yaml":"sources:\n  - path: skills\nsettings:\n  target: out\n",
    "skills/a/SKILL.md":"---\nname: a\n---\n","skills/a/inner/SKILL.md":"---\nname: inner\n---\n"}
   """
  When I run aism "sync"
  Then aism fails
  And the project path "out" does not exist

 Scenario: Two skills of one name stop sync
  Given the project
   """
   {"ai-skills.yaml":"sources:\n  - path: one\n  - path: two\nsettings:\n  target: out\n",
    "one/a/SKILL.md":"---\nname: same\n---\n","two/b/SKILL.md":"---\nname: same\n---\n"}
   """
  When I run aism "sync"
  Then aism fails
  And the project path "out" does not exist

 Scenario: A skill name that isn't kebab-case stops sync
  Given the project
   """
   {"ai-skills.yaml":"sources:\n  - path: skills\nsettings:\n  target: out\n",
    "skills/a/SKILL.md":"---\nname: Bad Name\n---\n"}
   """
  When I run aism "sync"
  Then aism fails
  And the project path "out" does not exist

 Scenario: A subpath that doesn't exist stops sync
  Given the project
   """
   {"ai-skills.yaml":"sources:\n  - path: skills\n    subpath: missing\nsettings:\n  target: out\n",
    "skills/a/SKILL.md":"---\nname: a\n---\n"}
   """
  When I run aism "sync"
  Then aism fails

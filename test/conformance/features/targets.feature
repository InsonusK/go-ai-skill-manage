Feature: What sync does with the folders already in the target

 Scenario: A dry run writes nothing
  Given the project
   """
   {"ai-skills.yaml":"sources:\n  - path: skills\nsettings:\n  target: out\n","skills/a/SKILL.md":"---\nname: a\n---\n"}
   """
  When I run aism "sync --dry-run"
  Then aism succeeds
  And the project path "out" does not exist

 Scenario: A second sync replaces the skill folder: files the skill dropped go away
  Given the project
   """
   {"ai-skills.yaml":"sources:\n  - path: skills\nsettings:\n  target: out\n",
    "skills/a/SKILL.md":"---\nname: a\n---\nv1\n","skills/a/old.md":"old\n"}
   """
  When I run aism "sync"
  And I remove the project path "skills/a/old.md"
  And I run aism "sync"
  Then aism succeeds
  And the project folder "out" holds exactly
   """
   ["a/SKILL.md","a/.ai-skills-managed"]
   """

 Scenario: A skill removed from the sources is removed from the target
  Given the project
   """
   {"ai-skills.yaml":"sources:\n  - path: skills\nsettings:\n  target: out\n  remove_orphans: true\n",
    "skills/a/SKILL.md":"---\nname: a\n---\n","skills/b/SKILL.md":"---\nname: b\n---\n"}
   """
  When I run aism "sync"
  And I remove the project path "skills/b"
  And I run aism "sync"
  Then aism succeeds
  And the project path "out/b" does not exist
  And the project path "out/a/SKILL.md" exists

 Scenario: A folder the tool didn't write is never replaced
  Given the project
   """
   {"ai-skills.yaml":"sources:\n  - path: skills\nsettings:\n  target: out\n",
    "skills/a/SKILL.md":"---\nname: a\n---\nnew\n","out/a/SKILL.md":"mine\n"}
   """
  When I run aism "sync"
  Then aism fails
  And the project file "out/a/SKILL.md" contains "mine"

 Scenario: A folder the tool didn't write, and no skill needs, is left alone
  Given the project
   """
   {"ai-skills.yaml":"sources:\n  - path: skills\nsettings:\n  target: out\n  remove_orphans: true\n",
    "skills/a/SKILL.md":"---\nname: a\n---\n","out/mine/SKILL.md":"mine\n"}
   """
  When I run aism "sync"
  Then aism succeeds
  And the project file "out/mine/SKILL.md" contains "mine"

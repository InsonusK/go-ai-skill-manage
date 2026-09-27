Feature: Every skill is written into the target as {name}/SKILL.md with its files

 # Configs use settings.target: both implementations read it.

 Scenario: Every source form of a skill becomes a folder named after it
  Given the project
   """
   {"ai-skills.yaml":"sources:\n  - path: skills\nsettings:\n  target: out\n",
    "skills/a/guide/SKILL.md":"---\nname: guide\n---\nGuide\n","skills/a/guide/docs/x.md":"x\n",
    "skills/h.skill/h.skill.md":"---\nname: h\n---\nH\n","skills/h.skill/y.md":"y\n",
    "skills/f.skill.md":"---\nname: f\n---\nF\n"}
   """
  When I run aism "sync"
  Then aism succeeds
  And the project folder "out" holds exactly
   """
   ["guide/SKILL.md","guide/docs/x.md","guide/.ai-skills-managed",
    "h/SKILL.md","h/y.md","h/.ai-skills-managed",
    "f/SKILL.md","f/.ai-skills-managed"]
   """
  And the project file "out/h/SKILL.md" contains "H\n"

 Scenario: An executable file stays executable
  Given the project
   """
   {"ai-skills.yaml":"sources:\n  - path: skills\nsettings:\n  target: out\n",
    "skills/a/SKILL.md":"---\nname: a\n---\n","skills/a/run.sh":"echo\n"}
   """
  And the project file "skills/a/run.sh" is executable
  When I run aism "sync"
  Then aism succeeds
  And the project file "out/a/run.sh" is executable in the target too

 Scenario: A Claude target gets whenToUse as when_to_use, another target keeps it
  Given the project
   """
   {"ai-skills.yaml":"sources:\n  - path: skills\nsettings:\n  target:\n    for_each:\n      adapters: [link-adapter]\n    default:\n      path: out\n    claude:\n      path: claude\n      adapters: [claude-property-adapter]\n",
    "skills/a/SKILL.md":"---\nname: a\ndescription: d\nwhenToUse: on review\n---\nBody\n"}
   """
  When I run aism "sync"
  Then aism succeeds
  And the project file "claude/a/SKILL.md" contains "when_to_use: on review"
  And the project file "claude/a/SKILL.md" does not contain "whenToUse"
  And the project file "out/a/SKILL.md" contains "whenToUse: on review"
  And the project file "out/a/SKILL.md" contains "Body"

 Scenario: Folders named examples are copied with the skill, even with a skill marker or a broken link inside
  Given the project
   """
   {"ai-skills.yaml":"sources:\n  - path: skills\nsettings:\n  target: out\n",
    "skills/a/SKILL.md":"---\nname: a\n---\n","skills/a/examples/demo/SKILL.md":"---\nname: demo\n---\n[gone](./gone.md)\n"}
   """
  When I run aism "sync"
  Then aism succeeds
  And the project folder "out" holds exactly
   """
   ["a/SKILL.md","a/.ai-skills-managed","a/examples/demo/SKILL.md"]
   """

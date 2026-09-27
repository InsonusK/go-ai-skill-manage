Feature: Links in the written skills lead to the same files at their new places

 # "leads to" reads a link the way a markdown viewer does: relative to the
 # folder of the file that holds it.

 Scenario: Links inside a skill, between skills, wikilinks and anchors keep working
  Given the project
   """
   {"ai-skills.yaml":"sources:\n  - path: skills\nsettings:\n  target: out\n",
    "skills/a/SKILL.md":"---\nname: a\n---\n[doc](./docs/x.md) [b](../b/SKILL.md) [[b/SKILL.md|wiki b]] [top](./docs/x.md#top)\n",
    "skills/a/docs/x.md":"# Top\n[back](../SKILL.md)\n",
    "skills/b/SKILL.md":"---\nname: b\n---\nB\n"}
   """
  When I run aism "sync"
  Then aism succeeds
  And in "out/a/SKILL.md" the link "doc" leads to "out/a/docs/x.md"
  And in "out/a/SKILL.md" the link "b" leads to "out/b/SKILL.md"
  And in "out/a/SKILL.md" the link "wiki b" leads to "out/b/SKILL.md"
  And in "out/a/SKILL.md" the link "top" leads to "out/a/docs/x.md"
  And in "out/a/docs/x.md" the link "back" leads to "out/a/SKILL.md"

 Scenario: A link to a human-dir or flat skill leads to its SKILL.md
  Given the project
   """
   {"ai-skills.yaml":"sources:\n  - path: skills\nsettings:\n  target: out\n",
    "skills/a/SKILL.md":"---\nname: a\n---\n[h](../h.skill/h.skill.md) [f](../f.skill.md)\n",
    "skills/h.skill/h.skill.md":"---\nname: h\n---\n","skills/f.skill.md":"---\nname: f\n---\n"}
   """
  When I run aism "sync"
  Then aism succeeds
  And in "out/a/SKILL.md" the link "h" leads to "out/h/SKILL.md"
  And in "out/a/SKILL.md" the link "f" leads to "out/f/SKILL.md"

 # A wikilink path without "./" starts at the source folder.
 Scenario: A link without .md leads to the note, even beside a folder of that name
  Given the project
   """
   {"ai-skills.yaml":"sources:\n  - path: skills\nsettings:\n  target: out\n",
    "skills/a/SKILL.md":"---\nname: a\n---\n[[a/x.extend|note]]\n",
    "skills/a/x.extend.md":"# Note\n","skills/a/x.extend/file.txt":"f\n"}
   """
  When I run aism "sync"
  Then aism succeeds
  And in "out/a/SKILL.md" the link "note" leads to "out/a/x.extend.md"

 Scenario: A link to a file outside every skill still works in the target
  Given the project
   """
   {"ai-skills.yaml":"sources:\n  - path: skills\nsettings:\n  target: out\n",
    "skills/a/SKILL.md":"---\nname: a\n---\n[notes](../shared/notes.md)\n",
    "skills/shared/notes.md":"shared notes\n"}
   """
  When I run aism "sync"
  Then aism succeeds
  And in "out/a/SKILL.md" the link "notes" leads to a file containing "shared notes"

 Scenario: A link to a file that doesn't exist stops sync
  Given the project
   """
   {"ai-skills.yaml":"sources:\n  - path: skills\nsettings:\n  target: out\n",
    "skills/a/SKILL.md":"---\nname: a\n---\n[gone](./gone.md)\n"}
   """
  When I run aism "sync"
  Then aism fails
  And the project path "out" does not exist

 Scenario: A broken link in a file the skill doesn't link to stops sync too
  Given the project
   """
   {"ai-skills.yaml":"sources:\n  - path: skills\nsettings:\n  target: out\n",
    "skills/a/SKILL.md":"---\nname: a\n---\nNo links\n","skills/a/docs/orphan.md":"[gone](./gone.md)\n"}
   """
  When I run aism "sync"
  Then aism fails
  And the project path "out" does not exist

 Scenario: A link to a heading that doesn't exist stops sync
  Given the project
   """
   {"ai-skills.yaml":"sources:\n  - path: skills\nsettings:\n  target: out\n",
    "skills/a/SKILL.md":"---\nname: a\n---\n[x](./docs/x.md#nowhere)\n","skills/a/docs/x.md":"# Top\n"}
   """
  When I run aism "sync"
  Then aism fails
  And the project path "out" does not exist

 Scenario: A badge, an image inside a link, is kept
  Given the project
   """
   {"ai-skills.yaml":"sources:\n  - path: skills\nsettings:\n  target: out\n",
    "skills/a/SKILL.md":"---\nname: a\n---\n[![CI](https://example.com/badge.svg)](https://example.com/ci)\n"}
   """
  When I run aism "sync"
  Then aism succeeds
  And the project file "out/a/SKILL.md" contains "[![CI](https://example.com/badge.svg)](https://example.com/ci)"

 Scenario: A link out of the source folder stops sync
  Given the project
   """
   {"ai-skills.yaml":"sources:\n  - path: skills/a\nsettings:\n  target: out\n  add_relations: true\n",
    "skills/a/SKILL.md":"---\nname: a\n---\n[b](../b/SKILL.md)\n","skills/b/SKILL.md":"---\nname: b\n---\n"}
   """
  When I run aism "sync"
  Then aism fails
  And the project path "out" does not exist

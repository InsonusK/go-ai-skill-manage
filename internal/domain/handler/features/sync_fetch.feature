Feature: Sync starts by loading and checking the skills

 Scenario: Sync stops at the problems found while loading and checking, before any target work
  Given a repository "one" holding
   """
   {"a/SKILL.md":"---\nname: a\n---\n[gone](./gone.md)\n"}
   """
  And the sources are
   | repository | subpath | tags | exclude_from_checks |
   | one        |         |      |                     |
   | gone       |         |      |                     |
  When I run sync
  Then the issues are
   """
   [["source-acquire","local:gone","","",""],
    ["missing-link-target","local:one","a","SKILL.md","[gone](./gone.md)"]]
   """

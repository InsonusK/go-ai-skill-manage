Feature: Expand related skills
 Scenario Outline: Dependency cycles terminate and policy is enforced
  Given relation policy "<policy>"
  And relation tree
   """
   {"a.skill.md":"---\nname: a\n---\n[B](b.skill.md)","b.skill.md":"---\nname: b\n---\n[A](a.skill.md)"}
   """
  When I expand from "a.skill.md"
  Then relation names are "<names>" and error contains "<error>"
  Examples:
   | policy | names | error |
   | enabled | a,b | |
   | disabled | a | unselected-skill |
 Scenario: Errors across files are collected
  Given relation policy "enabled"
  And relation tree
   """
   {"a/SKILL.md":"---\nname: a\n---\n[x](missing-one)","a/notes.md":"[y](missing-two)"}
   """
  When I expand from "a"
  Then relation names are "a" and error contains "missing-one"
  And relation names are "a" and error contains "missing-two"

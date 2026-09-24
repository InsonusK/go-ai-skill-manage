Feature: LinkValidator checks that every link leads to an existing file and anchor in a loaded skill

 # Issues are listed as [Code, Source, Skill, SkillPath, File, Link]: File
 # is the path of the file holding the link from its skill's folder.

 Scenario: Valid links give no issues
  Given a source "repo" holding
   """
   {"one/SKILL.md":"---\nname: one\n---\n# Intro\n[guide](./docs/guide.md) [two](../two/SKILL.md) [web](https://example.com) [top](#intro) [no-ext](./docs/guide) [[two/SKILL.md]]\n","one/docs/guide.md":"# Guide\n[back](../SKILL.md)\n","two/SKILL.md":"---\nname: two\n---\n"}
   """
  And skills at "one,two" of source "repo" are selected
  When I validate links
  Then there are no issues

 Scenario: Every broken link is reported with its skill and file
  Given a source "repo" holding
   """
   {"one/SKILL.md":"---\nname: one\n---\n[a](./missing.md) [b](../../outside.md)\n","one/docs/guide.md":"[c](./nope.md)\n"}
   """
  And skills at "one" of source "repo" are selected
  When I validate links
  Then the issues are
   """
   [
    ["missing-link-target","local:repo","one","one","SKILL.md","[a](./missing.md)"],
    ["path-escape","local:repo","one","one","SKILL.md","[b](../../outside.md)"],
    ["missing-link-target","local:repo","one","one","docs/guide.md","[c](./nope.md)"]
   ]
   """
  And issue 1 message contains "one/missing.md"

 Scenario: A link in a flat skill is checked from the skill's own file
  Given a source "repo" holding
   """
   {"flat.skill.md":"---\nname: flat\n---\n[a](./missing.md)\n"}
   """
  And skills at "flat.skill.md" of source "repo" are selected
  When I validate links
  Then the issues are
   """
   [["missing-link-target","local:repo","flat","flat.skill.md","flat.skill.md","[a](./missing.md)"]]
   """

 Scenario: A link to a skill that is not selected is an issue when add relations is off
  Given a source "repo" holding
   """
   {"one/SKILL.md":"---\nname: one\n---\n[two](../two/SKILL.md)\n","two/SKILL.md":"---\nname: two\n---\n"}
   """
  And add relations is "false"
  And skills at "one" of source "repo" are selected
  When I validate links
  Then the issues are
   """
   [["unselected-skill","local:repo","one","one","SKILL.md","[two](../two/SKILL.md)"]]
   """
  And the loaded skills are "one"

 Scenario: A link to a skill that is not selected loads it when add relations is on
  Given a source "repo" holding
   """
   {"one/SKILL.md":"---\nname: one\n---\n[two](../two/docs/x.md)\n","two/SKILL.md":"---\nname: two\n---\n","two/docs/x.md":"x","three/SKILL.md":"---\nname: three\n---\n"}
   """
  And add relations is "true"
  And skills at "one" of source "repo" are selected
  When I validate links
  Then there are no issues
  And the loaded skills are "one,two"

 Scenario: A skill loaded by a link is checked too, and so are the skills it loads
  Given a source "repo" holding
   """
   {"one/SKILL.md":"---\nname: one\n---\n[two](../two/SKILL.md)\n","two/SKILL.md":"---\nname: two\n---\n[three](../three/SKILL.md)\n","three/SKILL.md":"---\nname: three\n---\n[gone](./gone.md)\n"}
   """
  And add relations is "true"
  And skills at "one" of source "repo" are selected
  When I validate links
  Then the loaded skills are "one,two,three"
  And the issues are
   """
   [["missing-link-target","local:repo","three","three","SKILL.md","[gone](./gone.md)"]]
   """

 Scenario Outline: A link to a file outside every skill is an issue either way
  Given a source "repo" holding
   """
   {"one/SKILL.md":"---\nname: one\n---\n[readme](../README.md)\n","README.md":"hi"}
   """
  And add relations is "<relations>"
  And skills at "one" of source "repo" are selected
  When I validate links
  Then the issue codes are "<code>"
  Examples:
   | relations | code             |
   | false     | unselected-skill |
   | true      | skill-not-found  |

 Scenario: A link to an invalid skill reports why that skill cannot be loaded
  Given a source "repo" holding
   """
   {"one/SKILL.md":"---\nname: one\n---\n[two](../two/SKILL.md)\n","two/SKILL.md":"---\nname: two\n---\n","two/inner/SKILL.md":"---\nname: inner\n---\n"}
   """
  And add relations is "true"
  And skills at "one" of source "repo" are selected
  When I validate links
  Then the issue codes are "nested-skill"
  And issue 1 message contains "inner/SKILL.md"
  And the loaded skills are "one"

 Scenario Outline: A link's anchor must exist in its target file
  Given a source "repo" holding
   """
   {"one/SKILL.md":"---\nname: one\n---\n# Intro\n<link>\n","one/docs/guide.md":"# Getting Started!\n## Getting Started!\nText ^note\n<a id=\"top\"></a>\n```\n# Hidden\n```\n","one/docs/data.txt":"# Data\n"}
   """
  And skills at "one" of source "repo" are selected
  When I validate links
  Then the issue codes are "<codes>"
  Examples:
   | link                                       | codes          |
   | [g](./docs/guide.md#getting-started)       |                |
   | [g](./docs/guide.md#getting-started-1)     |                |
   | [g](./docs/guide.md#Getting-Started)       |                |
   | [g](./docs/guide.md#getting%2Dstarted)     |                |
   | [[one/docs/guide.md#Getting Started!]]     |                |
   | [g](./docs/guide.md#^note)                 |                |
   | [g](./docs/guide.md#top)                   |                |
   | [g](./docs/guide#getting-started)          |                |
   | [g](#intro)                                |                |
   | [g](./docs/guide.md#getting-started-2)     | missing-anchor |
   | [g](./docs/guide.md#hidden)                | missing-anchor |
   | [g](./docs/guide.md#nothing)               | missing-anchor |
   | [g](./docs/guide.md#^other)                | missing-anchor |
   | [g](#outro)                                | missing-anchor |
   | [g](./docs/data.txt#data)                  | missing-anchor |

 Scenario: Links in folders excluded from checks and in non-markdown files are not checked
  Given a source "repo" holding
   """
   {"one/SKILL.md":"---\nname: one\n---\n","one/examples/app/README.md":"[x](./missing.md)\n","one/data.txt":"[x](./missing.md)\n","one/docs/examples/page.md":"[x](./missing.md)\n"}
   """
  And skills at "one" of source "repo" are selected
  When I validate links
  Then the issues are
   """
   [["missing-link-target","local:repo","one","one","docs/examples/page.md","[x](./missing.md)"]]
   """

Feature: FetchAndValidateSkills loads what the sources select and checks it

 # The catalog is listed as repository:skill in load order; issues as
 # [Code, Source, Skill, File, Link].

 Scenario: Valid sources give a catalog of the selected skills and no issues
  Given a repository "one" holding
   """
   {"a/SKILL.md":"---\nname: a\n---\n[b](../b/SKILL.md)\n","b/SKILL.md":"---\nname: b\n---\n","c/SKILL.md":"---\nname: c\n---\n"}
   """
  And a repository "two" holding
   """
   {"d.skill.md":"---\nname: d\n---\n"}
   """
  And the sources are
   | repository | subpath | tags | exclude_from_checks |
   | one        | a       |      |                     |
   | one        | b       |      |                     |
   | two        |         |      |                     |
  When I fetch and validate skills
  Then there are no issues
  And the catalog holds "one:a,one:b,two:d"

 Scenario: Problems of several sources are all reported, the valid skills are still loaded
  Given a repository "nested" holding
   """
   {"a/SKILL.md":"---\nname: a\n---\n","a/x/SKILL.md":"---\nname: x\n---\n","ok/SKILL.md":"---\nname: ok\n---\n"}
   """
  And a repository "links" holding
   """
   {"b/SKILL.md":"---\nname: b\n---\n[gone](./gone.md)\n"}
   """
  And the sources are
   | repository | subpath    | tags | exclude_from_checks |
   | gone       |            |      |                     |
   | nested     |            |      |                     |
   | links      | b,missing  |      |                     |
  When I fetch and validate skills
  Then the issues are
   """
   [["source-acquire","local:gone","","",""],
    ["nested-skill","local:nested","a","x/SKILL.md",""],
    ["missing-subpath","local:links","","missing",""],
    ["missing-link-target","local:links","b","SKILL.md","[gone](./gone.md)"]]
   """
  And the catalog holds "nested:ok,links:b"

 Scenario: Without add relations a link to a skill no source selects is an issue
  Given a repository "one" holding
   """
   {"a/SKILL.md":"---\nname: a\n---\n[b](../b/SKILL.md)\n","b/SKILL.md":"---\nname: b\n---\n[gone](./gone.md)\n"}
   """
  And the sources are
   | repository | subpath | tags | exclude_from_checks |
   | one        | a       |      |                     |
  And add relations is "false"
  When I fetch and validate skills
  Then the issues are
   """
   [["unselected-skill","local:one","a","SKILL.md","[b](../b/SKILL.md)"]]
   """
  And the catalog holds "one:a"

 Scenario: With add relations the linked skill is loaded and checked too
  Given a repository "one" holding
   """
   {"a/SKILL.md":"---\nname: a\n---\n[b](../b/SKILL.md)\n","b/SKILL.md":"---\nname: b\n---\n[gone](./gone.md)\n"}
   """
  And the sources are
   | repository | subpath | tags | exclude_from_checks |
   | one        | a       |      |                     |
  And add relations is "true"
  When I fetch and validate skills
  Then the issues are
   """
   [["missing-link-target","local:one","b","SKILL.md","[gone](./gone.md)"]]
   """
  And the catalog holds "one:a,one:b"

 # Tags are not applied yet (see AGENTS.md): back when the tag filter is.
 @todo
 Scenario: A skill the tags reject is not loaded: it is neither checked nor a relation without add relations
  Given a repository "one" holding
   """
   {"a/SKILL.md":"---\nname: a\ntags: [go]\n---\n[b](../b/SKILL.md)\n","b/SKILL.md":"---\nname: b\n---\n","b/x/SKILL.md":"---\nname: x\n---\n","c/SKILL.md":"---\nname: a\n---\n"}
   """
  And the sources are
   | repository | subpath | tags | exclude_from_checks |
   | one        |         | go   |                     |
  When I fetch and validate skills
  Then the issues are
   """
   [["unselected-skill","local:one","a","SKILL.md","[b](../b/SKILL.md)"]]
   """
  And the catalog holds "one:a"

 Scenario: The same skill name in two sources is reported for each of them
  Given a repository "one" holding
   """
   {"x/SKILL.md":"---\nname: guide\n---\n"}
   """
  And a repository "two" holding
   """
   {"y/SKILL.md":"---\nname: guide\n---\n"}
   """
  And the sources are
   | repository | subpath | tags | exclude_from_checks |
   | one        |         |      |                     |
   | two        |         |      |                     |
  When I fetch and validate skills
  Then the issues are
   """
   [["duplicate-name","local:one","guide","",""],
    ["duplicate-name","local:two","guide","",""]]
   """

 Scenario: Folders excluded everywhere and by a source add up, per source
  Given a repository "one" holding
   """
   {"a/SKILL.md":"---\nname: a\n---\n","a/examples/SKILL.md":"---\nname: e\n---\n","a/demo/page.md":"[gone](./gone.md)\n"}
   """
  And a repository "two" holding
   """
   {"b/SKILL.md":"---\nname: b\n---\n","b/examples/page.md":"[gone](./gone.md)\n","b/demo/page.md":"[gone](./gone.md)\n"}
   """
  And folders excluded from checks everywhere are "examples"
  And the sources are
   | repository | subpath | tags | exclude_from_checks |
   | one        |         |      | demo                |
   | two        |         |      |                     |
  When I fetch and validate skills
  Then the issues are
   """
   [["missing-link-target","local:two","b","demo/page.md","[gone](./gone.md)"]]
   """
  And the catalog holds "one:a,two:b"

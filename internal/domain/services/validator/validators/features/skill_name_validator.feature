Feature: SkillNameValidator checks that no two loaded skills share a name

 # Issues are listed as [Code, Source, Skill, SkillPath, File, Link].

 Scenario: Unique names give no issues
  Given a source "a" holding
   """
   {"one/SKILL.md":"---\nname: one\n---\n","two.skill.md":"---\nname: two\n---\n"}
   """
  And skills at "." of source "a" are selected
  When I validate skill names
  Then there are no issues

 Scenario: The same name in two sources is reported once for each skill
  Given a source "a" holding
   """
   {"guide/SKILL.md":"---\nname: guide\n---\n"}
   """
  And a source "b" holding
   """
   {"x/guide/SKILL.md":"---\nname: guide\n---\n"}
   """
  And skills at "." of source "a" are selected
  And skills at "." of source "b" are selected
  When I validate skill names
  Then the issues are
   """
   [
    ["duplicate-name","local:a","guide","guide","",""],
    ["duplicate-name","local:b","guide","x/guide","",""]
   ]
   """
  And issue 1 message contains "also defined at local:b x/guide"
  And issue 2 message contains "also defined at local:a guide"

 Scenario: The same name twice in one source, one of them a flat skill
  Given a source "a" holding
   """
   {"guide/SKILL.md":"---\nname: guide\n---\n",
    "other/guide.skill.md":"---\nname: guide\n---\n"}
   """
  And skills at "." of source "a" are selected
  When I validate skill names
  Then the issues are
   """
   [
    ["duplicate-name","local:a","guide","guide","",""],
    ["duplicate-name","local:a","guide","other/guide.skill.md","",""]
   ]
   """

 Scenario: A duplicate loaded while following links is found when links are validated first
  Given a source "a" holding
   """
   {"one/SKILL.md":"---\nname: one\n---\n[dup](../dup/SKILL.md)\n","dup/SKILL.md":"---\nname: one\n---\n"}
   """
  And add relations is "true"
  And skills at "one" of source "a" are selected
  When I validate skill names
  Then there are no issues
  When I validate links and then skill names
  Then the loaded skills are "one,one"
  And the issue codes are "duplicate-name,duplicate-name"

 Scenario Outline: skill-name-validator may run without link-validator, but never before it
  When I register validators "<order>"
  Then <result>
  Examples:
   | order                               | result                                                                                   |
   | link-validator,skill-name-validator | registration succeeds                                                                    |
   | skill-name-validator                | registration succeeds                                                                    |
   | link-validator                      | registration succeeds                                                                    |
   | skill-name-validator,link-validator | registration fails with "validator "link-validator" must be registered before "skill-name-validator"" |

Feature: Every issue reports itself as one row a printer understands

 Scenario Outline: A skill issue places itself as source, skill, file, link
  Given a skill issue <issue>
  When I report the issue
  Then the report row is <row>
  And the error text is "<text>"
  Examples:
   | issue | row | text |
   | {"Code":"missing-link-target","Source":"local:repo","Skill":"guide","SkillPath":"a/guide","File":"docs/x.md","Link":"[x](./y.md)","Message":"link target does not exist"} | {"Code":"missing-link-target","Message":"link target does not exist","Where":[{"Kind":"source","Value":"local:repo"},{"Kind":"skill","Value":"guide (a/guide)"},{"Kind":"file","Value":"docs/x.md"},{"Kind":"link","Value":"[x](./y.md)"}],"Details":null} | E302 missing-link-target: local:repo guide (a/guide) docs/x.md [x](./y.md): link target does not exist |
   | {"Code":"missing-subpath","Source":"local:repo","File":"a/missing","Message":"no such folder"} | {"Code":"missing-subpath","Message":"no such folder","Where":[{"Kind":"source","Value":"local:repo"},{"Kind":"file","Value":"a/missing"}],"Details":null} | E203 missing-subpath: local:repo a/missing: no such folder |
   | {"Code":"nested-skill","Skill":"guide","File":"b/SKILL.md","Message":"nested"} | {"Code":"nested-skill","Message":"nested","Where":[{"Kind":"skill","Value":"guide"},{"Kind":"file","Value":"b/SKILL.md"}],"Details":null} | E206 nested-skill: guide b/SKILL.md: nested |
   | {"Code":"shared-file-links","Source":"local:repo","File":"shared/x.md","Message":"check its links:","Details":["[a](./a.md)","[[b]]"]} | {"Code":"shared-file-links","Message":"check its links:","Where":[{"Kind":"source","Value":"local:repo"},{"Kind":"file","Value":"shared/x.md"}],"Details":["[a](./a.md)","[[b]]"]} | W311 shared-file-links: local:repo shared/x.md: check its links: |
   | {"Code":"canceled","Message":"context canceled"} | {"Code":"canceled","Message":"context canceled","Where":[],"Details":null} | E901 canceled: context canceled |

 Scenario Outline: A config issue places itself as source, setting
  Given a config issue <issue>
  When I report the issue
  Then the report row is <row>
  And the error text is "<text>"
  Examples:
   | issue | row | text |
   | {"Code":"invalid-tags","Source":"local:repo","Setting":"sources[1].tags","Message":"invalid tag expression"} | {"Code":"invalid-tags","Message":"invalid tag expression","Where":[{"Kind":"source","Value":"local:repo"},{"Kind":"setting","Value":"sources[1].tags"}],"Details":null} | E101 invalid-tags: local:repo sources[1].tags: invalid tag expression |
   | {"Code":"duplicate-target","Setting":"targets[1].path","Message":"same path"} | {"Code":"duplicate-target","Message":"same path","Where":[{"Kind":"setting","Value":"targets[1].path"}],"Details":null} | duplicate-target: targets[1].path: same path |

 Scenario Outline: A target issue places itself as target, skill
  Given a target issue <issue>
  When I report the issue
  Then the report row is <row>
  And the error text is "<text>"
  Examples:
   | issue | row | text |
   | {"Code":"unmanaged-target","Target":"/p/.claude/skills","Skill":"guide","Message":"not ours"} | {"Code":"unmanaged-target","Message":"not ours","Where":[{"Kind":"target","Value":"/p/.claude/skills"},{"Kind":"skill","Value":"guide"}],"Details":null} | E401 unmanaged-target: /p/.claude/skills guide: not ours |
   | {"Code":"target-write","Target":"/p/out","Message":"disk full"} | {"Code":"target-write","Message":"disk full","Where":[{"Kind":"target","Value":"/p/out"}],"Details":null} | target-write: /p/out: disk full |

 Scenario: A list of issues is one error line per issue
  Given skill issues [{"Code":"a","File":"x.md","Message":"one"},{"Code":"b","Message":"two"}]
  Then the error text is "a: x.md: one\nb: two"

 Scenario Outline: A declared code is printed with its number, an undeclared one as it is
  Then the label of code "<code>" is "<label>"
  Examples:
   | code                | label                    |
   | missing-link-target | E302 missing-link-target |
   | invalid-tags        | E101 invalid-tags        |
   | unmanaged-target    | E401 unmanaged-target    |
   | shared-file-links   | W311 shared-file-links   |
   | not-declared        | not-declared             |

 Scenario: Every declared code has its own number, whatever its letter: a letter and three digits
  Then every declared code has a unique number like "^[EW][1-9][0-9]{2}$"

 Scenario Outline: The letter of a code's number is its severity; an undeclared code is an error
  Then the severity of code "<code>" is "<severity>"
  Examples:
   | code                | severity |
   | missing-link-target | error    |
   | shared-file-links   | warning  |
   | deprecated-setting  | warning  |
   | not-declared        | error    |

 Scenario Outline: Only errors stop the work; the rest of a list are its warnings
  Given skill issues <issues>
  Then the list has errors "<errors>"
  And the warnings of the list are "<warnings>"
  Examples:
   | issues | errors | warnings |
   | [] | false | |
   | [{"Code":"shared-file-links"},{"Code":"when-to-use-both"}] | false | shared-file-links,when-to-use-both |
   | [{"Code":"shared-file-links"},{"Code":"missing-anchor"}] | true | shared-file-links |
   | [{"Code":"not-declared"}] | true | |

 Scenario: Code keeps its codes in the registry, not as text
  Then no Go file under "../../../.." writes an issue code as text

Feature: Every issue reports itself as one row a printer understands

 Scenario Outline: A skill issue places itself as source, skill, file, link
  Given a skill issue <issue>
  When I report the issue
  Then the report row is <row>
  And the error text is "<text>"
  Examples:
   | issue | row | text |
   | {"Code":"missing-link-target","Source":"local:repo","Skill":"guide","SkillPath":"a/guide","File":"docs/x.md","Link":"[x](./y.md)","Message":"link target does not exist"} | {"Code":"missing-link-target","Message":"link target does not exist","Where":[{"Kind":"source","Value":"local:repo"},{"Kind":"skill","Value":"guide (a/guide)"},{"Kind":"file","Value":"docs/x.md"},{"Kind":"link","Value":"[x](./y.md)"}]} | missing-link-target: local:repo guide (a/guide) docs/x.md [x](./y.md): link target does not exist |
   | {"Code":"missing-subpath","Source":"local:repo","File":"a/missing","Message":"no such folder"} | {"Code":"missing-subpath","Message":"no such folder","Where":[{"Kind":"source","Value":"local:repo"},{"Kind":"file","Value":"a/missing"}]} | missing-subpath: local:repo a/missing: no such folder |
   | {"Code":"nested-skill","Skill":"guide","File":"b/SKILL.md","Message":"nested"} | {"Code":"nested-skill","Message":"nested","Where":[{"Kind":"skill","Value":"guide"},{"Kind":"file","Value":"b/SKILL.md"}]} | nested-skill: guide b/SKILL.md: nested |
   | {"Code":"canceled","Message":"context canceled"} | {"Code":"canceled","Message":"context canceled","Where":[]} | canceled: context canceled |

 Scenario Outline: A config issue places itself as source, setting
  Given a config issue <issue>
  When I report the issue
  Then the report row is <row>
  And the error text is "<text>"
  Examples:
   | issue | row | text |
   | {"Code":"invalid-tags","Source":"local:repo","Setting":"sources[1].tags","Message":"invalid tag expression"} | {"Code":"invalid-tags","Message":"invalid tag expression","Where":[{"Kind":"source","Value":"local:repo"},{"Kind":"setting","Value":"sources[1].tags"}]} | invalid-tags: local:repo sources[1].tags: invalid tag expression |
   | {"Code":"duplicate-target","Setting":"targets[1].path","Message":"same path"} | {"Code":"duplicate-target","Message":"same path","Where":[{"Kind":"setting","Value":"targets[1].path"}]} | duplicate-target: targets[1].path: same path |

 Scenario Outline: A target issue places itself as target, skill
  Given a target issue <issue>
  When I report the issue
  Then the report row is <row>
  And the error text is "<text>"
  Examples:
   | issue | row | text |
   | {"Code":"unmanaged-target","Target":"/p/.claude/skills","Skill":"guide","Message":"not ours"} | {"Code":"unmanaged-target","Message":"not ours","Where":[{"Kind":"target","Value":"/p/.claude/skills"},{"Kind":"skill","Value":"guide"}]} | unmanaged-target: /p/.claude/skills guide: not ours |
   | {"Code":"target-write","Target":"/p/out","Message":"disk full"} | {"Code":"target-write","Message":"disk full","Where":[{"Kind":"target","Value":"/p/out"}]} | target-write: /p/out: disk full |

 Scenario: A list of issues is one error line per issue
  Given skill issues [{"Code":"a","File":"x.md","Message":"one"},{"Code":"b","Message":"two"}]
  Then the error text is "a: x.md: one\nb: two"

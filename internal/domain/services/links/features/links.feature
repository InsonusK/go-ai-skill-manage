Feature: Resolve link targets (deferred, used only by relations)

 # Gherkin table cells unescape "\\" to "\".
 Scenario Outline: Links of files under a skip folder are not followed
  When I check whether "<file>" is in skip folders "<skip>"
  Then the file is skipped: <skipped>
  Examples:
   | file                   | skip     | skipped |
   | examples/main.md       | examples | true    |
   | docs/examples/a.md     | examples | true    |
   | docs\\examples\\a.md   | examples | true    |
   | docs/main.md           | examples | false   |
   | examples-old/main.md   | examples | false   |
   | examples/main.md       |          | false   |

 Scenario Outline: Raw paths follow repository conventions
  Given source paths "docs/a.md,guide.md"
  When I resolve "<raw>" from "docs/main.md"
  Then the resolved path is "<path>" and error contains "<error>"
  Examples:
   | raw | path | error |
   | ./a | docs/a.md | |
   | ../guide.md | guide.md | |
   | guide.md | guide.md | |
   | /source/guide.md | guide.md | |
   | absent | | missing-link |
   | ../../outside | | path-escape |

 Scenario: Missing path inside a selected skill follows Python ownership resolution
  Given source paths "guide/SKILL.md"
  And known skill root "guide"
  When I resolve "./templates/example.md" from "guide/SKILL.md"
  Then the resolved path is "guide/templates/example.md" and error contains ""

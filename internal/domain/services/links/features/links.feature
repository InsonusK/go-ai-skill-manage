Feature: Extract and classify references
 Scenario: Markdown and wiki links preserve labels and fragments
  Given link document
   """
   [Guide](./guide.md#intro) ![[images/a.png|image]] [[notes.skill#part]]
   """
  When I extract links from "docs/main.md"
  Then references are
   """
   [{"text":"Guide","path":"./guide.md","fragment":"#intro","image":false,"excluded":false},{"text":"image","path":"images/a.png","fragment":"","image":true,"excluded":false},{"text":"notes.skill","path":"notes.skill","fragment":"#part","image":false,"excluded":false}]
   """
 Scenario: Inline code and example fences are excluded
  Given link document
   """
   `[inline](bad)` [web](https://example.com) [anchor](#part)
   ```example
   [sample](bad)
   ```
   [real](docs/ok.md)
   """
  When I extract links from "main.md"
  Then references are
   """
   [{"text":"inline","path":"bad","fragment":"","image":false,"excluded":true},{"text":"web","path":"https://example.com","fragment":"","image":false,"excluded":true},{"text":"anchor","path":"","fragment":"#part","image":false,"excluded":true},{"text":"sample","path":"bad","fragment":"","image":false,"excluded":true},{"text":"real","path":"docs/ok.md","fragment":"","image":false,"excluded":false}]
   """
 Scenario: Folder exclusions apply to documents
  Given link document
   """
   [ignore](bad)
   """
  When I extract links from "examples/main.md"
  Then references are
   """
   [{"text":"ignore","path":"bad","fragment":"","image":false,"excluded":true}]
   """
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

 Scenario: Inline code in a link label excludes the reference for Python compatibility
  Given link document
   """
   [`template.md`](./missing.md) [normal](exists.md)
   """
  When I extract links from "main.md"
  Then references are
   """
   [{"text":"`template.md`","path":"./missing.md","fragment":"","image":false,"excluded":true},{"text":"normal","path":"exists.md","fragment":"","image":false,"excluded":false}]
   """

 Scenario: Searcher satisfies the entity.LinkSearcher contract used by File.Links
  Given link document
   """
   [Guide](./guide.md#intro)
   """
  When I search links via the entity.LinkSearcher contract
  Then references are
   """
   [{"text":"Guide","path":"./guide.md","fragment":"#intro","image":false}]
   """

 Scenario: Missing path inside a selected skill follows Python ownership resolution
  Given source paths "guide/SKILL.md"
  And known skill root "guide"
  When I resolve "./templates/example.md" from "guide/SKILL.md"
  Then the resolved path is "guide/templates/example.md" and error contains ""

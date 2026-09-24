Feature: WikilinkParser finds and parses wikilinks

 # Gherkin table cells unescape "\|" to "|" and "\\" to "\".
 Scenario Outline: Parse a raw wikilink
  When I parse the wikilink <raw>
  Then the parsed link has text "<text>" path "<path>" fragment "<fragment>" image <image> external <external>
  Examples:
   | raw                           | text        | path                | fragment | image | external |
   | [[notes.skill#part]]          | notes.skill | notes.skill         | #part    | false | false    |
   | [[docs/guide\|Guide]]         | Guide       | docs/guide          |          | false | false    |
   | ![[images/a.png\|image]]      | image       | images/a.png        |          | true  | false    |
   | [[docs/deep/page]]            | page        | docs/deep/page      |          | false | false    |
   | [[#part]]                     |             |                     | #part    | false | false    |
   | [[docs/a\\\|A]]               | A           | docs/a              |          | false | false    |
   | [[a\|b\|c]]                   | c           | a\|b                |          | false | false    |
   | [[https://example.com\|site]] | site        | https://example.com |          | false | true     |

 Scenario Outline: Parse rejects text that is not exactly one wikilink
  When I parse the wikilink <raw>
  Then parsing fails with "invalid-link"
  Examples:
   | raw               |
   | [Guide](guide.md) |
   | [[]]              |
   | see [[notes]]     |
   | [[a]] [[b]]       |

 Scenario: Find returns wikilink spans
  When I find wikilinks in
   """
   [[notes]] text ![[images/a.png|image]]
   """
  Then the found spans are
   """
   [[0,9],[15,38]]
   """

 Scenario: Find ignores markdown links
  When I find wikilinks in
   """
   [Guide](./guide.md) ![img](a.png)
   """
  Then the found spans are
   """
   []
   """

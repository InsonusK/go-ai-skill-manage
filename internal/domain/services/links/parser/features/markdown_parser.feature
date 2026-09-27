Feature: MarkdownParser finds and parses markdown links

 # Gherkin table cells unescape "\|" to "|" and "\\" to "\".
 Scenario Outline: Parse a raw markdown link
  When I parse the markdown link <raw>
  Then the parsed link has text "<text>" path "<path>" fragment "<fragment>" image <image>
  Examples:
   | raw                               | text                | path                | fragment | image |
   | [Guide](./guide.md#intro)         | Guide               | ./guide.md          | #intro   | false |
   | ![alt](img/a.png)                 | alt                 | img/a.png           |          | true  |
   | [](a.md)                          |                     | a.md                |          | false |
   | [anchor](#part)                   | anchor              |                     | #part    | false |
   | [x]()                             | x                   |                     |          | false |
   | [web](https://example.com/a#b)    | web                 | https://example.com/a | #b     | false |
   | [mail](MAILTO:a@b.c)              | mail                | MAILTO:a@b.c        |          | false |
   | [see [x] here](a.md)              | see [x] here        | a.md                |          | false |
   | [![build](badge.svg)](docs/ci.md) | ![build](badge.svg) | docs/ci.md          |          | false |

 Scenario Outline: Parse rejects text that is not exactly one markdown link
  When I parse the markdown link <raw>
  Then parsing fails with "invalid-link"
  Examples:
   | raw                   |
   | [[notes]]             |
   | see [a](b.md)         |
   | [a](b.md) [c](d.md)   |
   | [a](b.md "title")     |
   | [unbalanced [x](a.md) |
   | [a](b.md              |

 Scenario: Find returns markdown spans
  When I find markdown links in
   """
   [Guide](./guide.md) text ![img](a.png)
   """
  Then the found spans are
   """
   [[0,19],[25,38]]
   """

 Scenario: Find returns a link whose label holds brackets as one span
  When I find markdown links in
   """
   [see [x] here](a.md)
   """
  Then the found spans are
   """
   [[0,20]]
   """

 Scenario: Find returns both a linked image and the link around it
  When I find markdown links in
   """
   [![build](badge.svg)](docs/ci.md)
   """
  Then the found spans are
   """
   [[0,33],[1,20]]
   """

 Scenario: Find ignores wikilinks and plain text
  When I find markdown links in
   """
   plain [[notes]] ![[images/a.png|image]] text
   """
  Then the found spans are
   """
   []
   """

Feature: LinkFactory collects links from its registered parsers

 # Fake parsers report the spans they are told to and parse any raw text
 # into a ParsedLink whose Format is the parser's name and whose Path is the raw
 # text it received. Start and End are left empty by the fake, so the
 # factory is the only thing that can fill them.

 Scenario: Links from every registered parser are merged in content order
  Given fake parsers "md,wiki" are registered
  And fake parser "md" reports spans "10-14,0-4"
  And fake parser "wiki" reports spans "5-9"
  When I search links in
   """
   aaaa bbbb cccc
   """
  Then the searched links are
   """
   [
    {"Start":0,"End":4,"Text":"","Path":"aaaa","Fragment":"","Format":"md","Image":false},
    {"Start":5,"End":9,"Text":"","Path":"bbbb","Fragment":"","Format":"wiki","Image":false},
    {"Start":10,"End":14,"Text":"","Path":"cccc","Fragment":"","Format":"md","Image":false}
   ]
   """

 Scenario: Each parser parses exactly the text of its own spans
  Given fake parsers "md,wiki" are registered
  And fake parser "md" reports spans "0-4"
  And fake parser "wiki" reports spans "5-9,10-14"
  When I search links in
   """
   aaaa bbbb cccc
   """
  Then fake parser "md" parsed
   """
   ["aaaa"]
   """
  And fake parser "wiki" parsed
   """
   ["bbbb","cccc"]
   """

 Scenario: Touching spans are not a conflict
  Given fake parsers "md,wiki" are registered
  And fake parser "md" reports spans "0-4"
  And fake parser "wiki" reports spans "4-8"
  When I search links in
   """
   aaaabbbb
   """
  Then the searched links are
   """
   [
    {"Start":0,"End":4,"Text":"","Path":"aaaa","Fragment":"","Format":"md","Image":false},
    {"Start":4,"End":8,"Text":"","Path":"bbbb","Fragment":"","Format":"wiki","Image":false}
   ]
   """

 Scenario: Overlapping spans of different parsers are rejected
  Given fake parsers "md,wiki" are registered
  And fake parser "md" reports spans "0-6"
  And fake parser "wiki" reports spans "4-9"
  When I search links in
   """
   aaaabbbbb
   """
  Then searching fails with "link-overlap"
  And the search error mentions "aaaabb"
  And the search error mentions "bbbbb"

 Scenario: A span nested inside another is rejected
  Given fake parsers "md,wiki" are registered
  And fake parser "md" reports spans "0-9"
  And fake parser "wiki" reports spans "3-5"
  When I search links in
   """
   aaabbbccc
   """
  Then searching fails with "link-overlap"

 Scenario: A parse failure is propagated
  Given fake parsers "md" are registered
  And fake parser "md" reports spans "0-4"
  And fake parser "md" fails to parse "aaaa"
  When I search links in
   """
   aaaa
   """
  Then searching fails with "invalid-link"

 Scenario Outline: Invalid spans are rejected
  Given fake parsers "md" are registered
  And fake parser "md" reports spans "<span>"
  When I search links in
   """
   aaaa
   """
  Then searching fails with "invalid-link-span"
  Examples:
   | span |
   | 2-2  |
   | 3-1  |
   | 2-5  |

 Scenario: A factory without parsers finds nothing
  Given fake parsers "" are registered
  When I search links in
   """
   [Guide](./guide.md) [[notes]]
   """
  Then the searched links are
   """
   []
   """

 Scenario: Default factory finds markdown and wiki links in content order
  Given the default link factory
  When I search links in
   """
   [[notes.skill#part]] [Guide](./guide.md#intro) ![[images/a.png|image]]
   """
  Then the searched links are
   """
   [
    {"Start":0,"End":20,"Text":"notes.skill","Path":"notes.skill","Fragment":"#part","Format":"wikilink","Image":false},
    {"Start":21,"End":46,"Text":"Guide","Path":"./guide.md","Fragment":"#intro","Format":"markdown","Image":false},
    {"Start":47,"End":70,"Text":"image","Path":"images/a.png","Fragment":"","Format":"wikilink","Image":true}
   ]
   """

 Scenario: Default factory rejects a wikilink inside a markdown target
  Given the default link factory
  When I search links in
   """
   [a]([[b]])
   """
  Then searching fails with "link-overlap"

 Scenario: Default factory rejects a linked image as two links over the same text
  Given the default link factory
  When I search links in
   """
   [![build](badge.svg)](docs/ci.md)
   """
  Then searching fails with "link-overlap"

 Scenario: Parsers search the content masked by every excluder in turn
  Given fake parsers "md" are registered
  And fake excluder "first" hides spans "2-4"
  And fake excluder "second" hides spans "6-8"
  When I search links in
   """
   0123456789
   """
  Then fake excluder "second" searched
   """
   ["01  456789"]
   """
  And fake parser "md" searched
   """
   ["01  45  89"]
   """

 Scenario: A span touching excluded text is dropped before it is parsed
  Given fake parsers "md" are registered
  And fake parser "md" reports spans "0-5,7-9"
  And fake excluder "code" hides spans "3-6"
  When I search links in
   """
   aa `b` cc
   """
  Then the searched link raws are
   """
   ["cc"]
   """
  And fake parser "md" parsed
   """
   ["cc"]
   """

 Scenario: Default factory skips links in inline code and example fences but keeps external and anchor links
  Given the default link factory
  When I search links in
   """
   `[inline](bad)` [web](https://example.com) [anchor](#part)
   ```example
   [sample](bad)
   ```
   ``[a](`x`)`` [real](docs/ok.md)
   """
  Then the searched link raws are
   """
   ["[web](https://example.com)","[anchor](#part)","[real](docs/ok.md)"]
   """

 Scenario: Default factory drops a link whose label is inline code
  Given the default link factory
  When I search links in
   """
   [`template.md`](./missing.md) [normal](exists.md)
   """
  Then the searched link raws are
   """
   ["[normal](exists.md)"]
   """

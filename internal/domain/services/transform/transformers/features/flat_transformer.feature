Feature: FlatTransformer lays each skill out as {name}/SKILL.md and rewrites links to match

 # The target is listed as "{skill dir}/{file path}" -> content.

 Scenario: Every format of skill becomes a folder named after it with SKILL.md
  Given a repository "repo" holding
   """
   {"a/b/guide/SKILL.md":"---\nname: guide\n---\n","a/b/guide/docs/x.md":"x","a/b/guide/run.sh":"echo",
    "h.skill/h.skill.md":"---\nname: h\n---\n","h.skill/y.md":"y",
    "f.skill.md":"---\nname: f\n---\n"}
   """
  And skills at "." of "repo" are loaded
  When I make the target catalog
  And I flatten the target catalog
  Then the target holds
   """
   {"guide/SKILL.md":"---\nname: guide\n---\n","guide/docs/x.md":"x","guide/run.sh":"echo",
    "h/SKILL.md":"---\nname: h\n---\n","h/y.md":"y",
    "f/SKILL.md":"---\nname: f\n---\n"}
   """
  And the target skills are
   """
   [["guide","guide","guide/SKILL.md","agent-dir"],["f","f","f/SKILL.md","agent-dir"],["h","h","h/SKILL.md","agent-dir"]]
   """

 Scenario Outline: A link is rewritten to lead to the same file at its new place
  Given a repository "repo" holding
   """
   {"a/b/guide/SKILL.md":"---\nname: guide\n---\n","a/b/guide/docs/x.md":"# Top\n","a/b/guide/docs/i.png":"png",
    "h.skill/h.skill.md":"---\nname: h\n---\n# Top\n","h.skill/y.md":"see <link>\n",
    "f.skill.md":"---\nname: f\n---\n"}
   """
  And skills at "." of "repo" are loaded
  When I make the target catalog
  And I flatten the target catalog
  Then the target holds
   """
   {"guide/SKILL.md":"---\nname: guide\n---\n","guide/docs/x.md":"# Top\n","guide/docs/i.png":"png",
    "h/SKILL.md":"---\nname: h\n---\n# Top\n","h/y.md":"see <rewritten>\n",
    "f/SKILL.md":"---\nname: f\n---\n"}
   """
  Examples:
   | link                                                | rewritten                                   |
   | [m](./h.skill.md)                                   | [m](./SKILL.md)                             |
   | [m](./h.skill.md#top)                               | [m](./SKILL.md#top)                         |
   | [x](../a/b/guide/docs/x.md#top)                     | [x](../guide/docs/x.md#top)                 |
   | [x](a/b/guide/docs/x.md)                            | [x](../guide/docs/x.md)                     |
   | [x](../a/b/guide/docs/x)                            | [x](../guide/docs/x.md)                     |
   | [d](../a/b/guide/docs)                              | [d](../guide/docs)                          |
   | [g](../a/b/guide)                                   | [g](../guide)                               |
   | [f](../f.skill.md)                                  | [f](../f/SKILL.md)                          |
   | ![i](../a/b/guide/docs/i.png)                       | ![i](../guide/docs/i.png)                   |
   | [[a/b/guide/docs/x.md#top\|X]]                      | [X](../guide/docs/x.md#top)                 |
   | [[a/b/guide/docs/x.md]]                             | [x.md](../guide/docs/x.md)                  |
   | [[#top]]                                            | [top](#top)                                 |

 Scenario Outline: A link that already leads to the right place, a web link and an anchor stay as written
  Given a repository "repo" holding
   """
   {"a/guide/SKILL.md":"---\nname: guide\n---\nsee <link>\n","a/guide/docs/x.md":"# Top\n"}
   """
  And skills at "." of "repo" are loaded
  When I make the target catalog
  And I flatten the target catalog
  Then the target holds
   """
   {"guide/SKILL.md":"---\nname: guide\n---\nsee <link>\n","guide/docs/x.md":"# Top\n"}
   """
  Examples:
   | link                       |
   | [x](./docs/x.md)           |
   | [x](./docs/x.md#top)       |
   | [w](https://example.com/a) |
   | [t](#top)                  |

 Scenario: Links of skills in two repositories stay in their own repository
  Given a repository "one" holding
   """
   {"s/SKILL.md":"---\nname: one\n---\n[d](./docs/d.md)\n","s/docs/d.md":"[back](../SKILL.md)\n"}
   """
  And a repository "two" holding
   """
   {"s/SKILL.md":"---\nname: two\n---\n[d](./docs/d.md)\n","s/docs/d.md":"[back](../SKILL.md)\n"}
   """
  And skills at "." of "one" are loaded
  And skills at "." of "two" are loaded
  When I make the target catalog
  And I flatten the target catalog
  Then the target holds
   """
   {"one/SKILL.md":"---\nname: one\n---\n[d](./docs/d.md)\n","one/docs/d.md":"[back](../SKILL.md)\n",
    "two/SKILL.md":"---\nname: two\n---\n[d](./docs/d.md)\n","two/docs/d.md":"[back](../SKILL.md)\n"}
   """

 Scenario: Files excluded from checks and non-markdown files keep their links as written
  Given a repository "repo" holding
   """
   {"h.skill/h.skill.md":"---\nname: h\n---\n","h.skill/examples/app.md":"[m](../h.skill.md) [gone](./gone.md)\n","h.skill/data.txt":"[m](./h.skill.md)\n"}
   """
  And folders excluded from checks in "repo" are "examples"
  And skills at "." of "repo" are loaded
  When I make the target catalog
  And I flatten the target catalog
  Then the target holds
   """
   {"h/SKILL.md":"---\nname: h\n---\n","h/examples/app.md":"[m](../h.skill.md) [gone](./gone.md)\n","h/data.txt":"[m](./h.skill.md)\n"}
   """

 Scenario: A file changed before flattening is a bug: the flat transformer runs first
  Given a repository "repo" holding
   """
   {"a/guide/SKILL.md":"---\nname: guide\n---\n","a/guide/docs/x.md":"x"}
   """
  And skills at "." of "repo" are loaded
  When I make the target catalog
  And file "docs/x.md" of "guide" is already changed to "y"
  And I flatten the target catalog
  Then the flattening panics with "must run first"

 Scenario: A link outside every loaded skill is a bug: validation must have rejected it
  Given a repository "repo" holding
   """
   {"a/guide/SKILL.md":"---\nname: guide\n---\n[n](../../notes.md)\n","notes.md":"n"}
   """
  And skills at "a" of "repo" are loaded
  When I make the target catalog
  And I flatten the target catalog
  Then the flattening panics with "outside every loaded skill"

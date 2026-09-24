Feature: A Link knows the file it is written in and the path and skill it points to

 Background:
  Given a repository "/src" holding
   """
   {"a/SKILL.md":"---\nname: one\n---\n","a/guide.md":"g","a/docs/guide.md":"g","a/docs/page.md":"p","b/SKILL.md":"---\nname: two\n---\n","b/x.md":"x","b/y":"no extension"}
   """

 Scenario Outline: MakeLink reads Raw from the file content and tells a web link by its path
  Given the file "docs/intro.md" of the skill at "a" contains
   """
   see <link> here
   """
  When I read the first link of that file
  Then the link raw is "<link>" and external is <external>
  Examples:
   | link                     | external |
   | [G](./guide.md)          | false    |
   | ![img](./guide.md)       | false    |
   | [W](https://example.com) | true     |
   | [W](HTTP://example.com)  | true     |
   | [M](mailto:a@b.c)        | true     |
   | [F](ftp://x/y)           | true     |
   | [F](file:///x/y)         | true     |

 Scenario: MakeLink refuses a span outside the file content
  Given the file "docs/intro.md" of the skill at "a" contains
   """
   short
   """
  When I make a link of that file from span 2-100
  Then making the link fails with "invalid-link-span"

 Scenario Outline: A link's target path is read in the form it is written in
  Given the file "docs/intro.md" of the skill at "a" contains
   """
   <link>
   """
  When I read the first link of that file
  Then the link path as "os-absolute" is "<os>"
  And the link path as "repo-absolute" is "<repo>"
  And the link path as "file-relative" is "<file>"
  Examples:
   | link                  | os                    | repo            | file         |
   | [G](./guide.md)       | /src/a/docs/guide.md  | a/docs/guide.md | ./guide.md   |
   | [G](../guide.md)      | /src/a/guide.md       | a/guide.md      | ../guide.md  |
   | [G](..\guide.md)      | /src/a/guide.md       | a/guide.md      | ../guide.md  |
   | [G](b/x.md)           | /src/b/x.md           | b/x.md          | ../../b/x.md |
   | [G](/src/b/x.md)      | /src/b/x.md           | b/x.md          | ../../b/x.md |
   | [G](#part)            | /src/a/docs/intro.md  | a/docs/intro.md | ./intro.md   |
   | [G](.)                | /src/a/docs           | a/docs          | .            |
   | [[b/x.md]]            | /src/b/x.md           | b/x.md          | ../../b/x.md |

 Scenario Outline: A link with the ".md" left out points to the ".md" file, written explicitly
  Given the file "docs/intro.md" of the skill at "a" contains
   """
   <link>
   """
  When I read the first link of that file
  Then the link path as "repo-absolute" is "<repo>"
  And the link path as "file-relative" is "<file>"
  Examples:
   | link              | repo           | file          |
   | [P](./page)       | a/docs/page.md | ./page.md     |
   | [P](./page#part)  | a/docs/page.md | ./page.md     |
   | [[a/docs/page]]   | a/docs/page.md | ./page.md     |
   | [X](../../b/x)    | b/x.md         | ../../b/x.md  |
   | [Y](../../b/y)    | b/y            | ../../b/y     |

 Scenario Outline: A link whose target is outside the repository, missing, or on the web has no path
  Given the file "docs/intro.md" of the skill at "a" contains
   """
   <link>
   """
  When I read the first link of that file
  Then the link path as "repo-absolute" fails with "<error>"
  And the link skill fails with "<error>"
  And the resolver was asked for ""
  Examples:
   | link                     | error               |
   | [W](https://example.com) | web-link            |
   | [E](../../../x.md)       | path-escape         |
   | [E](/other/x.md)         | path-escape         |
   | [M](./missing)           | missing-link-target |

 Scenario: A link's skill is found through the resolver, searching up from the target path
  Given the file "docs/intro.md" of the skill at "a" contains
   """
   [X](../../b/x)
   """
  And the resolver finds the skill at "b"
  When I read the first link of that file
  Then the link skill is the skill at "b"
  And the resolver was asked for "local:repo b/x.md"

 Scenario: A link's skill-relative path starts from the folder of the skill holding the target
  Given the file "docs/intro.md" of the skill at "a" contains
   """
   [X](../../b/x.md)
   """
  And the resolver finds the skill at "b"
  When I read the first link of that file
  Then the link path as "skill-relative" is "x.md"

 Scenario: A link's skill-relative path inside its own skill
  Given the file "docs/intro.md" of the skill at "a" contains
   """
   [G](./guide.md)
   """
  And the resolver finds the skill at "a"
  When I read the first link of that file
  Then the link path as "skill-relative" is "docs/guide.md"

 Scenario: Only the skill-relative path asks the resolver
  Given the file "docs/intro.md" of the skill at "a" contains
   """
   [G](./guide.md)
   """
  When I read the first link of that file
  Then the link path as "os-absolute" is "/src/a/docs/guide.md"
  And the link path as "repo-absolute" is "a/docs/guide.md"
  And the link path as "file-relative" is "./guide.md"
  And the resolver was asked for ""

 Scenario: A skill the resolver has not loaded is reported as not cached
  Given the file "docs/intro.md" of the skill at "a" contains
   """
   [X](../../b/x.md)
   """
  And the resolver has the skill not cached
  When I read the first link of that file
  Then the link path as "skill-relative" fails as not cached

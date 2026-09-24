Feature: PathInRepo converts a path inside a repository between its written forms

 Scenario Outline: A written path's form is detected from how it starts
  Then the written path "<path>" is detected as "<kind>"
  Examples:
   | path            | kind          |
   | /src/a/b.md     | os-absolute   |
   | C:/src/a/b.md   | os-absolute   |
   | ./b.md          | file-relative |
   | ../x/b.md       | file-relative |
   | .               | file-relative |
   | ..              | file-relative |
   | a/b.md          | repo-absolute |
   | b               | repo-absolute |
   | .hidden/b.md    | repo-absolute |

 Scenario Outline: A path read in one form is written in every other form
  When I read the path "<path>" written as "<kind>" from "<base>" in repository "/src"
  Then the path written as "os-absolute" from "" is "<os>"
  And the path written as "repo-absolute" from "" is "<repo>"
  And the path written as "file-relative" from "a/docs" is "<fromDocs>"
  And the path written as "skill-relative" from "a" is "<fromSkill>"
  Examples:
   | path             | kind           | base   | os                | repo        | fromDocs      | fromSkill    |
   | /src/a/b/c.md    | os-absolute    |        | /src/a/b/c.md     | a/b/c.md    | ../b/c.md     | b/c.md       |
   | a/b/c.md         | repo-absolute  | x      | /src/a/b/c.md     | a/b/c.md    | ../b/c.md     | b/c.md       |
   | ./c.md           | file-relative  | a/b    | /src/a/b/c.md     | a/b/c.md    | ../b/c.md     | b/c.md       |
   | ../b/c.md        | file-relative  | a/docs | /src/a/b/c.md     | a/b/c.md    | ../b/c.md     | b/c.md       |
   | b/c.md           | skill-relative | a      | /src/a/b/c.md     | a/b/c.md    | ../b/c.md     | b/c.md       |
   | ./intro.md       | file-relative  | a/docs | /src/a/docs/intro.md | a/docs/intro.md | ./intro.md | docs/intro.md |
   | /src             | os-absolute    |        | /src              | .           | ../..         | ..           |
   | a/docs           | repo-absolute  |        | /src/a/docs       | a/docs      | .             | docs         |

 Scenario: A skill-relative path of a skill at the repository folder is its repository path
  When I read the path "a/b.md" written as "repo-absolute" from "" in repository "/src"
  Then the path written as "skill-relative" from "" is "a/b.md"

 Scenario Outline: A path leaving the repository is refused
  When I read the path "<path>" written as "<kind>" from "<base>" in repository "/src"
  Then reading the path fails with "path-escape"
  Examples:
   | path          | kind          | base |
   | /other/a.md   | os-absolute   |      |
   | /srcx/a.md    | os-absolute   |      |
   | ../a.md       | file-relative |      |
   | ../../a.md    | file-relative | a    |
   | ../a.md       | repo-absolute |      |
   | /a.md         | os-absolute   |      |

 Scenario: An OS path cannot be read against a repository without a known folder
  When I read the path "/src/a.md" written as "os-absolute" from "" in repository ""
  Then reading the path fails with "path-escape"

 Scenario: An unknown form is refused both ways
  When I read the path "a.md" written as "web" from "" in repository "/src"
  Then reading the path fails with "unknown path kind"
  When I read the path "a.md" written as "repo-absolute" from "" in repository "/src"
  Then writing the path as "web" fails with "unknown path kind"

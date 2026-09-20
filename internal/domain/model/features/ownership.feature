Feature: Resolve skill ownership and relative output paths
 Scenario Outline: Ownership and relative-path rules
  Given a skill rooted at "<root>" main "<main>" flat "<flat>"
  Then path "<path>" is owned "<owned>" and relative is "<relative>"
  Examples:
   | root       | main          | flat  | path              | owned | relative          |
   | docs       | docs/SKILL.md | false | docs/SKILL.md     | true  | SKILL.md          |
   | docs       | docs/SKILL.md | false | docs              | true  | SKILL.md          |
   | docs       | docs/SKILL.md | false | docs/guide.md     | true  | guide.md          |
   | docs       | docs/SKILL.md | false | other/file.md     | false | other/file.md     |
   | a.skill.md | a.skill.md    | true  | a.skill.md        | true  | SKILL.md          |
   | a.skill.md | a.skill.md    | true  | a.skill.md.bak    | false | a.skill.md.bak    |
   | .          | SKILL.md      | false | anything/here.md  | true  | anything/here.md  |

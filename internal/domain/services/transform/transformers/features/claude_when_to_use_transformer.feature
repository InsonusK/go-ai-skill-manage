Feature: ClaudeWhenToUseTransformer renames whenToUse to Claude Code's when_to_use

 # The target is listed as "{skill dir}/{file path}" -> content; the
 # catalog isn't flattened here, so skill dirs are the source ones.

 Scenario Outline: whenToUse becomes when_to_use, a list joined with ", "
  Given a repository "repo" holding
   """
   {"a/SKILL.md":"---\nname: a\ndescription: d\n<frontmatter>\n---\n# Body\n","a/docs/x.md":"x"}
   """
  And skills at "." of "repo" are loaded
  When I make the target catalog
  And I apply the claude when-to-use transformer
  Then the target holds
   """
   {"a/SKILL.md":"---\ndescription: d\nname: a\nwhen_to_use: <value>\n---\n# Body\n","a/docs/x.md":"x"}
   """
  And the log has no WARN
  Examples:
   | frontmatter                     | value              |
   | whenToUse: on review            | on review          |
   | whenToUse: [review, refactor]   | review, refactor   |

 Scenario: A skill without whenToUse is left byte for byte
  Given a repository "repo" holding
   """
   {"a/SKILL.md":"---\nname: a\n# comment\ndescription: d\n---\n"}
   """
  And skills at "." of "repo" are loaded
  When I make the target catalog
  And I apply the claude when-to-use transformer
  Then the target holds
   """
   {"a/SKILL.md":"---\nname: a\n# comment\ndescription: d\n---\n"}
   """

 Scenario: A skill with both keeps when_to_use and whenToUse, with a warning
  Given a repository "repo" holding
   """
   {"a/SKILL.md":"---\nname: a\nwhenToUse: old\nwhen_to_use: native\n---\n"}
   """
  And skills at "." of "repo" are loaded
  When I make the target catalog
  And I apply the claude when-to-use transformer
  Then the target holds
   """
   {"a/SKILL.md":"---\nname: a\nwhenToUse: old\nwhen_to_use: native\n---\n"}
   """
  And the log has WARN "skill=a"

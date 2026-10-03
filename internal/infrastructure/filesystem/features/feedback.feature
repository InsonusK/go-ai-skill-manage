Feature: Store reads a skill's marker; FeedbackDrafts keeps feedback drafts as files the user reads, edits and commits

 # Drafts lie in .ai-skills/feedback of the target folder used as the
 # project; the loaded draft is {ID, Status, Kind, Skill, Source, Commit,
 # SkillPath, Title, Body, IssueURL, CreatedAt, SentAt}.

 Scenario: The marker of a managed skill tells where the skill comes from
  Given an empty target
  And the marker of "guide" is
   """
   {"source":{"type":"github","path":"https://github.com/o/r","tree":"main"},"commit":"c0ffee","skill_path":"a/guide","transformers":["flat"],"version":"go-3"}
   """
  When I read the marker of "guide"
  Then the marker read is
   """
   {"source":{"type":"github","path":"https://github.com/o/r","tree":"main"},"commit":"c0ffee","skill_path":"a/guide","transformers":["flat"],"version":"go-3"}
   """

 Scenario Outline: A skill is not managed when <case>
  Given an empty target
  And target fixture file "other/.ai-skills-managed" contains "{}"
  And unmanaged target directory "plain"
  And target fixture file "file" contains "x"
  And target symlink "linked" points to "other"
  And target fixture file "markerdir/.ai-skills-managed/x" contains "x"
  When I read the marker of "<skill>"
  Then the skill is not managed

  Examples:
   | case                            | skill     |
   | there is no folder              | missing   |
   | the folder has no marker        | plain     |
   | the entry is a file             | file      |
   | the entry is a symlink          | linked    |
   | the marker isn't a regular file | markerdir |

 Scenario: A skill is not managed when the target folder doesn't exist
  Given an empty target
  When I read the marker of "guide" in a target folder that doesn't exist
  Then the skill is not managed

 Scenario: A marker of the older format asks to run sync
  Given an empty target
  And the marker of "guide" is
   """
   {"source":"local:repo","skill_path":"a/guide"}
   """
  When I read the marker of "guide"
  Then reading the marker fails with "written by an older version? run sync"

 Scenario: A skill name that isn't one folder name is refused
  Given an empty target
  When I read the marker of "../guide"
  Then reading the marker fails with "unsafe skill name"

 Scenario: A draft is written as frontmatter, the title heading and the body, and read back the same
  Given an empty target
  When I create the draft "2026-09-27-guide-broken-anchor" of a "bug" on "guide" titled "Broken anchor" with body "The anchor #x is missing.\n\nSee SKILL.md."
  Then the created id is "2026-09-27-guide-broken-anchor"
  And the draft file "2026-09-27-guide-broken-anchor.md" is
   """
   ---
   status: draft
   kind: bug
   skill: guide
   source:
     type: github
     path: https://github.com/o/r
     tree: main
   commit: c0ffee
   skill_path: a/guide
   created_at: 2026-09-27T10:00:00Z
   ---

   # Broken anchor

   The anchor #x is missing.

   See SKILL.md.
   """
  When I load the draft "2026-09-27-guide-broken-anchor"
  Then the loaded draft is
   """
   {"ID":"2026-09-27-guide-broken-anchor","Status":"draft","Kind":"bug","Skill":"guide",
    "Source":{"type":"github","path":"https://github.com/o/r","tree":"main"},"Commit":"c0ffee","SkillPath":"a/guide",
    "Title":"Broken anchor","Body":"The anchor #x is missing.\n\nSee SKILL.md.","IssueURL":"","CreatedAt":"2026-09-27T10:00:00Z","SentAt":""}
   """

 Scenario: A taken id gets the first free suffix
  Given an empty target
  When I create the draft "2026-09-27-guide" of a "bug" on "guide" titled "A" with body "a"
  And I create the draft "2026-09-27-guide" of a "bug" on "guide" titled "B" with body "b"
  Then the created id is "2026-09-27-guide-2"
  When I create the draft "2026-09-27-guide" of a "bug" on "guide" titled "C" with body "c"
  Then the created id is "2026-09-27-guide-3"

 Scenario: The drafts are listed by id, of any status; other files aren't drafts
  Given an empty target
  When I list the drafts
  Then the drafts listed are ""
  Given I create the draft "b" of a "bug" on "guide" titled "B" with body "b"
  And I create the draft "a" of a "bug" on "guide" titled "A" with body "a"
  And I load the draft "a"
  And I mark the loaded draft sent as "https://github.com/o/r/issues/7" at "2026-09-27T11:00:00Z" and save it
  And the draft file "notes.txt" contains
   """
   x
   """
  And the draft file "Upper.md" contains
   """
   x
   """
  And the draft file "folder.md/x" contains
   """
   x
   """
  When I list the drafts
  Then the drafts listed are "a,b"

 Scenario: A sent draft records the issue and when it was sent
  Given an empty target
  And I create the draft "d" of a "improvement" on "guide" titled "T" with body "B"
  And I load the draft "d"
  When I mark the loaded draft sent as "https://github.com/o/r/issues/7" at "2026-09-27T11:00:00Z" and save it
  Then the draft file "d.md" is
   """
   ---
   status: sent
   kind: improvement
   skill: guide
   source:
     type: github
     path: https://github.com/o/r
     tree: main
   commit: c0ffee
   skill_path: a/guide
   created_at: 2026-09-27T10:00:00Z
   issue_url: https://github.com/o/r/issues/7
   sent_at: 2026-09-27T11:00:00Z
   ---

   # T

   B
   """

 Scenario: A draft edited by hand is read as the user left it, CRLF line ends too
  Given an empty target
  And the draft file "d.md" contains
   """
   ---\r
   status: draft\r
   kind: bug\r
   skill: guide\r
   source: {type: github, path: https://github.com/o/r}\r
   skill_path: guide\r
   created_at: 2026-09-27T10:00:00Z\r
   ---\r
   \r
   # Edited title  \r
   \r
   Edited body.\r
   """
  When I load the draft "d"
  Then the loaded draft is
   """
   {"ID":"d","Status":"draft","Kind":"bug","Skill":"guide",
    "Source":{"type":"github","path":"https://github.com/o/r"},"Commit":"","SkillPath":"guide",
    "Title":"Edited title","Body":"Edited body.","IssueURL":"","CreatedAt":"2026-09-27T10:00:00Z","SentAt":""}
   """

 Scenario Outline: A draft file that <case> is refused
  Given an empty target
  And the draft file "d.md" contains
   """
   <content>
   """
  When I load the draft "d"
  Then the drafts fail with "<error>"

  Examples:
   | case                     | content                           | error                                   |
   | has no frontmatter       | # T                               | no frontmatter                          |
   | never closes frontmatter | ---                               | frontmatter isn't closed                |
   | has no title heading     | ---\nstatus: draft\n---\n\nbody   | the text must start with the title      |

 Scenario Outline: An id that isn't a plain draft name is refused (<id>)
  Given an empty target
  When I load the draft "<id>"
  Then the drafts fail with "invalid feedback id"
  When I save a draft "<id>"
  Then the drafts fail with "invalid feedback id"

  Examples:
   | id           |
   | ../escape    |
   | d.md         |
   | Upper        |

 Scenario: Only an existing draft is saved
  Given an empty target
  When I save a draft "missing"
  Then the drafts fail with "no feedback missing"

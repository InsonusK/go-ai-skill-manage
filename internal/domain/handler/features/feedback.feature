Feature: Feedback about a skill goes to its source's issue tracker only as the user confirmed it

 # Drafts are compared as {ID, Status, Kind, Skill, Source, Commit,
 # SkillPath, Title, Body, IssueURL, CreatedAt, SentAt}; the fake tracker
 # answers with https://github.com/o/r/issues/<n>.

 Background:
  Given the managed skills are
   | target            | skill | type   | path                  | tree | commit | skill_path |
   | /p/.claude/skills | guide | github | https://github.com/o/r | main | c0ffee | a/guide    |
   | /p/.claude/skills | notes | local  | /home/me/skills       |      |        | notes      |
   | /p/.claude/skills | odd   | gitlab | https://gitlab.com/o/r |      |        | odd        |
   | /p/.agents/skills | guide | github | https://github.com/o/other |  | beef   | guide      |
  And the feedback targets are "/p/.claude/skills,/p/.agents/skills"
  And it is "2026-09-27T10:00:00Z"

 Scenario: A draft takes the skill's source from the first target holding it and shows the issue it would open
  When I draft a "bug" feedback on "guide" titled "Broken anchor" with body "The anchor #x is missing."
  Then the feedback succeeds
  And the draft "2026-09-27-guide-broken-anchor" is
   """
   {"ID":"2026-09-27-guide-broken-anchor","Status":"draft","Kind":"bug","Skill":"guide",
    "Source":{"type":"github","path":"https://github.com/o/r","tree":"main"},"Commit":"c0ffee","SkillPath":"a/guide",
    "Title":"Broken anchor","Body":"The anchor #x is missing.","IssueURL":"","CreatedAt":"2026-09-27T10:00:00Z","SentAt":""}
   """
  And the shown issue is
   """
   {"title":"Broken anchor","labels":["bug"],
    "body":"The anchor #x is missing.\n\n---\nSkill: `guide` (`a/guide` at commit `c0ffee`)\nSource: github https://github.com/o/r (tree `main`)\nSent with ai-skill-manager 1.2.0\n"}
   """
  And the tracker opened
   """
   []
   """

 Scenario: An improvement is labeled enhancement
  When I draft an "improvement" feedback on "guide" titled "Add an example" with body "An example of X would help."
  Then the shown issue is
   """
   {"title":"Add an example","labels":["enhancement"],
    "body":"An example of X would help.\n\n---\nSkill: `guide` (`a/guide` at commit `c0ffee`)\nSource: github https://github.com/o/r (tree `main`)\nSent with ai-skill-manager 1.2.0\n"}
   """

 Scenario: Drafts of the same day and title get free ids; a title without latin letters gives no slug
  When I draft a "bug" feedback on "guide" titled "Broken anchor" with body "a"
  And I draft a "bug" feedback on "guide" titled "Broken anchor" with body "b"
  And I draft a "bug" feedback on "guide" titled "Сломан якорь" with body "c"
  Then the drafts are "2026-09-27-guide,2026-09-27-guide-broken-anchor,2026-09-27-guide-broken-anchor-2"

 Scenario Outline: A draft is refused for <case>
  When I draft a "<kind>" feedback on "<skill>" titled "<title>" with body "<body>"
  Then the feedback fails with "<error>"
  And the drafts are ""

  Examples:
   | case                      | kind     | skill   | title | body | error                                                         |
   | a skill in no target      | bug      | missing | T     | B    | skill "missing" is in none of the targets                     |
   | a local source            | bug      | notes   | T     | B    | comes from the local source /home/me/skills: edit it there by hand |
   | a source with no tracker  | bug      | odd     | T     | B    | no issue tracker for source type "gitlab"                     |
   | an unknown kind           | question | guide   | T     | B    | unknown feedback kind "question"                              |
   | an empty title            | bug      | guide   |       | B    | feedback title is empty                                       |
   | an empty body             | bug      | guide   | T     |      | feedback body is empty                                        |

 Scenario: Sending with the shown hash opens the issue and marks the draft sent
  Given I draft a "bug" feedback on "guide" titled "Broken anchor" with body "The anchor #x is missing."
  And it is "2026-09-27T10:05:00Z"
  When I send "2026-09-27-guide-broken-anchor" with the shown hash
  Then the feedback succeeds
  And the tracker opened
   """
   [{"source":{"type":"github","path":"https://github.com/o/r","tree":"main"},
     "issue":{"title":"Broken anchor","labels":["bug"],
      "body":"The anchor #x is missing.\n\n---\nSkill: `guide` (`a/guide` at commit `c0ffee`)\nSource: github https://github.com/o/r (tree `main`)\nSent with ai-skill-manager 1.2.0\n"}}]
   """
  And the draft "2026-09-27-guide-broken-anchor" is
   """
   {"ID":"2026-09-27-guide-broken-anchor","Status":"sent","Kind":"bug","Skill":"guide",
    "Source":{"type":"github","path":"https://github.com/o/r","tree":"main"},"Commit":"c0ffee","SkillPath":"a/guide",
    "Title":"Broken anchor","Body":"The anchor #x is missing.","IssueURL":"https://github.com/o/r/issues/1",
    "CreatedAt":"2026-09-27T10:00:00Z","SentAt":"2026-09-27T10:05:00Z"}
   """

 Scenario: A hash other than the shown one sends nothing
  Given I draft a "bug" feedback on "guide" titled "Broken anchor" with body "x"
  When I send "2026-09-27-guide-broken-anchor" with hash "0000"
  Then the feedback fails with "changed since it was confirmed"
  And the tracker opened
   """
   []
   """

 Scenario: A draft edited after it was shown is sent only after it is shown again
  Given I draft a "bug" feedback on "guide" titled "Broken anchor" with body "x"
  And the user edits the body of "2026-09-27-guide-broken-anchor" to "edited"
  When I send "2026-09-27-guide-broken-anchor" with the shown hash
  Then the feedback fails with "changed since it was confirmed"
  When I preview "2026-09-27-guide-broken-anchor"
  And I send "2026-09-27-guide-broken-anchor" with the shown hash
  Then the feedback succeeds
  And the tracker opened
   """
   [{"source":{"type":"github","path":"https://github.com/o/r","tree":"main"},
     "issue":{"title":"Broken anchor","labels":["bug"],
      "body":"edited\n\n---\nSkill: `guide` (`a/guide` at commit `c0ffee`)\nSource: github https://github.com/o/r (tree `main`)\nSent with ai-skill-manager 1.2.0\n"}}]
   """

 Scenario: A draft edited to an empty body is not sent even with its own hash
  Given I draft a "bug" feedback on "guide" titled "Broken anchor" with body "x"
  And the user edits the body of "2026-09-27-guide-broken-anchor" to " "
  And I preview "2026-09-27-guide-broken-anchor"
  When I send "2026-09-27-guide-broken-anchor" with the shown hash
  Then the feedback fails with "feedback body is empty"

 Scenario: A sent draft is not sent again
  Given I draft a "bug" feedback on "guide" titled "Broken anchor" with body "x"
  And I send "2026-09-27-guide-broken-anchor" with the shown hash
  When I send "2026-09-27-guide-broken-anchor" with the shown hash
  Then the feedback fails with "feedback 2026-09-27-guide-broken-anchor is already sent"

 Scenario: A declined draft stays as a trace and can't be sent
  Given I draft a "bug" feedback on "guide" titled "Broken anchor" with body "x"
  When I decline "2026-09-27-guide-broken-anchor"
  Then the feedback succeeds
  When I send "2026-09-27-guide-broken-anchor" with the shown hash
  Then the feedback fails with "is already declined"
  And the drafts are "2026-09-27-guide-broken-anchor"
  And the tracker opened
   """
   []
   """

 Scenario: Pending feedback is the drafts neither sent nor declined, by id
  Given I draft a "bug" feedback on "guide" titled "C" with body "x"
  And I draft a "bug" feedback on "guide" titled "A" with body "x"
  And I draft a "bug" feedback on "guide" titled "B" with body "x"
  And I draft a "bug" feedback on "guide" titled "D" with body "x"
  And I preview "2026-09-27-guide-b"
  And I send "2026-09-27-guide-b" with the shown hash
  And I decline "2026-09-27-guide-d"
  When I ask for the pending feedback
  Then the feedback succeeds
  And the pending feedback is "2026-09-27-guide-a,2026-09-27-guide-c"

 Scenario: Without drafts nothing is pending
  When I ask for the pending feedback
  Then the feedback succeeds
  And the pending feedback is ""

 Scenario: A tracker failure leaves the draft a draft
  Given I draft a "bug" feedback on "guide" titled "Broken anchor" with body "x"
  And the tracker fails with "401 Bad credentials"
  When I send "2026-09-27-guide-broken-anchor" with the shown hash
  Then the feedback fails with "401 Bad credentials"
  And the draft "2026-09-27-guide-broken-anchor" is
   """
   {"ID":"2026-09-27-guide-broken-anchor","Status":"draft","Kind":"bug","Skill":"guide",
    "Source":{"type":"github","path":"https://github.com/o/r","tree":"main"},"Commit":"c0ffee","SkillPath":"a/guide",
    "Title":"Broken anchor","Body":"x","IssueURL":"","CreatedAt":"2026-09-27T10:00:00Z","SentAt":""}
   """

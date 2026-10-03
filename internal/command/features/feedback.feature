Feature: Command line: feedback -- the agent drafts, the user confirms and sends

 # The skill guide was synced into "out" from a GitHub source; notes from a
 # local one. The fake tracker answers https://github.com/o/r/issues/<n>.

 Background:
  Given CLI project
   """
   {"ai-skills.yaml":"sources:\n  - path: skills\ntarget: out\n",
    "skills/notes/SKILL.md":"---\nname: notes\n---\n",
    "out/guide/SKILL.md":"---\nname: guide\n---\n",
    "out/guide/.ai-skills-managed":"{\"source\":{\"type\":\"github\",\"path\":\"https://github.com/o/r\",\"tree\":\"main\"},\"commit\":\"c0ffee\",\"skill_path\":\"a/guide\",\"transformers\":[\"flat\"],\"version\":\"go-3\"}",
    "out/notes/.ai-skills-managed":"{\"source\":{\"type\":\"local\",\"path\":\"/p/skills\"},\"skill_path\":\"notes\",\"transformers\":[\"flat\"],\"version\":\"go-3\"}"}
   """

 Scenario: A draft is written next to the config and nothing is sent
  When I run argv
   """
   ["feedback","draft","--skill","guide","--kind","bug","--title","Broken anchor","--body","The anchor #x is missing."]
   """
  Then exit code is "0"
  And stdout contains ".ai-skills/feedback/2026-09-27-guide-broken-anchor.md"
  And stdout contains "Repository: github https://github.com/o/r\nLabels: bug\nTitle: Broken anchor\n\nThe anchor #x is missing.\n\n---\nSkill: `guide` (`a/guide` at commit `c0ffee`)"
  And stdout contains "Nothing is sent yet. Review the draft (edit the file if needed); then the user sends it:\n  ai-skill-manager feedback send 2026-09-27-guide-broken-anchor\n"
  And project file ".ai-skills/feedback/2026-09-27-guide-broken-anchor.md" contains "status: draft"
  And the opened issues are ""

 Scenario: The body can come from stdin
  Given stdin holds "Line one.\nLine two.\n"
  When I run argv
   """
   ["feedback","draft","--skill","guide","--kind","improvement","--title","Examples","--body-file","-"]
   """
  Then exit code is "0"
  And project file ".ai-skills/feedback/2026-09-27-guide-examples.md" contains "# Examples\n\nLine one.\nLine two.\n"

 Scenario: A skill from a local source can't get feedback
  When I run argv
   """
   ["feedback","draft","--skill","notes","--kind","bug","--title","T","--body","B"]
   """
  Then exit code is "1"
  And console contains "comes from the local source /p/skills: edit it there by hand"
  And project path ".ai-skills/feedback" exists "false"

 Scenario: Without the user's terminal nothing is sent and the user is asked to run send
  Given I run argv
   """
   ["feedback","draft","--skill","guide","--kind","bug","--title","T","--body","B"]
   """
  And stdin holds "y\n"
  When I run arguments "feedback send 2026-09-27-guide-t"
  Then exit code is "1"
  And console contains "feedback send asks the user to confirm and needs a terminal: ask the user to run it:\n  ai-skill-manager feedback send 2026-09-27-guide-t\n"
  And the opened issues are ""
  And project file ".ai-skills/feedback/2026-09-27-guide-t.md" contains "status: draft"

 Scenario: The user confirms in the terminal: the issue is opened and the draft records it
  Given I run argv
   """
   ["feedback","draft","--skill","guide","--kind","bug","--title","T","--body","B"]
   """
  And the user's terminal answers "y"
  When I run arguments "feedback send .ai-skills/feedback/2026-09-27-guide-t.md"
  Then exit code is "0"
  And stdout contains "Title: T\n\nB\n"
  And stdout contains "Open this issue? It is public if the repository is. [y/N] Sent: https://github.com/o/r/issues/1\n"
  And the opened issues are "T"
  And project file ".ai-skills/feedback/2026-09-27-guide-t.md" contains "status: sent"
  And project file ".ai-skills/feedback/2026-09-27-guide-t.md" contains "issue_url: https://github.com/o/r/issues/1"
  When I run arguments "feedback send 2026-09-27-guide-t"
  Then exit code is "1"
  And console contains "feedback 2026-09-27-guide-t is already sent"
  And the opened issues are "T"

 Scenario Outline: Anything but yes sends nothing (<answer>)
  Given I run argv
   """
   ["feedback","draft","--skill","guide","--kind","bug","--title","T","--body","B"]
   """
  And the user's terminal answers "<answer>"
  When I run arguments "feedback send 2026-09-27-guide-t"
  Then exit code is "0"
  And stdout contains "Not sent."
  And the opened issues are ""

  Examples:
   | answer |
   | n      |
   |        |
   | yep    |

 Scenario: send all asks about every draft not yet sent or declined, each on its own answer
  Given I run argv
   """
   ["feedback","draft","--skill","guide","--kind","bug","--title","B","--body","b"]
   """
  And I run argv
   """
   ["feedback","draft","--skill","guide","--kind","bug","--title","A","--body","a"]
   """
  And I run argv
   """
   ["feedback","draft","--skill","guide","--kind","bug","--title","C","--body","c"]
   """
  And I run arguments "feedback decline 2026-09-27-guide-c"
  And the user's terminal answers "y\nn"
  When I run arguments "feedback send all"
  Then exit code is "0"
  And stdout contains "Feedback 2026-09-27-guide-a (1 of 2)\n\nRepository: github https://github.com/o/r\nLabels: bug\nTitle: A\n\na\n"
  And stdout contains "[y/N] Sent: https://github.com/o/r/issues/1\n\nFeedback 2026-09-27-guide-b (2 of 2)\n\n"
  And stdout contains "[y/N] Not sent.\n"
  And the opened issues are "A"
  And project file ".ai-skills/feedback/2026-09-27-guide-a.md" contains "status: sent"
  And project file ".ai-skills/feedback/2026-09-27-guide-b.md" contains "status: draft"
  Given the user's terminal answers "y"
  When I run arguments "feedback send all"
  Then exit code is "0"
  And stdout contains "Feedback 2026-09-27-guide-b (1 of 1)"
  And the opened issues are "A,B"
  And project file ".ai-skills/feedback/2026-09-27-guide-c.md" contains "status: declined"
  When I run arguments "feedback send all"
  Then exit code is "0"
  And stdout contains "No feedback drafts to send.\n"
  And the opened issues are "A,B"

 Scenario: send all without the user's terminal sends nothing
  Given I run argv
   """
   ["feedback","draft","--skill","guide","--kind","bug","--title","T","--body","B"]
   """
  And stdin holds "y\n"
  When I run arguments "feedback send all"
  Then exit code is "1"
  And console contains "needs a terminal: ask the user to run it:\n  ai-skill-manager feedback send all\n"
  And the opened issues are ""

 Scenario: send all sends the readable drafts and fails on a draft file it can't read
  Given CLI project
   """
   {"ai-skills.yaml":"sources:\n  - path: skills\ntarget: out\n",
    "skills/notes/SKILL.md":"---\nname: notes\n---\n",
    "out/guide/.ai-skills-managed":"{\"source\":{\"type\":\"github\",\"path\":\"https://github.com/o/r\"},\"skill_path\":\"guide\",\"transformers\":[],\"version\":\"go-3\"}",
    ".ai-skills/feedback/broken.md":"no frontmatter"}
   """
  And I run argv
   """
   ["feedback","draft","--skill","guide","--kind","bug","--title","T","--body","B"]
   """
  And the user's terminal answers "y"
  When I run arguments "feedback send all"
  Then exit code is "1"
  And console contains "feedback broken: no frontmatter"
  And the opened issues are "T"

 Scenario: show tells the status; a declined draft stays and can't be sent
  Given I run argv
   """
   ["feedback","draft","--skill","guide","--kind","bug","--title","T","--body","B"]
   """
  When I run arguments "feedback decline 2026-09-27-guide-t"
  Then exit code is "0"
  And stdout contains "Feedback 2026-09-27-guide-t declined; the draft stays as a trace"
  When I run arguments "feedback show 2026-09-27-guide-t"
  Then stdout contains "Feedback 2026-09-27-guide-t: declined\n\nRepository: github https://github.com/o/r"
  Given the user's terminal answers "y"
  When I run arguments "feedback send 2026-09-27-guide-t"
  Then exit code is "1"
  And console contains "feedback 2026-09-27-guide-t is already declined"
  And the opened issues are ""

 Scenario: With another config the drafts lie next to it and the send hint names it
  Given CLI project
   """
   {"sub/cfg.yaml":"sources:\n  - path: skills\ntarget: out\n",
    "sub/out/guide/.ai-skills-managed":"{\"source\":{\"type\":\"github\",\"path\":\"https://github.com/o/r\"},\"skill_path\":\"guide\",\"transformers\":[],\"version\":\"go-3\"}"}
   """
  When I run argv
   """
   ["feedback","draft","-c","sub/cfg.yaml","--skill","guide","--kind","bug","--title","T","--body","B"]
   """
  Then exit code is "0"
  And project path "sub/.ai-skills/feedback/2026-09-27-guide-t.md" exists "true"
  And stdout contains "ai-skill-manager feedback send 2026-09-27-guide-t -c sub/cfg.yaml"

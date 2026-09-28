Feature: MCP tools: the agent drafts a skill feedback, the user confirms it in the client's dialog

 # Checked on the current protocol (multi round-trip requests) and on the
 # one before it (server-initiated elicitation). The fake tracker answers
 # https://github.com/o/r/issues/<n>.

 Background:
  Given the skill "guide" comes from "github" "https://github.com/o/r"
  And the skill "notes" comes from "local" "/p/skills"

 Scenario Outline: The draft sends nothing and tells the agent what's next (<protocol>)
  Given the client speaks protocol "<protocol>"
  When the agent drafts a "bug" on "guide" titled "Broken anchor" with body "The anchor is missing."
  Then the tool result has
   """
   {"id":"2026-09-27-guide-broken-anchor","repository":"github https://github.com/o/r","labels":["bug"],"title":"Broken anchor",
    "body":"The anchor is missing.\n\n---\nSkill: `guide` (`a/guide` at commit `c0ffee`)\nSource: github https://github.com/o/r\nSent with ai-skill-manager 1.2.0\n",
    "next":"Nothing is sent. Tell the user the draft file to review (they may edit it), then call feedback_submit with id 2026-09-27-guide-broken-anchor."}
   """
  And the draft file lies in the drafts folder
  And the opened issues are ""

  Examples:
   | protocol   |
   | 2026-07-28 |
   | 2025-11-25 |

 Scenario Outline: The user sees the whole issue and ticks send: it is opened (<protocol>)
  Given the client speaks protocol "<protocol>"
  And the user answers the dialog "accept" ticking send
  When the agent drafts a "improvement" on "guide" titled "Add an example" with body "An example would help."
  And the agent submits the draft
  Then the user saw the dialog
   """
   Open an issue in github https://github.com/o/r?

   Labels: enhancement
   Title: Add an example

   An example would help.

   ---
   Skill: `guide` (`a/guide` at commit `c0ffee`)
   Source: github https://github.com/o/r
   Sent with ai-skill-manager 1.2.0

   """
  And the tool result has
   """
   {"sent":true,"issue_url":"https://github.com/o/r/issues/1","message":"Sent: https://github.com/o/r/issues/1"}
   """
  And the opened issues are "Add an example"
  And the draft is "sent"

  Examples:
   | protocol   |
   | 2026-07-28 |
   | 2025-11-25 |

 Scenario Outline: Anything but accept with send ticked sends nothing (<answer>)
  Given the client speaks protocol "2026-07-28"
  And the user answers the dialog <answer>
  When the agent drafts a "bug" on "guide" titled "T" with body "B"
  And the agent submits the draft
  Then the tool result has
   """
   {"sent":false,"message":"Not sent: the user didn't confirm. The draft stays; don't ask again unless the user wants to."}
   """
  And the opened issues are ""
  And the draft is "draft"

  Examples:
   | answer                                |
   | "accept" leaving send unticked        |
   | "decline"                             |
   | "cancel"                              |

 Scenario Outline: A client without dialogs sends nothing and points the user to the terminal (<protocol>)
  Given the client speaks protocol "<protocol>"
  And the client can't show dialogs
  When the agent drafts a "bug" on "guide" titled "T" with body "B"
  And the agent submits the draft
  Then the tool result has
   """
   {"sent":false}
   """
  And the tool result has
   """
   {"message":"Not sent: this client can't ask the user to confirm. Ask the user to review DRAFTS/2026-09-27-guide-t.md and send it from their terminal: ai-skill-manager feedback send 2026-09-27-guide-t"}
   """
  And the user saw no dialog
  And the opened issues are ""

  Examples:
   | protocol   |
   | 2026-07-28 |
   | 2025-11-25 |

 Scenario: A client showing only links (URL elicitation) gets no form and nothing is sent
  Given the client speaks protocol "2026-07-28"
  And the client shows only links, no forms
  When the agent drafts a "bug" on "guide" titled "T" with body "B"
  And the agent submits the draft
  Then the tool result has
   """
   {"sent":false,"message":"Not sent: this client can't ask the user to confirm. Ask the user to review DRAFTS/2026-09-27-guide-t.md and send it from their terminal: ai-skill-manager feedback send 2026-09-27-guide-t"}
   """
  And the user saw no dialog
  And the opened issues are ""

 Scenario: A draft edited while the dialog is open is not sent
  Given the client speaks protocol "2026-07-28"
  And the user answers the dialog "accept" ticking send
  And the user edits the draft while the dialog is open
  When the agent drafts a "bug" on "guide" titled "T" with body "B"
  And the agent submits the draft
  Then the tool fails with "changed since it was confirmed"
  And the opened issues are ""
  And the draft is "draft"

 Scenario: A sent draft is not asked about again
  Given the client speaks protocol "2026-07-28"
  And the user answers the dialog "accept" ticking send
  When the agent drafts a "bug" on "guide" titled "T" with body "B"
  And the agent submits the draft
  And the agent submits the draft
  Then the tool fails with "feedback 2026-09-27-guide-t is already sent"
  And the opened issues are "T"

 Scenario: A skill from a local source gets no draft
  Given the client speaks protocol "2026-07-28"
  When the agent drafts a "bug" on "notes" titled "T" with body "B"
  Then the tool fails with "comes from the local source /p/skills: edit it there by hand"

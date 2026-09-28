Feature: GitHub opens an issue in the repository of a source through the REST API

 Scenario Outline: The issue is opened in the repository the source URL names (<form>)
  Given GitHub answers 201 with
   """
   {"number":7,"html_url":"https://github.com/o/r/issues/7"}
   """
  When I open an issue titled "Broken anchor" with body "The anchor is missing." and labels "bug" in "<source>"
  Then the issue is "https://github.com/o/r/issues/7"
  And GitHub received
   """
   [{"Method":"POST","Path":"/repos/o/r/issues","Accept":"application/vnd.github+json","Authorization":"Bearer secret",
     "APIVersion":"2022-11-28","Body":{"title":"Broken anchor","body":"The anchor is missing.","labels":["bug"]}}]
   """

  Examples:
   | form  | source                    |
   | https | https://github.com/o/r    |
   | .git  | https://github.com/o/r.git |
   | ssh   | git@github.com:o/r.git    |

 Scenario: A source that isn't a GitHub repository sends nothing
  When I open an issue titled "T" with body "B" and labels "bug" in "https://gitlab.com/o/r"
  Then opening fails with "not a GitHub repository URL"
  And GitHub received
   """
   []
   """

 Scenario: No token sends nothing and tells how to get one
  Given the token is ""
  When I open an issue titled "T" with body "B" and labels "bug" in "https://github.com/o/r"
  Then opening fails with "no GitHub token: run gh auth login, or set GH_TOKEN (see docs/feedback.md)"
  And GitHub received
   """
   []
   """

 Scenario: A token that can't be read sends nothing
  Given the token can't be read: "gh: not logged in"
  When I open an issue titled "T" with body "B" and labels "bug" in "https://github.com/o/r"
  Then opening fails with "gh: not logged in"
  And GitHub received
   """
   []
   """

 Scenario: A refusal is reported with GitHub's message
  Given GitHub answers 410 with
   """
   {"message":"Issues are disabled for this repo"}
   """
  When I open an issue titled "T" with body "B" and labels "bug" in "https://github.com/o/r"
  Then opening fails with "GitHub issue in o/r: HTTP 410: Issues are disabled for this repo"

 Scenario: An answer without the issue's page is an error
  Given GitHub answers 201 with
   """
   {"number":7}
   """
  When I open an issue titled "T" with body "B" and labels "bug" in "https://github.com/o/r"
  Then opening fails with "no html_url in the answer"

 Scenario Outline: A refusal to see or write the repository points at the token setup (<code>)
  Given GitHub answers <code> with
   """
   {"message":"<message>"}
   """
  When I open an issue titled "T" with body "B" and labels "bug" in "https://github.com/o/r"
  Then opening fails with "GitHub issue in o/r: HTTP <code>: <message>: check that the token may open issues there (see docs/feedback.md)"

  Examples:
   | code | message                                |
   | 401  | Bad credentials                        |
   | 403  | Resource not accessible by integration |
   | 404  | Not Found                              |

 Scenario Outline: The token is looked up as gh does: GH_TOKEN, GITHUB_TOKEN, then gh (<case>)
  Given the environment has GH_TOKEN="<gh_token>"
  And the environment has GITHUB_TOKEN="<github_token>"
  And gh gives the token "from-gh"
  When I look for the GitHub token
  Then the token found is "<found>"

  Examples:
   | case            | gh_token | github_token | found       |
   | GH_TOKEN first  | gh-env   | github-env   | gh-env      |
   | then GITHUB_TOKEN |        | github-env   | github-env  |
   | then gh         |          |              | from-gh     |

 Scenario: Without gh and a variable the error tells both ways
  Given gh isn't installed
  When I look for the GitHub token
  Then looking for the token fails with "no GitHub token: install gh and run gh auth login, or set GH_TOKEN (see docs/feedback.md)"

 Scenario: A gh that isn't logged in is reported with its message
  Given gh fails with "no oauth token found for github.com"
  When I look for the GitHub token
  Then looking for the token fails with "no GitHub token from gh (no oauth token found for github.com): run gh auth login"

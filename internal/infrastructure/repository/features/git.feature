Feature: Git adapter process contract
 Scenario: Git clone arguments preserve branch and URL boundaries, then the cloned commit is read
  When I prepare a clone for branch "feature/demo"
  Then git calls are
   """
   [["clone","--depth","1","--single-branch","--branch","feature/demo","--","https://github.com/owner/repo.git","/tmp/dest"],
    ["-C","/tmp/dest","rev-parse","HEAD"]]
   """
  And the cloned commit is "c0ffee"
 Scenario: Default branch is master
  When I prepare a clone for branch ""
  Then git calls are
   """
   [["clone","--depth","1","--single-branch","--branch","master","--","https://github.com/owner/repo.git","/tmp/dest"],
    ["-C","/tmp/dest","rev-parse","HEAD"]]
   """
 Scenario: Git process executes a real local command and returns its output
  When I run Git with "version"
  Then git process error contains ""
  And git output starts with "git version"
 Scenario: Git process propagates errors
  When I run Git with "not-a-command"
  Then git process error contains "git:"
 Scenario: Git process respects cancellation
  When I run Git in a cancelled context
  Then git process error contains "context canceled"

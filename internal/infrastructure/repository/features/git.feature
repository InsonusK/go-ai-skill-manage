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
 Scenario Outline: A full commit hash is fetched by itself, not cloned as a branch
  When I prepare a clone for branch "<tree>"
  Then git calls are
   """
   [["init","--quiet","--","/tmp/dest"],
    ["-C","/tmp/dest","fetch","--quiet","--depth","1","--","https://github.com/owner/repo.git","<fetched>"],
    ["-C","/tmp/dest","checkout","--quiet","FETCH_HEAD"],
    ["-C","/tmp/dest","rev-parse","HEAD"]]
   """
  Examples:
   | tree                                                             | fetched                                                          |
   | 874b5b81d18ad24af34b2a294830f049de5c3f30                         | 874b5b81d18ad24af34b2a294830f049de5c3f30                         |
   | 874B5B81D18AD24AF34B2A294830F049DE5C3F30                         | 874b5b81d18ad24af34b2a294830f049de5c3f30                         |
   | 0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef | 0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef |

 Scenario Outline: Anything else is cloned as a branch or a tag
  When I prepare a clone for branch "<tree>"
  Then git calls are
   """
   [["clone","--depth","1","--single-branch","--branch","<tree>","--","https://github.com/owner/repo.git","/tmp/dest"],
    ["-C","/tmp/dest","rev-parse","HEAD"]]
   """
  Examples:
   | tree                                      |
   | v1.2.0                                    |
   | 874b5b8                                   |
   | 874b5b81d18ad24af34b2a294830f049de5c3f3   |
   | 874b5b81d18ad24af34b2a294830f049de5c3f3g  |

 Scenario Outline: Real git takes a branch, a tag or a commit
  Given a git repository with commits
   | message | content | tag |
   | first   | one     | v1  |
   | second  | two     |     |
  When I clone that repository at "<tree>"
  Then the clone's file "f.txt" holds "<content>"
  And the clone's commit is the "<commit>" commit
  Examples:
   | tree   | content | commit |
   | master | two     | second |
   | v1     | one     | first  |
   | @first | one     | first  |

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

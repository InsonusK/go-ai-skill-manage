Feature: Sources of the same repository don't contradict each other

 Scenario Outline: Two sources of the same repository
  When I validate configuration
   """
   sources:
     - {path: skills, <first>}
     - {path: skills, <second>}
   """
  Then the config issues with codes "duplicate-source,conflicting-exclude" are
   """
   <issues>
   """
  Examples:
   | first                                   | second                                   | issues |
   | subpath: [a]                            | subpath: [b]                             | [] |
   | subpath: [a], tags: [go]                | subpath: [a], tags: [cli]                | [] |
   | subpath: [a], tags: [go]                | subpath: [./a], tags: [go]               | [["duplicate-source","local:/project/skills@master","sources[1]"]] |
   | subpath: [a, b]                         | subpath: [b, a]                          | [["duplicate-source","local:/project/skills@master","sources[1]"]] |
   | subpath: [.]                            | tags: []                                 | [["duplicate-source","local:/project/skills@master","sources[1]"]] |
   | subpath: [a], exclude_from_checks: [x]  | subpath: [b], exclude_from_checks: [x]   | [] |
   | subpath: [a], exclude_from_checks: [x]  | subpath: [b]                             | [["conflicting-exclude","local:/project/skills@master","sources[1].exclude_from_checks"]] |
   | subpath: [a], exclude_from_checks: [x]  | subpath: [a], exclude_from_checks: [y]   | [["duplicate-source","local:/project/skills@master","sources[1]"],["conflicting-exclude","local:/project/skills@master","sources[1].exclude_from_checks"]] |

 Scenario: Different repositories are never compared
  When I validate configuration
   """
   sources:
     - {path: one, exclude_from_checks: [x]}
     - {path: two}
     - {path: one, tree: dev}
   """
  Then there are no config issues

 Scenario: Each later source is compared with the first one of its repository
  When I validate configuration
   """
   sources:
     - {path: skills, subpath: [a]}
     - {path: skills, subpath: [a]}
     - {path: skills, subpath: [a]}
   """
  Then the config issues are
   """
   [["duplicate-source","local:/project/skills@master","sources[1]"],
    ["duplicate-source","local:/project/skills@master","sources[2]"]]
   """
  And config issue 1 message contains "sources[0]"

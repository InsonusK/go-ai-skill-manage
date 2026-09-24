Feature: A resolved configuration is checked by every config validator at once

 Scenario: A valid configuration has no issues
  When I validate configuration
   """
   sources:
     - path: skills
       subpath: [a, /project/skills/b]
       tags: ["stack/go & !deprecated"]
       exclude_from_checks: [demo]
     - path: skills
       subpath: [c]
       exclude_from_checks: [demo]
     - type: github
       path: https://github.com/o/r.git
   target:
     default: {}
     claude: {}
   """
  Then there are no config issues

 Scenario: Problems of every validator are reported together, in validator order
  When I validate configuration
   """
   sources:
     - path: skills
       subpath: [../up]
       tags: ["(go"]
     - path: skills
       subpath: [../up]
       tags: ["(go"]
   target:
     one: {path: out}
     two: {path: out}
   """
  Then the config issues are
   """
   [["invalid-tags","local:/project/skills@master","sources[0].tags[0]"],
    ["invalid-tags","local:/project/skills@master","sources[1].tags[0]"],
    ["unsafe-subpath","local:/project/skills@master","sources[0].subpath[0]"],
    ["unsafe-subpath","local:/project/skills@master","sources[1].subpath[0]"],
    ["duplicate-source","local:/project/skills@master","sources[1]"],
    ["target-overlap","","targets[1].path"]]
   """

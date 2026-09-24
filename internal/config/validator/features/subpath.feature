Feature: Every subpath stays inside its source

 Scenario Outline: Subpaths of a local source
  When I validate configuration
   """
   sources:
     - path: skills
       subpath: ["<subpath>"]
   """
  Then the config issues are
   """
   <issues>
   """
  Examples:
   | subpath               | issues |
   | a/b                   | [] |
   | ./a                   | [] |
   | a/../b                | [] |
   | /project/skills       | [] |
   | /project/skills/a     | [] |
   | ../x                  | [["unsafe-subpath","local:/project/skills@master","sources[0].subpath[0]"]] |
   | a/../../x             | [["unsafe-subpath","local:/project/skills@master","sources[0].subpath[0]"]] |
   | a\\\\b                | [["unsafe-subpath","local:/project/skills@master","sources[0].subpath[0]"]] |
   | /project/other        | [["unsafe-subpath","local:/project/skills@master","sources[0].subpath[0]"]] |
   | /project/skills-extra | [["unsafe-subpath","local:/project/skills@master","sources[0].subpath[0]"]] |

 Scenario Outline: Subpaths of a github source
  When I validate configuration
   """
   sources:
     - type: github
       path: https://github.com/o/r.git
       subpath: ["<subpath>"]
   """
  Then the config issues are
   """
   <issues>
   """
  Examples:
   | subpath | issues |
   | skills  | [] |
   | /skills | [["unsafe-subpath","github:https://github.com/o/r.git@master","sources[0].subpath[0]"]] |
   | ../x    | [["unsafe-subpath","github:https://github.com/o/r.git@master","sources[0].subpath[0]"]] |

 Scenario Outline: The message says why the subpath is unsafe
  When I validate configuration
   """
   sources:
     - {type: <type>, path: <path>, subpath: ["<subpath>"]}
   """
  Then config issue 0 message contains "<reason>"
  Examples:
   | type   | path                       | subpath        | reason                        |
   | github | https://github.com/o/r.git | /skills        | is absolute in a github source |
   | local  | skills                     | /project/other | is outside the source /project/skills |
   | local  | skills                     | ../x           | leads out of the source       |
   | local  | skills                     | a\\\\b         | contains a backslash          |

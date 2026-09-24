Feature: Every tag expression of every source parses

 Scenario Outline: Tag expressions
  When I validate configuration
   """
   sources:
     - path: skills
       tags: <tags>
   """
  Then the config issues with codes "invalid-tags" are
   """
   <issues>
   """
  Examples:
   | tags                         | issues |
   | ["go"]                       | [] |
   | ["stack/go & !deprecated"]   | [] |
   | ["(go"]                      | [["invalid-tags","local:/project/skills@master","sources[0].tags[0]"]] |
   | ["go", "a &"]                | [["invalid-tags","local:/project/skills@master","sources[0].tags[1]"]] |

 Scenario: The message names the expression
  When I validate configuration
   """
   sources:
     - path: skills
       tags: ["(go"]
   """
  Then config issue 0 message contains "invalid tag expression: \"(go\""

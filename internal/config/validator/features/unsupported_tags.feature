Feature: Tags are rejected until selecting skills by tags is back

 Scenario Outline: A source with tags is an issue, one without is not
  When I validate configuration
   """
   sources:
     - {path: skills<tags>}
   """
  Then the config issues with codes "unsupported-tags" are
   """
   <issues>
   """
  Examples:
   | tags                       | issues |
   |                            | [] |
   | , tags: []                 | [] |
   | , tags: [go]               | [["unsupported-tags","local:/project/skills@master","sources[0].tags"]] |
   | , tags: [go, "!deprecated"] | [["unsupported-tags","local:/project/skills@master","sources[0].tags"]] |

 Scenario: The message says tags aren't supported yet
  When I validate configuration
   """
   sources:
     - {path: skills, tags: [go]}
   """
  Then config issue 0 message contains "not supported yet"

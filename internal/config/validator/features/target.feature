Feature: No two targets write into the same folder

 Scenario Outline: Two targets
  When I validate configuration
   """
   target:
     one: {path: <one>}
     two: {path: <two>}
   """
  Then the config issues are
   """
   <issues>
   """
  Examples:
   | one      | two          | issues |
   | out      | out2         | [] |
   | out      | out          | [["target-overlap","","targets[1].path"]] |
   | out      | out/nested   | [["target-overlap","","targets[1].path"]] |
   | out/in   | out          | [["target-overlap","","targets[1].path"]] |
   | /        | /project/out | [["target-overlap","","targets[1].path"]] |

 Scenario: The message names both targets
  When I validate configuration
   """
   target:
     one: {path: out}
     two: {path: out/nested}
   """
  Then config issue 0 message contains "target \"two\" path /project/out/nested overlaps target \"one\" path /project/out"

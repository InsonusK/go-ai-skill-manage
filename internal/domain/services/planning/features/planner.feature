Feature: Plan decides what a sync does with each skill folder of a target

 Scenario: A missing folder is created, a folder with the marker is updated, a marked orphan is removed
  Given the skills to write are "guide,review"
  And the target holds
   | name  | marker |
   | guide | yes    |
   | old   | yes    |
   | zeta  | yes    |
   | mine  | no     |
  And orphans are removed
  When I plan the target
  Then the operations are
   """
   [["update","guide","guide"],["create","review","review"],["remove","old",""],["remove","zeta",""]]
   """
  And the plan issues are
   """
   []
   """

 Scenario: Orphans are kept unless asked to remove them
  Given the skills to write are "guide"
  And the target holds
   | name  | marker |
   | old   | yes    |
  And orphans are kept
  When I plan the target
  Then the operations are
   """
   [["create","guide","guide"]]
   """

 Scenario: A folder of a skill's name that this tool didn't write is an issue, the rest is still planned
  Given the skills to write are "guide,review"
  And the target holds
   | name   | marker |
   | review | no     |
   | old    | yes    |
  And orphans are removed
  When I plan the target
  Then the operations are
   """
   [["create","guide","guide"],["remove","old",""]]
   """
  And the plan issues are
   """
   [["unmanaged-target","/p/.claude/skills","review"]]
   """

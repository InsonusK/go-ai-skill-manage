Feature: Synchronization orchestration
 Scenario Outline: Dry run and validation gate all writes
  Given sync source content "<content>" and dry run "<dry>"
  When I synchronize
  Then writer calls equal "<writes>"
  And sync error contains "<error>"
  Examples:
   | content | dry | writes | error |
   | valid | false | 1 | |
   | valid | true | 0 | |
   | broken | false | 0 | missing-link |
 Scenario: All target plans are validated before any write
  Given sync source content "valid" and dry run "false"
  And a second target fails state loading
  When I synchronize
  Then writer calls equal "0"
  And sync error contains "state unavailable"

 Scenario: Configured temporary directory reaches source acquisition
  Given sync source content "valid" and dry run "true"
  And request temporary directory is "/project/.tmp"
  When I synchronize
  Then source acquisition temp dir equals "/project/.tmp"

 Scenario: A failing skill detector blocks writes without real discovery
  Given sync source content "valid" and dry run "false"
  And skill detection always fails with "boom"
  When I synchronize
  Then writer calls equal "0"
  And sync error contains "boom"

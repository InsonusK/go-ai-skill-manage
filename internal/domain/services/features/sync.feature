Feature: Synchronization orchestration
 Scenario Outline: Dry run and validation gate all writes
  Given sync source content "<content>" and dry run "<dry>"
  When I synchronize
  Then writer calls equal "<writes>" and cleanup calls equal "1"
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
  Then writer calls equal "0" and cleanup calls equal "1"
  And sync error contains "state unavailable"

 Scenario: Configured temporary directory reaches source acquisition
  Given sync source content "valid" and dry run "true"
  And request temporary directory is "/project/.tmp"
  When I synchronize
  Then source acquisition temp dir equals "/project/.tmp"

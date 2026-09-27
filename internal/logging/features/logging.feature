Feature: Structured logging levels
 Scenario Outline: Debug is opt-in
  When I log with debug "<debug>"
  Then log contains debug "<present>" and info "true"
  Examples:
   | debug | present |
   | false | false |
   | true | true |

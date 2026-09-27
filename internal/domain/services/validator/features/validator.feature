Feature: Manager runs validators in their registered order and collects every issue

 Scenario: Every validator runs, even after an earlier one found issues, and all issues are returned in order
  Given fake validator "a" depending on "" (optional) reports "a1,a2"
  And fake validator "b" depending on "a" (required) reports "b1"
  And fake validator "c" depending on "" (optional) reports ""
  When I register validators "a,b,c"
  Then registration succeeds
  When I run the registered validators
  Then the validators ran in order "a,b,c"
  And the issue codes are "a1,a2,b1"

 Scenario Outline: A registration order that breaks a dependency is refused
  Given fake validator "a" depending on "" (optional) reports ""
  And fake validator "b" depending on "a" (<kind>) reports ""
  When I register validators "<order>"
  Then registration fails with "<error>"
  Examples:
   | kind     | order | error                                                        |
   | required | b     | validator "b" requires "a", which is not registered          |
   | required | b,a   | validator "a" must be registered before "b", which depends on it |
   | optional | b,a   | validator "a" must be registered before "b", which depends on it |
   | optional | a,a   | validator "a" is registered twice                            |

 Scenario Outline: A registration order that keeps every dependency is accepted
  Given fake validator "a" depending on "" (optional) reports ""
  And fake validator "b" depending on "a" (<kind>) reports ""
  When I register validators "<order>"
  Then registration succeeds
  Examples:
   | kind     | order |
   | required | a,b   |
   | optional | a,b   |
   | optional | b     |

 Scenario: Every broken dependency is reported at once
  Given fake validator "a" depending on "" (optional) reports ""
  And fake validator "b" depending on "a,c" (required) reports ""
  When I register validators "b,a"
  Then registration fails with "validator "a" must be registered before "b""
  And registration fails with "validator "b" requires "c", which is not registered"

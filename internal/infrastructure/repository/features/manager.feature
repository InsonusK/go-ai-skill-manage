Feature: Cache and dispatch source acquisitions
 Scenario: Acquiring the same source twice reuses the cached repository
  Given a source manager with a counting local provider
  When I acquire "local" source "same" twice
  Then the provider was called "1" times
  And both acquisitions returned the same repository

 Scenario: Acquiring different sources calls the provider for each
  Given a source manager with a counting local provider
  When I acquire "local" source "a" and "local" source "b"
  Then the provider was called "2" times

 Scenario: Unknown source type is rejected
  Given a source manager with a counting local provider
  When I acquire "unknown" source "x"
  Then acquiring fails with "unknown source type"

 Scenario: Close runs every acquired repository's Close in reverse order
  Given a source manager with a counting local provider
  When I acquire "local" source "a" and "local" source "b"
  And I close the source manager
  Then repositories were closed in order "b,a"

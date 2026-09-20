Feature: Index acquired repositories by source key
 Scenario: Registering and looking up repositories
  Given a source map
  When I put repository "a" and repository "b"
  Then repositories in order are "a,b"
  And repository "a" root is "/root-a"
  And repository "missing" is not found

 Scenario: Re-registering a source key updates it without reordering
  Given a source map
  When I put repository "a" and repository "b"
  And I put repository "a" again with root "/root-a2"
  Then repositories in order are "a,b"
  And repository "a" root is "/root-a2"

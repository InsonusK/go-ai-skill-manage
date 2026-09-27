Feature: Obtain source trees
 Scenario: Local source paths remain relative to source root
  Given a repository source
  When I acquire the local source
  Then acquired file "skills/a.skill.md" equals "content"
 Scenario: Failed clone falls back to GitHub archive
  Given a repository source
  When I fetch GitHub with clone failure "true"
  Then acquired file "skills/a.skill.md" equals "content"
  And archive calls equal "1"
 Scenario: Successful clone needs no archive
  Given a repository source
  When I fetch GitHub with clone failure "false"
  Then acquired file "skills/a.skill.md" equals "content"
  And archive calls equal "0"
 Scenario: GitHub workspace uses requested temporary directory
  Given a repository source
  When I fetch GitHub in the configured temporary directory
  Then acquired repository root matches "aism-source-*/repo" below temporary directory
 Scenario: Archive traversal is rejected
  Given a repository source
  When I extract an archive with path "../escape"
  Then repository error contains "unsafe archive"
 Scenario: Archive response is extracted
  Given a repository source
  When I extract an archive with path "repo/skills/a.skill.md"
  Then acquired file "skills/a.skill.md" equals "content"

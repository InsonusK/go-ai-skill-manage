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
  And the acquired commit is "archive-commit"
 Scenario: Successful clone needs no archive
  Given a repository source
  When I fetch GitHub with clone failure "false"
  Then acquired file "skills/a.skill.md" equals "content"
  And archive calls equal "0"
  And the acquired commit is "clone-commit"
 Scenario: Local source has no commit
  Given a repository source
  When I acquire the local source
  Then the acquired commit is ""
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
  And the archive commit is ""
 Scenario: GitHub archive names its commit in the pax global header
  Given a repository source
  When I extract an archive with path "repo/skills/a.skill.md" of commit "9a9ca973b37c904ff6e547b911f523177d664072"
  Then acquired file "skills/a.skill.md" equals "content"
  And the archive commit is "9a9ca973b37c904ff6e547b911f523177d664072"
 Scenario: A global header comment that is not a commit is ignored
  Given a repository source
  When I extract an archive with path "repo/skills/a.skill.md" of commit "not a commit"
  Then acquired file "skills/a.skill.md" equals "content"
  And the archive commit is ""

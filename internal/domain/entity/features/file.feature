Feature: Access a skill file through the File contract
 Scenario: Content reads a file once and caches its bytes
  Given a file "docs/intro.md" containing "intro" attached to a mocked skill rooted at "a" in repository "/source"
  When I read the contract file content
  Then the contract file content is "intro"
  And the contract file was read 1 time
  When I read the contract file content
  Then the contract file content is "intro"
  And the contract file was read 1 time

 Scenario: Path returns a path relative to the owning skill
  Given a file "docs/intro.md" containing "intro" attached to a mocked skill rooted at "a" in repository "/source"
  When I request the contract file path as "skill-relative"
  Then the requested file path is "docs/intro.md"

 Scenario: Path returns a path relative to the repository
  Given a file "docs/intro.md" containing "intro" attached to a mocked skill rooted at "a" in repository "/source"
  When I request the contract file path as "repo-relative"
  Then the requested file path is "a/docs/intro.md"

 Scenario: Path returns an absolute filesystem path
  Given a file "docs/intro.md" containing "intro" attached to a mocked skill rooted at "a" in repository "/source"
  When I request the contract file path as "absolute"
  Then the requested file path is "/source/a/docs/intro.md"

 Scenario: Path rejects an unknown path kind
  Given a file "docs/intro.md" containing "intro" attached to a mocked skill rooted at "a" in repository "/source"
  When I request the contract file path as "unknown"
  Then requesting the file path fails with "unknown file path kind"

 Scenario: Skill returns the file's owning skill
  Given a file "docs/intro.md" containing "intro" attached to a mocked skill rooted at "a" in repository "/source"
  Then the contract file belongs to the mocked skill
  
 #TODO: Implement the following scenario when the LinkSearcher is implemented
 #Scenario: Links returns the file's discovered links
 # Given a file "docs/intro.md" containing "intro" attached to a mocked skill rooted at "a" in repository "/source"
 # And the contract file has link target "repo\u0000other/SKILL.md"
 # Then the contract file link targets are "repo\u0000other/SKILL.md"

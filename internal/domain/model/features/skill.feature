Feature: Lazily load a skill's nested file content
 Scenario: FileData reads a nested file once and caches it
  Given a skill "guide" rooted at "a" main "a/SKILL.md" with nested file "notes.md" containing "hello"
  Then nested file "notes.md" data is not yet loaded
  When I read nested file "notes.md" data
  Then nested file "notes.md" data is "hello"
  And nested file "notes.md" was read "1" time
  When I read nested file "notes.md" data again
  Then nested file "notes.md" data is "hello"
  And nested file "notes.md" was read "1" time

 Scenario: FileData surfaces the read error for a missing nested file
  Given a skill "guide" rooted at "a" main "a/SKILL.md" with a missing nested file "gone.md"
  When I read nested file "gone.md" data
  Then reading nested file data fails with "file does not exist"

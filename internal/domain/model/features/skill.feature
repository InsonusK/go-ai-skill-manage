Feature: Lazily load a skill's nested file content
 Scenario: FileData reads a nested file once and caches it
  Given a skill "guide" rooted at "a" main "a/SKILL.md" with nested file "notes.md" containing "hello"
  Then nested file "notes.md" data is not yet loaded
  Then nested file "notes.md" was read "0" time
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

 Scenario: FilesByPath lists everything under the skill root
  Given a skill "guide" rooted at "a" main "a/SKILL.md" with tree
   """
   {"docs/intro.md":"intro","examples/big.txt":"big"}
   """
  When I list files by path ""
  Then listed files are "docs/intro.md,examples/big.txt"

 Scenario: FilesByPath scoped to a sub-path only sees files under it
  Given a skill "guide" rooted at "a" main "a/SKILL.md" with tree
   """
   {"docs/intro.md":"intro","examples/big.txt":"big"}
   """
  When I list files by path "docs"
  Then listed files are "docs/intro.md"

 Scenario: FilesByPath caches per requested path and does not re-walk
  Given a skill "guide" rooted at "a" main "a/SKILL.md" with tree
   """
   {"docs/intro.md":"intro"}
   """
  When I list files by path "docs"
  Then total directory reads so far are remembered
  When I list files by path "docs" again
  Then no additional directories were read

 Scenario: Find filters listed files by regex within the scoped subtree
  Given a skill "guide" rooted at "a" main "a/SKILL.md" with tree
   """
   {"docs/intro.md":"intro","docs/notes.txt":"notes"}
   """
  When I find files by path "docs" matching "\.md$"
  Then listed files are "docs/intro.md"

 Scenario: Data reads and caches a file's content by skill-relative path
  Given a skill "guide" rooted at "a" main "a/SKILL.md" with tree
   """
   {"docs/intro.md":"intro"}
   """
  When I read file data at "docs/intro.md"
  Then file data at "docs/intro.md" is "intro"

 Scenario: FilesByPath lists a nested skill marker file like any other file
  Given a skill "guide" rooted at "a" main "a/SKILL.md" with tree
   """
   {"docs/intro.md":"intro","examples/SKILL.md":"---\nname: nested\n---\n"}
   """
  When I list files by path "examples"
  Then listed files are "examples/SKILL.md"

 Scenario: A file outside the scoped subtree is not seen
  Given a skill "guide" rooted at "a" main "a/SKILL.md" with tree
   """
   {"docs/intro.md":"intro","examples/SKILL.md":"---\nname: nested\n---\n"}
   """
  When I list files by path "docs"
  Then listed files are "docs/intro.md"

 Scenario: Data loaded through Find stays cached on a later FilesByPath with the same scope
  Given a skill "guide" rooted at "a" main "a/SKILL.md" with tree
   """
   {"docs/intro.md":"intro"}
   """
  When I find files by path "" matching "\.md$"
  And I read data of the first found file
  Then the first found file's data is "intro"
  When I list files by path ""
  Then the first listed file's data is "intro"
  And nested file "docs/intro.md" was read "1" time

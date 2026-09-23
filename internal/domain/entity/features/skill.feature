Feature: Access a skill through the Skill contract

 Scenario: Key combines repository and main-file identity
  Given an "agent-dir" skill "guide" rooted at "skills/guide" with main file "skills/guide/SKILL.md" in repository "/source" and files
   """
   {"skills/guide/SKILL.md":"---\nname: guide\n---\nmain"}
   """
  When I request the skill key
  Then the skill key is
   """
   "repo\u0000skills/guide/SKILL.md"
   """

 Scenario: Metadata returns the skill metadata
  Given an "agent-dir" skill "guide" rooted at "skills/guide" with main file "skills/guide/SKILL.md" in repository "/source" and files
   """
   {"skills/guide/SKILL.md":"---\nname: guide\n---\nmain"}
   """
  When I request the skill metadata
  Then the skill metadata is
   """
   {"name":"guide","mainFilePath":"skills/guide/SKILL.md","skillDirPath":"skills/guide","format":"agent-dir","repositoryId":"repo","repositoryRoot":"/source"}
   """

 Scenario: FilesByPath lists every nested file under the skill root
  Given an "agent-dir" skill "guide" rooted at "skills/guide" with main file "skills/guide/SKILL.md" in repository "/source" and files
   """
   {"skills/guide/SKILL.md":"---\nname: guide\n---\nmain","skills/guide/docs/intro.md":"intro","skills/guide/examples/demo.txt":"demo"}
   """
  When I list skill files at path ""
  Then the listed skill-relative paths are
   """
   ["docs/intro.md","examples/demo.txt"]
   """

 Scenario: FilesByPath limits traversal to the requested subtree
  Given an "agent-dir" skill "guide" rooted at "skills/guide" with main file "skills/guide/SKILL.md" in repository "/source" and files
   """
   {"skills/guide/SKILL.md":"---\nname: guide\n---\nmain","skills/guide/docs/intro.md":"intro","skills/guide/examples/demo.txt":"demo"}
   """
  When I list skill files at path "docs"
  Then the listed skill-relative paths are
   """
   ["docs/intro.md"]
   """

 Scenario: FilesByPath returns immediately for a flat skill
  Given a "flat" skill "guide" rooted at "" with main file "guide.skill.md" in repository "/source" and files
   """
   {"guide.skill.md":"---\nname: guide\n---\nmain","unrelated.md":"other"}
   """
  When I list skill files at path "anything"
  Then the listed skill-relative paths are
   """
   []
   """
  And the repository open count is 1

 Scenario: FilesByPath skips Git metadata
  Given an "agent-dir" skill "guide" rooted at "skills/guide" with main file "skills/guide/SKILL.md" in repository "/source" and files
   """
   {"skills/guide/SKILL.md":"---\nname: guide\n---\nmain","skills/guide/docs/intro.md":"intro","skills/guide/.git/config":"secret"}
   """
  When I list skill files at path ""
  Then the listed skill-relative paths are
   """
   ["docs/intro.md"]
   """

 Scenario: FilesByPath handles a dot skill root
  Given an "agent-dir" skill "guide" rooted at "." with main file "SKILL.md" in repository "/source" and files
   """
   {"SKILL.md":"---\nname: guide\n---\nmain","docs/intro.md":"intro"}
   """
  When I list skill files at path ""
  Then the listed skill-relative paths are
   """
   ["docs/intro.md"]
   """

 Scenario: FilesByPath caches each requested path
  Given an "agent-dir" skill "guide" rooted at "skills/guide" with main file "skills/guide/SKILL.md" in repository "/source" and files
   """
   {"skills/guide/SKILL.md":"---\nname: guide\n---\nmain","skills/guide/docs/intro.md":"intro"}
   """
  When I list skill files at path "docs"
  And I remember the repository open count
  When I list skill files at path "docs"
  Then the repository open count has not increased

 Scenario: FilesByPath caches distinct paths independently
  Given an "agent-dir" skill "guide" rooted at "skills/guide" with main file "skills/guide/SKILL.md" in repository "/source" and files
   """
   {"skills/guide/SKILL.md":"---\nname: guide\n---\nmain","skills/guide/docs/intro.md":"intro","skills/guide/examples/demo.txt":"demo"}
   """
  When I list skill files at path "docs"
  Then the listed skill-relative paths are
   """
   ["docs/intro.md"]
   """
  When I list skill files at path "examples"
  Then the listed skill-relative paths are
   """
   ["examples/demo.txt"]
   """
  And I remember the repository open count
  When I list skill files at path "docs"
  Then the listed skill-relative paths are
   """
   ["docs/intro.md"]
   """
  And the repository open count has not increased

 Scenario: FilesByPath treats nested markers as ordinary files
  Given an "agent-dir" skill "guide" rooted at "skills/guide" with main file "skills/guide/SKILL.md" in repository "/source" and files
   """
   {"skills/guide/SKILL.md":"---\nname: guide\n---\nmain","skills/guide/examples/SKILL.md":"nested","skills/guide/examples/legacy.skill.md":"nested"}
   """
  When I list skill files at path "examples"
  Then the listed skill-relative paths are
   """
   ["examples/SKILL.md","examples/legacy.skill.md"]
   """

 Scenario: FilesByPath filters cached paths by regular expression
  Given an "agent-dir" skill "guide" rooted at "skills/guide" with main file "skills/guide/SKILL.md" in repository "/source" and files
   """
   {"skills/guide/SKILL.md":"---\nname: guide\n---\nmain","skills/guide/docs/intro.md":"intro","skills/guide/docs/notes.txt":"notes"}
   """
  When I list skill files at path "docs"
  Then the listed skill-relative paths are
   """
   ["docs/intro.md","docs/notes.txt"]
   """
  And I remember the repository open count
  When I list skill files at path "docs" matching "\.md$"
  Then the listed skill-relative paths are
   """
   ["docs/intro.md"]
   """
  And the repository open count has not increased

 Scenario: FilesByPath surfaces traversal errors
  Given an "agent-dir" skill "guide" rooted at "skills/guide" with main file "skills/guide/SKILL.md" in repository "/source" and files
   """
   {"skills/guide/SKILL.md":"---\nname: guide\n---\nmain"}
   """
  When I list skill files at path "missing"
  Then listing skill files fails with filesystem error "not-exist"

 Scenario: FilesByPath allows paths outside the skill root but inside the repository
  Given an "agent-dir" skill "guide" rooted at "skills/guide" with main file "skills/guide/SKILL.md" in repository "/source" and files
   """
   {"skills/guide/SKILL.md":"---\nname: guide\n---\nmain","shared/reference.md":"reference"}
   """
  When I list skill files at path "../../shared"
  Then the listed skill-relative paths are
   """
   ["../../shared/reference.md"]
   """

 Scenario: FilesByPath rejects paths outside the repository root
  Given an "agent-dir" skill "guide" rooted at "skills/guide" with main file "skills/guide/SKILL.md" in repository "/source" and files
   """
   {"skills/guide/SKILL.md":"---\nname: guide\n---\nmain"}
   """
  When I list skill files at path "../../../outside"
  Then listing skill files fails with filesystem error "invalid-path"

 Scenario: Construction fails when the main file has no name property
  Given an "agent-dir" skill "guide" rooted at "skills/guide" with main file "skills/guide/SKILL.md" in repository "/source" and files
   """
   {"skills/guide/SKILL.md":"main"}
   """
  Then building the skill fails with "skill name is missing"

Feature: Store writes a plan into a target folder and reads what the folder holds

 # The target is listed as path from the target folder -> content.

 Scenario: A created skill folder has every file of the skill, the marker included, with their modes
  Given an empty target
  And a skill to write with files
   """
   {"SKILL.md":"---\nname: sample\n---\n","docs/x.md":"x","run.sh":"echo"}
   """
  And file "run.sh" of the skill to write has mode "775"
  When I apply "create" for "sample"
  Then applying succeeds
  And the target holds
   """
   {"sample/SKILL.md":"---\nname: sample\n---\n","sample/docs/x.md":"x","sample/run.sh":"echo","sample/.ai-skills-managed":"{}\n"}
   """
  And target file "sample/run.sh" has mode "775"
  And target file "sample/docs/x.md" has mode "644"

 Scenario: An update replaces the folder as a whole, files the skill no longer has go away
  Given an empty target
  And a skill to write with files
   """
   {"SKILL.md":"---\nname: sample\n---\nnew\n"}
   """
  And target fixture file "sample/.ai-skills-managed" contains "{}"
  And target fixture file "sample/obsolete.md" contains "stale"
  When I apply "update" for "sample"
  Then applying succeeds
  And the target holds
   """
   {"sample/SKILL.md":"---\nname: sample\n---\nnew\n","sample/.ai-skills-managed":"{}\n"}
   """

 Scenario: A managed folder is removed, one without the marker is not
  Given an empty target
  And target fixture file "old/.ai-skills-managed" contains "{}"
  And unmanaged target directory "manual"
  When I apply "remove" for "old"
  Then applying succeeds
  And target path "old" exists "false"
  When I apply "remove" for "manual"
  Then applying fails with "unmanaged target manual"
  And target path "manual" exists "true"

 Scenario Outline: A folder this tool didn't write is never replaced, whatever the plan says
  Given an empty target
  And a skill to write with files
   """
   {"SKILL.md":"---\nname: sample\n---\n"}
   """
  And target fixture file "sample/mine.md" contains "personal"
  When I apply "<action>" for "sample"
  Then applying fails with "unmanaged target sample"
  And the target holds
   """
   {"sample/mine.md":"personal"}
   """
  Examples:
   | action |
   | create |
   | update |

 Scenario Outline: A plan is checked before anything is written
  Given an empty target
  And <skill> skill to write with files
   """
   {"SKILL.md":"---\nname: sample\n---\n"}
   """
  When I apply "create" for "<name>"
  Then applying fails with "<error>"
  And the target holds
   """
   {}
   """
  Examples:
   | skill       | name      | error                     |
   | a           | ../escape | unsafe or duplicate       |
   | a           | a/b       | unsafe or duplicate       |
   | an unmarked | sample    | has no .ai-skills-managed |

 Scenario: A target folder reached through a symlink is refused
  Given an empty target
  And a skill to write with files
   """
   {"SKILL.md":"---\nname: sample\n---\n"}
   """
  When I apply "create" for "sample" into a target reached through a symlink
  Then applying fails with "unsafe target symlink"

 Scenario: The snapshot tells every entry apart: a managed folder, any other folder, a file, a symlink, a folder whose marker isn't a file
  Given an empty target
  And target fixture file "managed/.ai-skills-managed" contains "{}"
  And unmanaged target directory "manual"
  And target fixture file "file" contains "x"
  And target symlink "link" points to "managed"
  And unmanaged target directory "fake/.ai-skills-managed"
  When I take a snapshot of the target
  Then the snapshot is
   """
   {"managed":[true,true],"manual":[true,false],"file":[true,false],"link":[true,false],"fake":[true,false]}
   """

 Scenario: A target folder that doesn't exist yet holds nothing
  Given an empty target
  When I take a snapshot of the target folder that doesn't exist
  Then the snapshot is
   """
   {}
   """

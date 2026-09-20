Feature: Apply managed changes
 Scenario: Created skills receive state and skipped copies remain untouched
  Given an empty target
  When I apply a "create" operation for "sample" containing "body"
  Then target file "sample/SKILL.md" equals "body"
  And state for "sample" is managed with hash "hash"
  When I apply a "skip" operation for "sample" containing "changed"
  Then target file "sample/SKILL.md" equals "body"
 Scenario: Update replaces obsolete managed files
  Given an empty target
  When I apply a "create" operation for "sample" containing "old"
  And target fixture file "sample/obsolete.txt" contains "stale"
  And I apply a "update" operation for "sample" containing "new"
  Then target file "sample/SKILL.md" equals "new"
  And target path "sample/obsolete.txt" exists "false"
 Scenario: Removal only deletes managed directories
  Given an empty target
  When I apply a "create" operation for "sample" containing "body"
  And I apply a "remove" operation for "sample" containing ""
  Then target path "sample" exists "false"
 Scenario: Unmanaged directories cannot be removed
  Given an empty target
  And unmanaged target directory "manual"
  When I apply a "remove" operation for "manual" containing ""
  Then filesystem error contains "unmanaged"
  And target path "manual" exists "true"
 Scenario: Escaping output paths are rejected before writing
  Given an empty target
  When I apply a "create" operation for "../escape" containing "body"
  Then filesystem error contains "unsafe"

 Scenario: Existing files cannot be planned as new skills
  Given an empty target
  And target fixture file "sample" contains "personal"
  Then snapshot marks "sample" as existing unmanaged entry
 Scenario: Symlinks cannot be planned as new skills
  Given an empty target
  And target symlink "sample" points to "missing"
  Then snapshot marks "sample" as existing unmanaged entry

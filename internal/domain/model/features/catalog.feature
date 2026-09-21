Feature: Grow and query a skill catalog
 Scenario: Re-adding the same skill is a no-op
  Given a catalog with conflict policy "error"
  When I add skill "a" from repo "repo" main "a/SKILL.md"
  And I add skill "a" from repo "repo" main "a/SKILL.md"
  Then catalog error contains ""
  And catalog skills are "a"

 Scenario: Duplicate name from a different source errors by default
  Given a catalog with conflict policy "error"
  When I add skill "a" from repo "repo" main "a/SKILL.md"
  And I add skill "a" from repo "other" main "a/SKILL.md"
  Then catalog error contains "duplicate-name"

 Scenario: last_wins replaces the earlier skill with the same name
  Given a catalog with conflict policy "last_wins"
  When I add skill "a" from repo "repo" main "a/SKILL.md"
  And I add skill "a" from repo "other" main "a/SKILL.md"
  Then catalog error contains ""
  And catalog skill "a" belongs to repo "other"

 Scenario: Owner finds the skill owning a path inside its root
  Given a catalog with conflict policy "error"
  When I add skill "a" from repo "repo" main "a/SKILL.md"
  Then catalog owner of "repo" path "a/guide.md" is "a"
  And catalog owner of "repo" path "b/guide.md" is not found
  And catalog owner of "other" path "a/guide.md" is not found

 Scenario: GetOrAdd indexes a skill's own and nested file destinations
  Given a catalog with conflict policy "error"
  When I add skill "guide" from repo "repo" main "a/SKILL.md" with files "notes.md"
  Then catalog destination of "repo" path "a/SKILL.md" is "guide" at "guide/SKILL.md"
  And catalog destination of "repo" path "a/notes.md" is "guide" at "guide/notes.md"
  And catalog destination of "repo" path "a" is "guide" at "guide/SKILL.md"
  And catalog destination of "repo" path "missing.md" is not found

 Scenario: last_wins re-indexes destinations onto the newer skill's source
  Given a catalog with conflict policy "last_wins"
  When I add skill "a" from repo "repo" main "a/SKILL.md"
  And I add skill "a" from repo "other" main "a/SKILL.md"
  Then catalog destination of "repo" path "a/SKILL.md" is not found
  And catalog destination of "other" path "a/SKILL.md" is "a" at "a/SKILL.md"

 Scenario: The directory root alias is actually indexed, not only fallback-resolved
  Given a catalog with conflict policy "error"
  When I add skill "guide" from repo "repo" main "a/SKILL.md" with files "notes.md"
  Then catalog destinations map has "repo" "a/SKILL.md" pointing to "guide/SKILL.md"
  And catalog destinations map has "repo" "a" pointing to "guide/SKILL.md"
  And catalog destinations map has "repo" "a/notes.md" pointing to "guide/notes.md"

 Scenario: last_wins removes the old skill's indexed root alias too
  Given a catalog with conflict policy "last_wins"
  When I add skill "a" from repo "repo" main "a/SKILL.md"
  And I add skill "a" from repo "other" main "a/SKILL.md"
  Then catalog destinations map has no entry for "repo" "a"
  And catalog destinations map has "other" "a" pointing to "a/SKILL.md"

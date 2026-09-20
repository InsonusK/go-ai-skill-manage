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

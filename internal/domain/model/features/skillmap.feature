Feature: Index the finalized catalog into a skill map
 Scenario: Build a skill map and resolve ownership
  Given skill entries
    """
    [
      {"name": "a", "source": "repo", "root": "a", "main": "a/SKILL.md", "flat": false},
      {"name": "b", "source": "repo", "root": "b.skill.md", "main": "b.skill.md", "flat": true}
    ]
    """
  And skill file destinations
    """
    {
      "a": {"a/SKILL.md": "a/SKILL.md", "a/guide.md": "a/guide.md", "a": "a/SKILL.md"},
      "b": {"b.skill.md": "b/SKILL.md"}
    }
    """
  When I build the skill map
  Then owner of "repo" path "a/guide.md" is skill "a" at "a/guide.md"
  And owner of "repo" path "a/extra.md" is skill "a" at "a/extra.md"
  And owner of "repo" path "missing.md" is not found

 Scenario: Colliding destinations across skills are rejected
  Given skill entries
    """
    [
      {"name": "a", "source": "repo", "root": "a", "main": "a/SKILL.md", "flat": false},
      {"name": "b", "source": "repo", "root": "b", "main": "b/SKILL.md", "flat": false}
    ]
    """
  And skill file destinations
    """
    {
      "a": {"a/SKILL.md": "shared/SKILL.md"},
      "b": {"b/SKILL.md": "shared/SKILL.md"}
    }
    """
  When I build the skill map
  Then building the skill map fails with "output-collision"

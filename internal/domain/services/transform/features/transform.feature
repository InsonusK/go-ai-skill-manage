Feature: Target-specific document transformations
 Scenario: References are rewritten to Markdown
  Given transformation document
   """
   Start [[old#part|Read]] ![image](old) end
   """
  When I rewrite links to ".agents/skills/new/SKILL.md"
  Then transformed text is
   """
   Start [Read](.agents/skills/new/SKILL.md#part) ![image](.agents/skills/new/SKILL.md) end
   """
 Scenario: Claude moves custom metadata and normalizes whenToUse
  Given transformation document
   """
   ---
   name: example
   description: Demo
   whenToUse: [one, two]
   tags: [go]
   ---
   Body
   """
  When I transform Claude properties
  Then transformed properties are
   """
   {"name":"example","description":"Demo","when_to_use":"one,two"}
   """
  And transformed text contains "## Metadata"
  And transformed text contains "tags:"
